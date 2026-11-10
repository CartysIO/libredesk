package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC9_CLYR7OrganizationFields(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_org_fields")
	// A database from before this migration has neither column.
	db.MustExec(`DROP INDEX index_organizations_on_domains; ALTER TABLE organizations DROP COLUMN custom_attributes, DROP COLUMN domains`)
	db.MustExec(`INSERT INTO organizations (name, external_id) VALUES ('Acme', 'team_1')`)

	// Running it twice proves it is idempotent, which matters because a fresh install runs every migration.
	for range 2 {
		if err := V2_9_0_RC9_CLYR7(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}
	var attrs string
	var domains int
	if err := db.QueryRow(`SELECT custom_attributes::TEXT, cardinality(domains) FROM organizations WHERE external_id = 'team_1'`).Scan(&attrs, &domains); err != nil {
		t.Fatal(err)
	}
	if attrs != "{}" || domains != 0 {
		t.Fatalf("existing organization got %s and %d domains", attrs, domains)
	}
}
