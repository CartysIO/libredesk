package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9_CLYR3 adds the conversations:delete permission to the Admin role and the
// conversation_deleted activity log type (Clyr fork).
func V2_9_0_RC9_CLYR3(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	if _, err := db.Exec(`
		UPDATE roles
		SET permissions = array_append(permissions, 'conversations:delete')
		WHERE name = 'Admin' AND NOT ('conversations:delete' = ANY(permissions));
	`); err != nil {
		return err
	}
	_, err := db.Exec(`ALTER TYPE activity_log_type ADD VALUE IF NOT EXISTS 'conversation_deleted';`)
	return err
}
