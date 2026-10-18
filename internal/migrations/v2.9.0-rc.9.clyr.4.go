package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9_CLYR4 adds the "On you" (open) and "On customer" (waiting) conversation statuses
// that the Kanban board uses (Clyr fork).
func V2_9_0_RC9_CLYR4(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		INSERT INTO conversation_statuses (name, category) VALUES
		('On you', 'open'),
		('On customer', 'waiting')
		ON CONFLICT (name) DO UPDATE SET category = EXCLUDED.category;
	`)
	return err
}
