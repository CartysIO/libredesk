-- name: get-overview-counts
WITH convs AS (
    SELECT
        COUNT(*) AS open,
        COUNT(*) FILTER (
            WHERE
                c.last_message_sender = 'contact'
        ) AS awaiting_response,
        COUNT(*) FILTER (
            WHERE
                c.assigned_user_id IS NULL
        ) AS unassigned,
        COUNT(*) FILTER (
            WHERE
                c.first_reply_at IS NULL
        ) AS pending
    FROM
        conversations c
        INNER JOIN conversation_statuses s ON c.status_id = s.id
    WHERE
        s.category != 'resolved'
),
agents AS (
    SELECT
        COUNT(*) FILTER (
            WHERE
                availability_status IN ('online', 'away')
        ) AS agents_online,
        COUNT(*) FILTER (
            WHERE
                availability_status = 'away_manual'
        ) AS agents_away,
        COUNT(*) FILTER (
            WHERE
                availability_status = 'away_and_reassigning'
        ) AS agents_reassigning,
        COUNT(*) FILTER (
            WHERE
                availability_status = 'offline'
        ) AS agents_offline
    FROM
        users
    WHERE
        type = 'agent'
        AND deleted_at IS NULL
)
SELECT
    json_build_object(
        'open', open,
        'awaiting_response', awaiting_response,
        'unassigned', unassigned,
        'pending', pending,
        'agents_online', agents_online,
        'agents_away', agents_away,
        'agents_reassigning', agents_reassigning,
        'agents_offline', agents_offline
    )
FROM
    convs,
    agents;

-- name: get-overview-sla-counts
-- Count only each conversation's latest applied SLA; superseded rows are kept as history and would double-count.
WITH latest_applied AS (
    SELECT DISTINCT ON (conversation_id)
        created_at, first_response_met_at, first_response_breached_at,
        resolution_met_at, resolution_breached_at
    FROM applied_slas
    WHERE created_at >= CASE
        WHEN %d = 0 THEN DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1
        ELSE NOW() - INTERVAL '%d days'
    END
    ORDER BY conversation_id, created_at DESC, id DESC
),
first_and_resolution AS (
    SELECT
        COUNT(*) FILTER (
            WHERE
                first_response_met_at IS NOT NULL
        ) AS first_response_met_count,
        COUNT(*) FILTER (
            WHERE
                first_response_breached_at IS NOT NULL
        ) AS first_response_breached_count,
        COUNT(*) FILTER (
            WHERE
                resolution_met_at IS NOT NULL
        ) AS resolution_met_count,
        COUNT(*) FILTER (
            WHERE
                resolution_breached_at IS NOT NULL
        ) AS resolution_breached_count,
        COALESCE(
            AVG(
                EXTRACT(
                    EPOCH
                    FROM
                        (first_response_met_at - created_at)
                )
            ) FILTER (
                WHERE
                    first_response_met_at IS NOT NULL
            ),
            0
        ) AS avg_first_response_time_sec,
        COALESCE(
            AVG(
                EXTRACT(
                    EPOCH
                    FROM
                        (resolution_met_at - created_at)
                )
            ) FILTER (
                WHERE
                    resolution_met_at IS NOT NULL
            ),
            0
        ) AS avg_resolution_time_sec
    FROM
        latest_applied
),
next_response AS (
    -- A reply after the deadline carries both met_at and breached_at, so counting the
    -- timestamps puts one event in both buckets. status holds a single terminal verdict.
    SELECT
        COUNT(*) FILTER (
            WHERE
                status = 'met'
        ) AS next_response_met_count,
        COUNT(*) FILTER (
            WHERE
                status = 'breached'
        ) AS next_response_breached_count,
        COALESCE(
            AVG(
                EXTRACT(
                    EPOCH
                    FROM
                        (met_at - created_at)
                )
            ) FILTER (
                WHERE
                    status = 'met'
            ),
            0
        ) AS avg_next_response_time_sec
    FROM
        sla_events
    WHERE
        created_at >= CASE
            WHEN %d = 0 THEN DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1
            ELSE NOW() - INTERVAL '%d days'
        END
        AND type = 'next_response'
)
SELECT
    fas.first_response_met_count,
    fas.first_response_breached_count,
    fas.avg_first_response_time_sec,
    nr.next_response_met_count,
    nr.next_response_breached_count,
    nr.avg_next_response_time_sec,
    fas.resolution_met_count,
    fas.resolution_breached_count,
    fas.avg_resolution_time_sec,
    CASE
        WHEN (fas.first_response_met_count + fas.first_response_breached_count) > 0
        THEN ROUND((fas.first_response_met_count::numeric / (fas.first_response_met_count + fas.first_response_breached_count)::numeric) * 100, 1)
        ELSE 0
    END AS first_response_compliance_percent,
    CASE
        WHEN (nr.next_response_met_count + nr.next_response_breached_count) > 0
        THEN ROUND((nr.next_response_met_count::numeric / (nr.next_response_met_count + nr.next_response_breached_count)::numeric) * 100, 1)
        ELSE 0
    END AS next_response_compliance_percent,
    CASE
        WHEN (fas.resolution_met_count + fas.resolution_breached_count) > 0
        THEN ROUND((fas.resolution_met_count::numeric / (fas.resolution_met_count + fas.resolution_breached_count)::numeric) * 100, 1)
        ELSE 0
    END AS resolution_compliance_percent
FROM
    first_and_resolution fas,
    next_response nr;

-- name: get-overview-charts
WITH new_conversations AS (
    SELECT
        json_agg(row_to_json(agg)) AS data
    FROM
        (
            SELECT
                TO_CHAR(created_at AT TIME ZONE $1, 'YYYY-MM-DD') AS date,
                COUNT(*) AS count
            FROM
                conversations c
            WHERE
                c.created_at >= CASE
                    WHEN %d = 0 THEN DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1
                    ELSE NOW() - INTERVAL '%d days'
                END
            GROUP BY
                date
            ORDER BY
                date
        ) agg
),
resolved_conversations AS (
    SELECT
        json_agg(row_to_json(agg)) AS data
    FROM
        (
            SELECT
                TO_CHAR(resolved_at AT TIME ZONE $1, 'YYYY-MM-DD') AS date,
                COUNT(*) AS count
            FROM
                conversations c
            WHERE
                c.resolved_at >= CASE
                    WHEN %d = 0 THEN DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1
                    ELSE NOW() - INTERVAL '%d days'
                END
            GROUP BY
                date
            ORDER BY
                date
        ) agg
)
SELECT
    json_build_object(
        'new_conversations',
        (
            SELECT
                data
            FROM
                new_conversations
        ),
        'resolved_conversations',
        (
            SELECT
                data
            FROM
                resolved_conversations
        )
    ) AS result;

-- name: get-overview-csat
SELECT
    json_build_object(
        'average_rating',
        COALESCE(AVG(rating) FILTER (WHERE rating > 0), 0),
        'total_responses',
        COUNT(*) FILTER (WHERE rating > 0),
        'total_sent',
        COUNT(*),
        'response_rate',
        CASE
            WHEN COUNT(*) > 0
            THEN ROUND((COUNT(*) FILTER (WHERE rating > 0)::numeric / COUNT(*)::numeric) * 100, 1)
            ELSE 0
        END
    ) AS result
FROM
    csat_responses
WHERE
    created_at >= CASE
        WHEN %d = 0 THEN DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1
        ELSE NOW() - INTERVAL '%d days'
    END;

-- name: get-overview-message-volume
WITH per_conversation AS (
    SELECT
        conversation_id,
        COUNT(*) AS total,
        COUNT(*) FILTER (WHERE type = 'incoming') AS incoming,
        COUNT(*) FILTER (WHERE type = 'outgoing') AS outgoing
    FROM
        conversation_messages
    WHERE
        type IN ('incoming', 'outgoing')
        AND private = false
        AND (type = 'incoming' OR status = 'sent')
        AND created_at >= CASE
            WHEN %d = 0 THEN DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1
            ELSE NOW() - INTERVAL '%d days'
        END
    GROUP BY
        conversation_id
),
stats AS (
    SELECT
        COALESCE(SUM(total), 0) AS total,
        COALESCE(SUM(incoming), 0) AS incoming,
        COALESCE(SUM(outgoing), 0) AS outgoing,
        COUNT(*) AS convos
    FROM
        per_conversation
)
SELECT
    json_build_object(
        'total_messages', total,
        'incoming_messages', incoming,
        'outgoing_messages', outgoing,
        'messages_per_conversation',
        CASE
            WHEN convos > 0 THEN ROUND(total::numeric / convos::numeric, 1)
            ELSE 0
        END
    ) AS result
FROM
    stats;

-- name: get-overview-tag-distribution
WITH tag_counts AS (
    SELECT
        t.id AS tag_id,
        t.name AS tag_name,
        COUNT(c.id) AS count
    FROM
        tags t
        LEFT JOIN conversation_tags ct ON t.id = ct.tag_id
        LEFT JOIN conversations c ON ct.conversation_id = c.id
            AND c.created_at >= CASE
                WHEN %d = 0 THEN DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1
                ELSE NOW() - INTERVAL '%d days'
            END
    GROUP BY
        t.id, t.name
    ORDER BY
        count DESC, t.id
    LIMIT 10
),
tagging AS (
    SELECT
        COUNT(*) AS total,
        COUNT(*) FILTER (
            WHERE EXISTS (
                SELECT 1 FROM conversation_tags ct
                WHERE ct.conversation_id = c.id
            )
        ) AS tagged
    FROM
        conversations c
    WHERE
        c.created_at >= CASE
            WHEN %d = 0 THEN DATE_TRUNC('day', NOW() AT TIME ZONE $1) AT TIME ZONE $1
            ELSE NOW() - INTERVAL '%d days'
        END
)
SELECT
    json_build_object(
        'top_tags',
        COALESCE((SELECT json_agg(row_to_json(tc)) FROM tag_counts tc), '[]'::json),
        'tagged_conversations', tagged,
        'untagged_conversations', total - tagged,
        'tagged_percentage',
        CASE
            WHEN total > 0
            THEN ROUND((tagged::numeric / total::numeric) * 100, 1)
            ELSE 0
        END
    ) AS result
FROM
    tagging;

-- name: get-issues-scalars
SELECT
    json_build_object(
        'all_time_accounts',
        (
            SELECT
                COUNT(*)
            FROM
                organizations
        ),
        'all_time_issues',
        (
            SELECT
                COUNT(*)
            FROM
                conversations
        ),
        'issues_created',
        (
            SELECT
                COUNT(*)
            FROM
                conversations c
            WHERE
                c.created_at >= $1::timestamptz
                AND c.created_at < $2::timestamptz
        ),
        'issues_solved',
        (
            SELECT
                COUNT(*)
            FROM
                conversations c
            WHERE
                c.resolved_at IS NOT NULL
                AND c.resolved_at >= $1::timestamptz
                AND c.resolved_at < $2::timestamptz
        ),
        'issues_unsolved',
        (
            SELECT
                COUNT(*)
            FROM
                conversations c
            WHERE
                c.created_at >= $1::timestamptz
                AND c.created_at < $2::timestamptz
                AND c.resolved_at IS NULL
        ),
        'reopened',
        (
            SELECT
                COUNT(DISTINCT src.conversation_id)
            FROM
                (
                    SELECT
                        un.conversation_id
                    FROM
                        user_notifications un
                    WHERE
                        un.notification_type = 'conversation_reopened'
                        AND un.created_at >= $1::timestamptz
                        AND un.created_at < $2::timestamptz
                        AND un.conversation_id IS NOT NULL
                    UNION
                    SELECT
                        m.conversation_id
                    FROM
                        conversation_messages m
                        INNER JOIN conversations c ON c.id = m.conversation_id
                        INNER JOIN conversation_statuses s ON s.name = COALESCE(
                            NULLIF(TRIM(m.meta->>'new_value'), ''),
                            NULLIF(
                                TRIM(
                                    (regexp_match(m.content, 'marked the conversation as (.+)$'))[1]
                                ),
                                ''
                            )
                        )
                    WHERE
                        m.type = 'activity'
                        AND m.meta->>'activity_type' = 'status_change'
                        AND (
                            NULLIF(TRIM(m.meta->>'new_value'), '') IS NOT NULL
                            OR m.content ~ 'marked the conversation as '
                        )
                        AND m.created_at >= $1::timestamptz
                        AND m.created_at < $2::timestamptz
                        AND c.last_resolved_at IS NOT NULL
                        AND m.created_at > c.last_resolved_at
                        AND s.category != 'resolved'
                ) src
        )
    ) AS result;

-- name: get-issues-charts
WITH range_bounds AS (
    SELECT
        $2::timestamptz AS starts_at,
        $3::timestamptz AS ends_at
),
volume_by_day AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.date)
                FROM
                    (
                        SELECT
                            TO_CHAR(c.created_at AT TIME ZONE $1, 'YYYY-MM-DD') AS date,
                            COUNT(*) AS count
                        FROM
                            conversations c,
                            range_bounds rb
                        WHERE
                            c.created_at >= rb.starts_at
                            AND c.created_at < rb.ends_at
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
volume_over_time AS (
    SELECT
        json_build_object(
            'created',
            COALESCE(
                (
                    SELECT
                        json_agg(row_to_json(agg) ORDER BY agg.date)
                    FROM
                        (
                            SELECT
                                TO_CHAR(c.created_at AT TIME ZONE $1, 'YYYY-MM-DD') AS date,
                                COUNT(*) AS count
                            FROM
                                conversations c,
                                range_bounds rb
                            WHERE
                                c.created_at >= rb.starts_at
                            AND c.created_at < rb.ends_at
                            GROUP BY
                                1
                        ) agg
                ),
                '[]'::json
            ),
            'closed',
            COALESCE(
                (
                    SELECT
                        json_agg(row_to_json(agg) ORDER BY agg.date)
                    FROM
                        (
                            SELECT
                                TO_CHAR(c.resolved_at AT TIME ZONE $1, 'YYYY-MM-DD') AS date,
                                COUNT(*) AS count
                            FROM
                                conversations c,
                                range_bounds rb
                            WHERE
                                c.resolved_at IS NOT NULL
                                AND c.resolved_at >= rb.starts_at
                            AND c.resolved_at < rb.ends_at
                            GROUP BY
                                1
                        ) agg
                ),
                '[]'::json
            )
        ) AS data
),
by_hour AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.hour)
                FROM
                    (
                        SELECT
                            EXTRACT(
                                HOUR
                                FROM
                                    c.created_at AT TIME ZONE $1
                            )::int AS hour,
                            COUNT(*) AS count
                        FROM
                            conversations c,
                            range_bounds rb
                        WHERE
                            c.created_at >= rb.starts_at
                            AND c.created_at < rb.ends_at
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
by_dow AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.dow)
                FROM
                    (
                        SELECT
                            EXTRACT(
                                ISODOW
                                FROM
                                    c.created_at AT TIME ZONE $1
                            )::int AS dow,
                            TO_CHAR(c.created_at AT TIME ZONE $1, 'Dy') AS label,
                            COUNT(*) AS count
                        FROM
                            conversations c,
                            range_bounds rb
                        WHERE
                            c.created_at >= rb.starts_at
                            AND c.created_at < rb.ends_at
                        GROUP BY
                            1,
                            2
                    ) agg
            ),
            '[]'::json
        ) AS data
),
by_source AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.count DESC)
                FROM
                    (
                        SELECT
                            COALESCE(NULLIF(TRIM(c.meta->>'pylon_source'), ''), i.channel::text) AS source,
                            COUNT(*) AS count
                        FROM
                            conversations c
                            INNER JOIN inboxes i ON i.id = c.inbox_id,
                            range_bounds rb
                        WHERE
                            c.created_at >= rb.starts_at
                            AND c.created_at < rb.ends_at
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
tags_by_week AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.week, agg.count DESC)
                FROM
                    (
                        SELECT
                            TO_CHAR(
                                DATE_TRUNC('week', c.created_at AT TIME ZONE $1),
                                'YYYY-MM-DD'
                            ) AS week,
                            t.name AS tag,
                            COUNT(*) AS count
                        FROM
                            conversations c
                            INNER JOIN conversation_tags ct ON ct.conversation_id = c.id
                            INNER JOIN tags t ON t.id = ct.tag_id,
                            range_bounds rb
                        WHERE
                            c.created_at >= rb.starts_at
                            AND c.created_at < rb.ends_at
                        GROUP BY
                            1,
                            2
                    ) agg
            ),
            '[]'::json
        ) AS data
),
median_sla_by_week AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.week)
                FROM
                    (
                        SELECT
                            TO_CHAR(
                                DATE_TRUNC('week', la.created_at AT TIME ZONE $1),
                                'YYYY-MM-DD'
                            ) AS week,
                            COALESCE(
                                percentile_cont(0.5) WITHIN GROUP (
                                    ORDER BY
                                        EXTRACT(
                                            EPOCH
                                            FROM
                                                (la.first_response_met_at - la.created_at)
                                        )
                                ) FILTER (
                                    WHERE
                                        la.first_response_met_at IS NOT NULL
                                ),
                                0
                            ) AS median_first_response_sec,
                            COALESCE(
                                percentile_cont(0.5) WITHIN GROUP (
                                    ORDER BY
                                        EXTRACT(
                                            EPOCH
                                            FROM
                                                (la.resolution_met_at - la.created_at)
                                        )
                                ) FILTER (
                                    WHERE
                                        la.resolution_met_at IS NOT NULL
                                ),
                                0
                            ) AS median_resolution_sec
                        FROM
                            (
                                SELECT DISTINCT ON (conversation_id)
                                    conversation_id,
                                    created_at,
                                    first_response_met_at,
                                    resolution_met_at
                                FROM
                                    applied_slas a,
                                    range_bounds rb
                                WHERE
                                    a.created_at >= rb.starts_at
                                    AND a.created_at < rb.ends_at
                                ORDER BY
                                    conversation_id,
                                    created_at DESC,
                                    id DESC
                            ) la
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
solved_within_week AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.week)
                FROM
                    (
                        SELECT
                            TO_CHAR(
                                DATE_TRUNC('week', c.resolved_at AT TIME ZONE $1),
                                'YYYY-MM-DD'
                            ) AS week,
                            COUNT(*) FILTER (
                                WHERE
                                    c.resolved_at - c.created_at <= INTERVAL '7 days'
                            ) AS within_week,
                            COUNT(*) FILTER (
                                WHERE
                                    c.resolved_at - c.created_at > INTERVAL '7 days'
                            ) AS over_week
                        FROM
                            conversations c,
                            range_bounds rb
                        WHERE
                            c.resolved_at IS NOT NULL
                            AND c.resolved_at >= rb.starts_at
                            AND c.resolved_at < rb.ends_at
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
mean_touches_by_source AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.source)
                FROM
                    (
                        SELECT
                            COALESCE(NULLIF(TRIM(c.meta->>'pylon_source'), ''), i.channel::text) AS source,
                            ROUND(
                                AVG(
                                    COALESCE(touch_counts.touches, 0)
                                )::numeric,
                                2
                            ) AS mean_touches
                        FROM
                            conversations c
                            INNER JOIN inboxes i ON i.id = c.inbox_id
                            CROSS JOIN range_bounds rb
                            LEFT JOIN LATERAL (
                                SELECT
                                    COUNT(*) AS touches
                                FROM
                                    conversation_messages m
                                WHERE
                                    m.conversation_id = c.id
                                    AND m.type IN ('incoming', 'outgoing')
                                    AND m.private = false
                                    AND m.created_at >= rb.starts_at
                                    AND m.created_at < rb.ends_at
                            ) touch_counts ON true
                        WHERE
                            c.created_at >= rb.starts_at
                            AND c.created_at < rb.ends_at
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
median_time_in_status AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.week, agg.status)
                FROM
                    (
                        SELECT
                            TO_CHAR(
                                DATE_TRUNC('week', seg.segment_start AT TIME ZONE $1),
                                'YYYY-MM-DD'
                            ) AS week,
                            seg.status_name AS status,
                            percentile_cont(0.5) WITHIN GROUP (
                                ORDER BY
                                    EXTRACT(
                                        EPOCH
                                        FROM
                                            (seg.segment_end - seg.segment_start)
                                    )
                            ) AS median_seconds
                        FROM
                            (
                                WITH status_events AS (
                                    SELECT
                                        m.conversation_id,
                                        m.created_at,
                                        COALESCE(
                                            NULLIF(TRIM(m.meta->>'new_value'), ''),
                                            NULLIF(
                                                TRIM(
                                                    (
                                                        regexp_match(
                                                            m.content,
                                                            'marked the conversation as (.+)$'
                                                        )
                                                    )[1]
                                                ),
                                                ''
                                            )
                                        ) AS status_name
                                    FROM
                                        conversation_messages m
                                    WHERE
                                        m.type = 'activity'
                                        AND m.meta->>'activity_type' = 'status_change'
                                        AND (
                                            NULLIF(TRIM(m.meta->>'new_value'), '') IS NOT NULL
                                            OR m.content ~ 'marked the conversation as '
                                        )
                                ),
                                occupied_segments AS (
                                    SELECT
                                        e.conversation_id,
                                        e.created_at AS segment_start,
                                        COALESCE(
                                            LEAD(e.created_at) OVER (
                                                PARTITION BY
                                                    e.conversation_id
                                                ORDER BY
                                                    e.created_at
                                            ),
                                            NOW()
                                        ) AS segment_end,
                                        e.status_name
                                    FROM
                                        status_events e
                                    WHERE
                                        e.status_name IS NOT NULL
                                ),
                                initial_segments AS (
                                    SELECT
                                        c.id AS conversation_id,
                                        c.created_at AS segment_start,
                                        fe.first_at AS segment_end,
                                        (
                                            SELECT
                                                name
                                            FROM
                                                conversation_statuses
                                            WHERE
                                                name = 'Open'
                                            LIMIT
                                                1
                                        ) AS status_name
                                    FROM
                                        conversations c
                                        INNER JOIN LATERAL (
                                            SELECT
                                                MIN(e.created_at) AS first_at
                                            FROM
                                                status_events e
                                            WHERE
                                                e.conversation_id = c.id
                                        ) fe ON fe.first_at IS NOT NULL
                                        AND fe.first_at > c.created_at
                                    UNION ALL
                                    SELECT
                                        c.id AS conversation_id,
                                        c.created_at AS segment_start,
                                        NOW() AS segment_end,
                                        cs.name AS status_name
                                    FROM
                                        conversations c
                                        INNER JOIN conversation_statuses cs ON cs.id = c.status_id
                                    WHERE
                                        NOT EXISTS (
                                            SELECT
                                                1
                                            FROM
                                                status_events e
                                            WHERE
                                                e.conversation_id = c.id
                                        )
                                ),
                                all_segments AS (
                                    SELECT
                                        segment_start,
                                        segment_end,
                                        status_name
                                    FROM
                                        occupied_segments
                                    UNION ALL
                                    SELECT
                                        segment_start,
                                        segment_end,
                                        status_name
                                    FROM
                                        initial_segments
                                )
                                SELECT
                                    s.segment_start,
                                    s.segment_end,
                                    s.status_name
                                FROM
                                    all_segments s
                            ) seg,
                            range_bounds rb
                        WHERE
                            seg.status_name IS NOT NULL
                            AND seg.segment_end > seg.segment_start
                            AND seg.segment_start >= rb.starts_at
                            AND seg.segment_start < rb.ends_at
                        GROUP BY
                            1,
                            2
                    ) agg
            ),
            '[]'::json
        ) AS data
)
SELECT
    json_build_object(
        'volume_by_day',
        (
            SELECT
                data
            FROM
                volume_by_day
        ),
        'volume_over_time',
        (
            SELECT
                data
            FROM
                volume_over_time
        ),
        'by_hour',
        (
            SELECT
                data
            FROM
                by_hour
        ),
        'by_dow',
        (
            SELECT
                data
            FROM
                by_dow
        ),
        'by_source',
        (
            SELECT
                data
            FROM
                by_source
        ),
        'tags_by_week',
        (
            SELECT
                data
            FROM
                tags_by_week
        ),
        'median_sla_by_week',
        (
            SELECT
                data
            FROM
                median_sla_by_week
        ),
        'solved_within_week',
        (
            SELECT
                data
            FROM
                solved_within_week
        ),
        'mean_touches_by_source',
        (
            SELECT
                data
            FROM
                mean_touches_by_source
        ),
        'median_time_in_status',
        (
            SELECT
                data
            FROM
                median_time_in_status
        )
    ) AS result;

-- name: get-issues-top-accounts
WITH range_bounds AS (
    SELECT
        $1::timestamptz AS starts_at,
        $2::timestamptz AS ends_at
),
account_stats AS (
    SELECT
        COALESCE(o.name, '') AS account_name,
        o.id AS organization_id,
        COUNT(*) FILTER (
            WHERE
                c.resolved_at IS NULL
        ) AS open,
        COUNT(*) FILTER (
            WHERE
                c.resolved_at IS NOT NULL
        ) AS closed,
        COUNT(*) AS total,
        COALESCE(
            ROUND(
                AVG(
                    CASE
                        WHEN c.resolved_at IS NOT NULL THEN EXTRACT(
                            EPOCH
                            FROM
                                (c.resolved_at - c.created_at)
                        )
                    END
                )::numeric,
                0
            ),
            0
        ) AS avg_resolution_sec,
        COALESCE(
            ROUND(AVG(touch_counts.touches)::numeric, 1),
            0
        ) AS avg_touches
    FROM
        conversations c
        LEFT JOIN organizations o ON o.id = c.organization_id
        CROSS JOIN range_bounds rb
        LEFT JOIN LATERAL (
            SELECT
                COUNT(*) AS touches
            FROM
                conversation_messages m
            WHERE
                m.conversation_id = c.id
                AND m.type IN ('incoming', 'outgoing')
                AND m.private = false
                AND m.created_at >= rb.starts_at
                AND m.created_at < rb.ends_at
        ) touch_counts ON true
    WHERE
        c.created_at >= rb.starts_at
        AND c.created_at < rb.ends_at
        AND (
            $3 = ''
            OR COALESCE(o.name, '') ILIKE '%%' || $3 || '%%'
        )
    GROUP BY
        o.id,
        o.name
    ORDER BY
        %s,
        account_name
    LIMIT
        10
)
SELECT
    COALESCE(
        (
            SELECT
                json_agg(row_to_json(account_stats))
            FROM
                account_stats
        ),
        '[]'::json
    ) AS result;

-- name: get-workforce-scalars
WITH range_bounds AS (
    SELECT
        $1::timestamptz AS starts_at,
        $2::timestamptz AS ends_at
),
latest_sla AS (
    SELECT DISTINCT ON (a.conversation_id)
        a.id,
        a.conversation_id,
        a.created_at,
        a.status,
        a.first_response_deadline_at,
        a.resolution_deadline_at,
        a.first_response_met_at,
        a.resolution_met_at,
        a.first_response_breached_at,
        a.resolution_breached_at,
        c.created_at AS conv_created_at,
        c.resolved_at
    FROM
        applied_slas a
        INNER JOIN conversations c ON c.id = a.conversation_id,
        range_bounds rb
    WHERE
        a.created_at >= rb.starts_at
        AND a.created_at < rb.ends_at
    ORDER BY
        a.conversation_id,
        a.created_at DESC,
        a.id DESC
),
next_response AS (
    SELECT
        COUNT(*) FILTER (
            WHERE
                e.status = 'breached'
        ) AS breached,
        COUNT(*) FILTER (
            WHERE
                e.status IN ('met', 'breached')
        ) AS total
    FROM
        sla_events e
        INNER JOIN latest_sla la ON la.id = e.applied_sla_id,
        range_bounds rb
    WHERE
        e.type = 'next_response'
        AND e.created_at >= rb.starts_at
        AND e.created_at < rb.ends_at
)
SELECT
    json_build_object(
        'median_first_response_sec',
        COALESCE(
            (
                SELECT
                    percentile_cont(0.5) WITHIN GROUP (
                        ORDER BY
                            EXTRACT(
                                EPOCH
                                FROM
                                    (
                                        COALESCE(la.first_response_met_at, la.first_response_breached_at) - la.conv_created_at
                                    )
                            )
                    )
                FROM
                    latest_sla la
                WHERE
                    COALESCE(la.first_response_met_at, la.first_response_breached_at) IS NOT NULL
            ),
            0
        ),
        'median_resolution_sec',
        COALESCE(
            (
                SELECT
                    percentile_cont(0.5) WITHIN GROUP (
                        ORDER BY
                            EXTRACT(
                                EPOCH
                                FROM
                                    (
                                        COALESCE(la.resolution_met_at, la.resolution_breached_at) - la.conv_created_at
                                    )
                            )
                    )
                FROM
                    latest_sla la
                WHERE
                    COALESCE(la.resolution_met_at, la.resolution_breached_at) IS NOT NULL
            ),
            0
        ),
        'sla_achievement_rate',
        COALESCE(
            ROUND(
                100.0 * (
                    SELECT
                        COUNT(*)::numeric
                    FROM
                        latest_sla la
                    WHERE
                        la.status = 'met'
                ) / NULLIF((SELECT COUNT(*) FROM latest_sla), 0),
                1
            ),
            0
        ),
        'sla_breached_tickets_percent',
        COALESCE(
            ROUND(
                100.0 * (
                    SELECT
                        COUNT(*)::numeric
                    FROM
                        latest_sla la
                    WHERE
                        la.first_response_breached_at IS NOT NULL
                        OR la.resolution_breached_at IS NOT NULL
                ) / NULLIF((SELECT COUNT(*) FROM latest_sla), 0),
                1
            ),
            0
        ),
        'active_breached_count',
        (
            SELECT
                COUNT(*)::int
            FROM
                (
                    SELECT DISTINCT ON (a.conversation_id)
                        a.status,
                        a.first_response_breached_at,
                        a.resolution_breached_at,
                        s.category AS conv_status_category
                    FROM
                        applied_slas a
                        INNER JOIN conversations c ON c.id = a.conversation_id
                        INNER JOIN conversation_statuses s ON s.id = c.status_id
                    ORDER BY
                        a.conversation_id,
                        a.created_at DESC,
                        a.id DESC
                ) active_sla
            WHERE
                active_sla.status = 'pending'
                AND active_sla.conv_status_category != 'resolved'
                AND (
                    active_sla.first_response_breached_at IS NOT NULL
                    OR active_sla.resolution_breached_at IS NOT NULL
                )
        ),
        'first_response_breach_percent',
        COALESCE(
            ROUND(
                100.0 * (
                    SELECT
                        COUNT(*)::numeric
                    FROM
                        latest_sla la
                    WHERE
                        la.first_response_breached_at IS NOT NULL
                ) / NULLIF(
                    (
                        SELECT
                            COUNT(*)::numeric
                        FROM
                            latest_sla la
                        WHERE
                            la.first_response_deadline_at IS NOT NULL
                    ),
                    0
                ),
                1
            ),
            0
        ),
        'resolution_breach_percent',
        COALESCE(
            ROUND(
                100.0 * (
                    SELECT
                        COUNT(*)::numeric
                    FROM
                        latest_sla la
                    WHERE
                        la.resolution_breached_at IS NOT NULL
                ) / NULLIF(
                    (
                        SELECT
                            COUNT(*)::numeric
                        FROM
                            latest_sla la
                        WHERE
                            la.resolution_deadline_at IS NOT NULL
                    ),
                    0
                ),
                1
            ),
            0
        ),
        'next_response_breach_percent',
        COALESCE(
            ROUND(100.0 * nr.breached::numeric / NULLIF(nr.total, 0), 1),
            0
        )
    )
FROM
    next_response nr;

-- name: get-workforce-charts
WITH range_bounds AS (
    SELECT
        $2::timestamptz AS starts_at,
        $3::timestamptz AS ends_at
),
latest_sla AS (
    SELECT DISTINCT ON (a.conversation_id)
        a.id,
        a.conversation_id,
        a.created_at,
        a.status,
        a.first_response_deadline_at,
        a.resolution_deadline_at,
        a.first_response_met_at,
        a.resolution_met_at,
        a.first_response_breached_at,
        a.resolution_breached_at,
        c.created_at AS conv_created_at,
        c.resolved_at,
        c.assigned_user_id,
        cs.category AS conv_status_category
    FROM
        applied_slas a
        INNER JOIN conversations c ON c.id = a.conversation_id
        INNER JOIN conversation_statuses cs ON cs.id = c.status_id,
        range_bounds rb
    WHERE
        a.created_at >= rb.starts_at
        AND a.created_at < rb.ends_at
    ORDER BY
        a.conversation_id,
        a.created_at DESC,
        a.id DESC
),
bucket_labels AS (
    SELECT
        *
    FROM
        (
            VALUES
                (1, '<1h'),
                (2, '1-4h'),
                (3, '4-8h'),
                (4, '8-24h'),
                (5, '1-3d'),
                (6, '3d+')
        ) AS t (sort_key, label)
),
fr_durations AS (
    SELECT
        CASE
            WHEN sec < 3600 THEN '<1h'
            WHEN sec < 14400 THEN '1-4h'
            WHEN sec < 28800 THEN '4-8h'
            WHEN sec < 86400 THEN '8-24h'
            WHEN sec < 259200 THEN '1-3d'
            ELSE '3d+'
        END AS label
    FROM
        (
            SELECT
                EXTRACT(
                    EPOCH
                    FROM
                        (
                            COALESCE(la.first_response_met_at, la.first_response_breached_at) - la.conv_created_at
                        )
                ) AS sec
            FROM
                latest_sla la
            WHERE
                COALESCE(la.first_response_met_at, la.first_response_breached_at) IS NOT NULL
        ) d
    WHERE
        sec >= 0
),
res_durations AS (
    SELECT
        CASE
            WHEN sec < 3600 THEN '<1h'
            WHEN sec < 14400 THEN '1-4h'
            WHEN sec < 28800 THEN '4-8h'
            WHEN sec < 86400 THEN '8-24h'
            WHEN sec < 259200 THEN '1-3d'
            ELSE '3d+'
        END AS label
    FROM
        (
            SELECT
                EXTRACT(
                    EPOCH
                    FROM
                        (
                            COALESCE(la.resolution_met_at, la.resolution_breached_at) - la.conv_created_at
                        )
                ) AS sec
            FROM
                latest_sla la
            WHERE
                COALESCE(la.resolution_met_at, la.resolution_breached_at) IS NOT NULL
        ) d
    WHERE
        sec >= 0
),
first_response_buckets AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(
                        json_build_object('label', bl.label, 'count', COALESCE(cnt.c, 0))
                        ORDER BY
                            bl.sort_key
                    )
                FROM
                    bucket_labels bl
                    LEFT JOIN (
                        SELECT
                            label,
                            COUNT(*)::int AS c
                        FROM
                            fr_durations
                        GROUP BY
                            1
                    ) cnt ON cnt.label = bl.label
            ),
            '[]'::json
        ) AS data
),
resolution_buckets AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(
                        json_build_object('label', bl.label, 'count', COALESCE(cnt.c, 0))
                        ORDER BY
                            bl.sort_key
                    )
                FROM
                    bucket_labels bl
                    LEFT JOIN (
                        SELECT
                            label,
                            COUNT(*)::int AS c
                        FROM
                            res_durations
                        GROUP BY
                            1
                    ) cnt ON cnt.label = bl.label
            ),
            '[]'::json
        ) AS data
),
median_sla_by_week AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.week)
                FROM
                    (
                        SELECT
                            TO_CHAR(
                                DATE_TRUNC('week', la.created_at AT TIME ZONE $1),
                                'YYYY-MM-DD'
                            ) AS week,
                            COALESCE(
                                percentile_cont(0.5) WITHIN GROUP (
                                    ORDER BY
                                        EXTRACT(
                                            EPOCH
                                            FROM
                                                (
                                                    COALESCE(la.first_response_met_at, la.first_response_breached_at) - la.conv_created_at
                                                )
                                        )
                                ) FILTER (
                                    WHERE
                                        COALESCE(la.first_response_met_at, la.first_response_breached_at) IS NOT NULL
                                ),
                                0
                            ) AS median_first_response_sec,
                            COALESCE(
                                percentile_cont(0.5) WITHIN GROUP (
                                    ORDER BY
                                        EXTRACT(
                                            EPOCH
                                            FROM
                                                (
                                                    COALESCE(la.resolution_met_at, la.resolution_breached_at) - la.conv_created_at
                                                )
                                        )
                                ) FILTER (
                                    WHERE
                                        COALESCE(la.resolution_met_at, la.resolution_breached_at) IS NOT NULL
                                ),
                                0
                            ) AS median_resolution_sec
                        FROM
                            latest_sla la
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
achieved_vs_breached_by_week AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.week)
                FROM
                    (
                        SELECT
                            TO_CHAR(
                                DATE_TRUNC('week', la.created_at AT TIME ZONE $1),
                                'YYYY-MM-DD'
                            ) AS week,
                            COUNT(*) FILTER (
                                WHERE
                                    la.first_response_breached_at IS NULL
                                    AND la.resolution_breached_at IS NULL
                            )::int AS achieved,
                            COUNT(*) FILTER (
                                WHERE
                                    la.first_response_breached_at IS NOT NULL
                                    OR la.resolution_breached_at IS NOT NULL
                            )::int AS breached
                        FROM
                            latest_sla la
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
breach_rate_by_hour AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.hour)
                FROM
                    (
                        SELECT
                            EXTRACT(
                                HOUR
                                FROM
                                    la.conv_created_at AT TIME ZONE $1
                            )::int AS hour,
                            COALESCE(
                                ROUND(
                                    100.0 * COUNT(*) FILTER (
                                        WHERE
                                            la.first_response_breached_at IS NOT NULL
                                            OR la.resolution_breached_at IS NOT NULL
                                    )::numeric / NULLIF(COUNT(*), 0),
                                    1
                                ),
                                0
                            ) AS breach_rate
                        FROM
                            latest_sla la
                        GROUP BY
                            1
                    ) agg
            ),
            '[]'::json
        ) AS data
),
breach_rate_by_dow AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.dow)
                FROM
                    (
                        SELECT
                            EXTRACT(
                                ISODOW
                                FROM
                                    la.conv_created_at AT TIME ZONE $1
                            )::int AS dow,
                            TO_CHAR(la.conv_created_at AT TIME ZONE $1, 'Dy') AS label,
                            COALESCE(
                                ROUND(
                                    100.0 * COUNT(*) FILTER (
                                        WHERE
                                            la.first_response_breached_at IS NOT NULL
                                            OR la.resolution_breached_at IS NOT NULL
                                    )::numeric / NULLIF(COUNT(*), 0),
                                    1
                                ),
                                0
                            ) AS breach_rate
                        FROM
                            latest_sla la
                        GROUP BY
                            1,
                            2
                    ) agg
            ),
            '[]'::json
        ) AS data
),
assignee_stats AS (
    SELECT
        la.assigned_user_id,
        COALESCE(
            NULLIF(TRIM(CONCAT_WS(' ', u.first_name, u.last_name)), ''),
            NULLIF(TRIM(u.email), ''),
            'Unassigned'
        ) AS assignee_name,
        COUNT(*) FILTER (
            WHERE
                la.conv_status_category != 'resolved'
        )::int AS open_count,
        COUNT(*) FILTER (
            WHERE
                la.conv_status_category = 'resolved'
        )::int AS closed_count,
        COUNT(*)::int AS total,
        COALESCE(
            percentile_cont(0.5) WITHIN GROUP (
                ORDER BY
                    EXTRACT(
                        EPOCH
                        FROM
                            (
                                COALESCE(la.first_response_met_at, la.first_response_breached_at) - la.conv_created_at
                            )
                    )
            ) FILTER (
                WHERE
                    COALESCE(la.first_response_met_at, la.first_response_breached_at) IS NOT NULL
            ),
            0
        ) AS median_first_response_sec,
        COALESCE(
            percentile_cont(0.5) WITHIN GROUP (
                ORDER BY
                    EXTRACT(
                        EPOCH
                        FROM
                            (
                                COALESCE(la.resolution_met_at, la.resolution_breached_at) - la.conv_created_at
                            )
                    )
            ) FILTER (
                WHERE
                    COALESCE(la.resolution_met_at, la.resolution_breached_at) IS NOT NULL
            ),
            0
        ) AS median_resolution_sec,
        COALESCE(
            ROUND(
                100.0 * COUNT(*) FILTER (
                    WHERE
                        la.first_response_breached_at IS NOT NULL
                        OR la.resolution_breached_at IS NOT NULL
                )::numeric / NULLIF(COUNT(*), 0),
                1
            ),
            0
        ) AS breach_rate_percent
    FROM
        latest_sla la
        LEFT JOIN users u ON u.id = la.assigned_user_id
    GROUP BY
        la.assigned_user_id,
        assignee_name
),
assignee_overview AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.total DESC, agg.assignee_name)
                FROM
                    (
                        SELECT
                            assigned_user_id,
                            assignee_name,
                            open_count,
                            closed_count,
                            total,
                            median_first_response_sec,
                            median_resolution_sec,
                            breach_rate_percent
                        FROM
                            assignee_stats
                    ) agg
            ),
            '[]'::json
        ) AS data
),
first_response_by_assignee AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.median_first_response_sec DESC)
                FROM
                    (
                        SELECT
                            assignee_name,
                            median_first_response_sec
                        FROM
                            assignee_stats
                        WHERE
                            median_first_response_sec > 0
                        ORDER BY
                            median_first_response_sec DESC
                        LIMIT
                            15
                    ) agg
            ),
            '[]'::json
        ) AS data
),
resolution_by_assignee AS (
    SELECT
        COALESCE(
            (
                SELECT
                    json_agg(row_to_json(agg) ORDER BY agg.median_resolution_sec DESC)
                FROM
                    (
                        SELECT
                            assignee_name,
                            median_resolution_sec
                        FROM
                            assignee_stats
                        WHERE
                            median_resolution_sec > 0
                        ORDER BY
                            median_resolution_sec DESC
                        LIMIT
                            15
                    ) agg
            ),
            '[]'::json
        ) AS data
)
SELECT
    json_build_object(
        'first_response_buckets',
        (SELECT data FROM first_response_buckets),
        'resolution_buckets',
        (SELECT data FROM resolution_buckets),
        'median_sla_by_week',
        (SELECT data FROM median_sla_by_week),
        'achieved_vs_breached_by_week',
        (SELECT data FROM achieved_vs_breached_by_week),
        'breach_rate_by_hour',
        (SELECT data FROM breach_rate_by_hour),
        'breach_rate_by_dow',
        (SELECT data FROM breach_rate_by_dow),
        'first_response_by_assignee',
        (SELECT data FROM first_response_by_assignee),
        'resolution_by_assignee',
        (SELECT data FROM resolution_by_assignee),
        'assignee_overview',
        (SELECT data FROM assignee_overview)
    ) AS result;