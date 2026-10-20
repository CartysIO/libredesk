package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9_CLYR7 gives organizations custom attributes and email domains (Clyr fork).
// An email contact with no organization joins the one organization that owns its email domain.
func V2_9_0_RC9_CLYR7(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE organizations ADD COLUMN IF NOT EXISTS custom_attributes JSONB DEFAULT '{}'::jsonb NOT NULL;
		ALTER TABLE organizations ADD COLUMN IF NOT EXISTS domains TEXT[] DEFAULT '{}'::TEXT[] NOT NULL;
		CREATE INDEX IF NOT EXISTS index_organizations_on_domains ON organizations USING GIN (domains);
	`)
	return err
}
