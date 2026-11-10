package migrations

import (
	"slices"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
)

// slackSchema returns the columns, indexes and constraints of the Slack bridge tables.
func slackSchema(t *testing.T, db *sqlx.DB) []string {
	t.Helper()
	var out []string
	if err := db.Select(&out, `
		SELECT table_name || '.' || column_name || ':' || data_type || ':' || is_nullable || ':' || COALESCE(column_default, '')
		FROM information_schema.columns WHERE table_name IN ('slack_threads', 'slack_messages')
		UNION ALL
		SELECT indexname || ':' || indexdef FROM pg_indexes WHERE tablename IN ('slack_threads', 'slack_messages')
		UNION ALL
		SELECT conname || ':' || pg_get_constraintdef(oid) FROM pg_constraint
		WHERE conrelid::regclass::text IN ('slack_threads', 'slack_messages')`); err != nil {
		t.Fatal(err)
	}
	slices.Sort(out)
	return out
}

func TestV2_9_0_RC9_CLYR2SlackBridge(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_slack_bridge")
	want := slackSchema(t, db)
	if len(want) == 0 {
		t.Fatal("schema.sql has no slack bridge tables")
	}

	db.MustExec(`DROP TABLE slack_messages; DROP TABLE slack_threads`)
	// Running it twice proves it is idempotent, which matters because a fresh install runs every migration.
	for range 2 {
		if err := V2_9_0_RC9_CLYR2(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}
	got := slackSchema(t, db)
	if !slices.Equal(got, want) {
		t.Fatalf("migration result differs from schema.sql\nmigration: %v\nschema:    %v", got, want)
	}
}
