package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9_CLYR2 adds the Slack bridge tables (Clyr fork): one Slack thread per conversation,
// and the Slack messages the bridge has handled.
func V2_9_0_RC9_CLYR2(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS slack_threads (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			channel_id TEXT NOT NULL,
			thread_ts TEXT NOT NULL,
			CONSTRAINT constraint_slack_threads_on_conversation_id_unique UNIQUE (conversation_id),
			CONSTRAINT constraint_slack_threads_on_channel_thread_unique UNIQUE (channel_id, thread_ts)
		);

		CREATE TABLE IF NOT EXISTS slack_messages (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			channel_id TEXT NOT NULL,
			slack_ts TEXT NOT NULL,
			direction TEXT NOT NULL,
			message_id BIGINT REFERENCES conversation_messages(id) ON DELETE CASCADE ON UPDATE CASCADE NULL,
			CONSTRAINT constraint_slack_messages_on_direction CHECK (direction IN ('in', 'out')),
			CONSTRAINT constraint_slack_messages_on_channel_ts_unique UNIQUE (channel_id, slack_ts)
		);
		CREATE INDEX IF NOT EXISTS index_slack_messages_on_message_id ON slack_messages (message_id) WHERE message_id IS NOT NULL;
	`)
	return err
}
