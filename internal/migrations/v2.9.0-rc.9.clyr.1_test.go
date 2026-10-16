package migrations

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
)

// organizationSchema returns the columns, indexes and foreign keys that the organizations migration owns.
func organizationSchema(t *testing.T, db *sqlx.DB) []string {
	t.Helper()
	var out []string
	if err := db.Select(&out, `
		SELECT table_name || '.' || column_name || ':' || data_type || ':' || is_nullable || ':' || COALESCE(column_default, '')
		FROM information_schema.columns
		WHERE table_name = 'organizations' OR (table_name IN ('users', 'conversations') AND column_name = 'organization_id')
		UNION ALL
		SELECT indexname || ':' || indexdef FROM pg_indexes WHERE indexdef ILIKE '%organization%'
		UNION ALL
		SELECT conrelid::regclass || ':' || pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid::regclass::text IN ('organizations', 'users', 'conversations') AND pg_get_constraintdef(oid) ILIKE '%organization%'
		   OR conrelid::regclass::text = 'organizations'`); err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

func TestV2_9_0_RC9_CLYR1Organizations(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_organizations")
	want := organizationSchema(t, db)
	if len(want) == 0 {
		t.Fatal("schema.sql has no organization objects")
	}

	db.MustExec(`ALTER TABLE conversations DROP COLUMN organization_id; ALTER TABLE users DROP COLUMN organization_id; DROP TABLE organizations`)
	// Running it twice proves it is idempotent, which matters because a fresh install runs every migration.
	for range 2 {
		if err := V2_9_0_RC9_CLYR1(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}
	got := organizationSchema(t, db)
	if !slices.Equal(got, want) {
		t.Fatalf("migration result differs from schema.sql\nmigration: %v\nschema:    %v", got, want)
	}
}
