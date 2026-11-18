package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC10_CLYR3 adds working-second SLA durations on applied_slas and backfills Pylon import metadata.
func V2_9_0_RC10_CLYR3(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE applied_slas
			ADD COLUMN IF NOT EXISTS first_response_working_sec INT NULL,
			ADD COLUMN IF NOT EXISTS resolution_working_sec INT NULL;
	`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
		UPDATE applied_slas a
		SET
			first_response_working_sec = COALESCE(a.first_response_working_sec, sub.fr_sec),
			resolution_working_sec = COALESCE(a.resolution_working_sec, sub.res_sec)
		FROM (
			SELECT
				a2.id,
				COALESCE(
					NULLIF(c.meta->>'pylon_first_response_seconds', '')::int,
					NULLIF(c.meta#>>'{pylon,first_response_seconds}', '')::int
				) AS fr_sec,
				COALESCE(
					NULLIF(c.meta->>'pylon_resolution_seconds', '')::int,
					NULLIF(c.meta#>>'{pylon,resolution_seconds}', '')::int
				) AS res_sec
			FROM applied_slas a2
			INNER JOIN conversations c ON c.id = a2.conversation_id
			WHERE
				(
					a2.first_response_working_sec IS NULL
					AND (
						jsonb_path_exists(c.meta, '$.pylon_first_response_seconds')
						OR jsonb_path_exists(c.meta, '$.pylon.first_response_seconds')
					)
				)
				OR (
					a2.resolution_working_sec IS NULL
					AND (
						jsonb_path_exists(c.meta, '$.pylon_resolution_seconds')
						OR jsonb_path_exists(c.meta, '$.pylon.resolution_seconds')
					)
				)
		) sub
		WHERE
			a.id = sub.id
			AND (sub.fr_sec IS NOT NULL OR sub.res_sec IS NOT NULL);
	`)
	return err
}
