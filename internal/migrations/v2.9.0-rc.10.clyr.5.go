package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC10_CLYR5 adds a tombstone table so deleted conversations do not lose inbound email deduplication.
func V2_9_0_RC10_CLYR5(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS ingested_message_source_ids (
			source_id TEXT PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
		);

		INSERT INTO ingested_message_source_ids (source_id)
		SELECT DISTINCT NULLIF(TRIM(source_id), '')
		FROM conversation_messages
		WHERE NULLIF(TRIM(source_id), '') IS NOT NULL
		ON CONFLICT (source_id) DO NOTHING;
	`)
	return err
}
