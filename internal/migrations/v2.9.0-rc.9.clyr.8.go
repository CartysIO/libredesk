package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9_CLYR8 adds the Linear tables (Clyr fork): the Linear issues linked to a conversation,
// and the Linear comments already added to a conversation as a note.
func V2_9_0_RC9_CLYR8(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_linear_issues (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			issue_id TEXT NOT NULL,
			identifier TEXT NOT NULL,
			url TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			state TEXT NOT NULL DEFAULT '',
			created_by BIGINT REFERENCES users(id) ON DELETE SET NULL ON UPDATE CASCADE NULL,
			CONSTRAINT constraint_conversation_linear_issues_on_conversation_issue_unique UNIQUE (conversation_id, issue_id)
		);
		CREATE INDEX IF NOT EXISTS index_conversation_linear_issues_on_issue_id ON conversation_linear_issues (issue_id);

		CREATE TABLE IF NOT EXISTS linear_comment_notes (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			comment_id TEXT NOT NULL,
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			message_id BIGINT REFERENCES conversation_messages(id) ON DELETE SET NULL ON UPDATE CASCADE NULL,
			CONSTRAINT constraint_linear_comment_notes_on_comment_conversation_unique UNIQUE (comment_id, conversation_id)
		);
	`)
	return err
}
