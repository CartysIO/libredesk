package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V2_9_0_RC9_CLYR5 lets a contact belong to several organizations (Clyr fork).
// users.organization_id stays as the contact's default organization and gets a membership row.
// The Agent and Admin roles get the new conversations:update_organization permission.
func V2_9_0_RC9_CLYR5(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS contact_organizations (
			contact_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
			PRIMARY KEY (contact_id, organization_id)
		);
		CREATE INDEX IF NOT EXISTS index_contact_organizations_on_organization_id ON contact_organizations(organization_id);

		INSERT INTO contact_organizations (contact_id, organization_id)
		SELECT id, organization_id FROM users WHERE organization_id IS NOT NULL
		ON CONFLICT DO NOTHING;

		UPDATE roles SET permissions = array_append(permissions, 'conversations:update_organization')
		WHERE name IN ('Agent', 'Admin') AND NOT ('conversations:update_organization' = ANY(permissions));
	`)
	return err
}
