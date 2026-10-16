package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9_CLYR1 adds organizations (Clyr fork). Contacts and conversations get an optional organization.
// The version sorts after v2.9.0-rc.9 and before upstream's next release, so upstream migrations still run after it.
func V2_9_0_RC9_CLYR1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS organizations (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			updated_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			"name" TEXT NOT NULL,
			external_id TEXT NULL,
			CONSTRAINT constraint_organizations_on_name CHECK (length("name") <= 255),
			CONSTRAINT constraint_organizations_on_external_id CHECK (length(external_id) <= 128)
		);
		CREATE UNIQUE INDEX IF NOT EXISTS index_unique_organizations_on_external_id ON organizations (external_id) WHERE external_id IS NOT NULL;
		CREATE INDEX IF NOT EXISTS index_tgrm_organizations_on_name ON organizations USING GIN ("name" gin_trgm_ops);

		ALTER TABLE users ADD COLUMN IF NOT EXISTS organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL ON UPDATE CASCADE NULL;
		CREATE INDEX IF NOT EXISTS index_users_on_organization_id ON users(organization_id) WHERE organization_id IS NOT NULL;

		ALTER TABLE conversations ADD COLUMN IF NOT EXISTS organization_id BIGINT REFERENCES organizations(id) ON DELETE SET NULL ON UPDATE CASCADE;
		CREATE INDEX IF NOT EXISTS index_conversations_on_organization_id ON conversations (organization_id) WHERE organization_id IS NOT NULL;
	`)
	return err
}
