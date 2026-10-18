package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC9_CLYR4KanbanStatuses(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_kanban_statuses")
	// A database from before this migration does not have the two statuses.
	db.MustExec(`DELETE FROM conversation_statuses WHERE name IN ('On you', 'On customer')`)

	// Running it twice proves it is idempotent, which matters because a fresh install runs every migration.
	for range 2 {
		if err := V2_9_0_RC9_CLYR4(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	want := map[string]string{"On you": "open", "On customer": "waiting"}
	for name, category := range want {
		var got []string
		if err := db.Select(&got, `SELECT category::text FROM conversation_statuses WHERE name = $1`, name); err != nil {
			t.Fatal(err)
		}
		if len(got) != 1 || got[0] != category {
			t.Fatalf("status %q has categories %v, want exactly [%s]", name, got, category)
		}
	}
}
