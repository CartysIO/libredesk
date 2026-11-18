-- Pylon Workforce SLA parity (see docs/reports/pylon-workforce-parity.md).
-- Auto-ack cluster: 3.5h–4.5h wall clock between conversation create and first response.

CREATE OR REPLACE FUNCTION pylon_workforce_anchor_at(meta jsonb, conv_created_at timestamptz)
RETURNS timestamptz
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT COALESCE(
        NULLIF(meta->>'pylon_created_at', '')::timestamptz,
        NULLIF(meta->'pylon'->>'created_at', '')::timestamptz,
        conv_created_at
    );
$$;

CREATE OR REPLACE FUNCTION pylon_is_imported_conversation(meta jsonb)
RETURNS boolean
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT meta ? 'pylon_issue_id'
        OR NULLIF(meta->>'pylon_number', '') IS NOT NULL
        OR meta ? 'pylon';
$$;

CREATE OR REPLACE FUNCTION pylon_resolution_seconds(meta jsonb)
RETURNS double precision
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT COALESCE(
        NULLIF(meta->>'pylon_resolution_seconds', '')::double precision,
        NULLIF(meta#>>'{pylon,resolution_seconds}', '')::double precision
    );
$$;

CREATE OR REPLACE FUNCTION workforce_first_response_duration_sec(
    conv_created_at timestamptz,
    first_response_met_at timestamptz,
    first_response_breached_at timestamptz
)
RETURNS double precision
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE
        WHEN COALESCE(first_response_met_at, first_response_breached_at) IS NULL THEN NULL
        WHEN EXTRACT(
            EPOCH
            FROM
                (COALESCE(first_response_met_at, first_response_breached_at) - conv_created_at)
        ) BETWEEN 12600 AND 16200 THEN NULL
        ELSE EXTRACT(
            EPOCH
            FROM
                (COALESCE(first_response_met_at, first_response_breached_at) - conv_created_at)
        )
    END;
$$;

CREATE OR REPLACE FUNCTION workforce_resolution_duration_sec(
    meta jsonb,
    conv_created_at timestamptz,
    resolution_working_sec int,
    resolution_met_at timestamptz,
    resolution_breached_at timestamptz
)
RETURNS double precision
LANGUAGE sql
IMMUTABLE
AS $$
    SELECT CASE
        WHEN resolution_working_sec IS NOT NULL THEN resolution_working_sec::double precision
        WHEN pylon_resolution_seconds(meta) IS NOT NULL THEN pylon_resolution_seconds(meta)
        WHEN pylon_is_imported_conversation(meta) THEN NULL
        WHEN COALESCE(resolution_met_at, resolution_breached_at) IS NOT NULL THEN EXTRACT(
            EPOCH
            FROM (COALESCE(resolution_met_at, resolution_breached_at) - conv_created_at)
        )
        ELSE NULL
    END;
$$;