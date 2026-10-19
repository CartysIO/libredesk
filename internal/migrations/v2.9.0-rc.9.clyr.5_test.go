package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC9_CLYR5ContactOrganizations(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_contact_organizations")
	// A database from before this migration has no membership table and no new permission.
	db.MustExec(`DROP TABLE contact_organizations`)
	db.MustExec(`UPDATE roles SET permissions = array_remove(permissions, 'conversations:update_organization')`)

	var orgID, contactID int
	db.QueryRow(`INSERT INTO organizations (name, external_id) VALUES ('Acme', 'team_acme') RETURNING id`).Scan(&orgID)
	db.QueryRow(`INSERT INTO users (type, first_name, last_name, email, organization_id) VALUES ('contact', 'Alex', '', 'alex@example.com', $1) RETURNING id`, orgID).Scan(&contactID)
	if orgID == 0 || contactID == 0 {
		t.Fatal("seeding failed")
	}

	// Running it twice proves it is idempotent, which matters because a fresh install runs every migration.
	for range 2 {
		if err := V2_9_0_RC9_CLYR5(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	var members int
	if err := db.Get(&members, `SELECT COUNT(*) FROM contact_organizations WHERE contact_id = $1 AND organization_id = $2`, contactID, orgID); err != nil {
		t.Fatal(err)
	}
	if members != 1 {
		t.Fatalf("contact has %d membership rows for its organization, want 1", members)
	}

	for _, role := range []string{"Agent", "Admin"} {
		var n int
		if err := db.Get(&n, `SELECT COUNT(*) FROM roles r, unnest(r.permissions) p WHERE r.name = $1 AND p = 'conversations:update_organization'`, role); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("role %s has the permission %d times, want 1", role, n)
		}
	}
}
