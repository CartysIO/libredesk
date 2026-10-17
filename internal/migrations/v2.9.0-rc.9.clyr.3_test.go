package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC9_CLYR3ConversationDelete(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_conversation_delete")
	db.MustExec(`UPDATE roles SET permissions = array_remove(permissions, 'conversations:delete') WHERE name = 'Admin'`)

	// Running it twice proves it is idempotent, which matters because a fresh install runs every migration.
	for range 2 {
		if err := V2_9_0_RC9_CLYR3(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM roles, unnest(permissions) p WHERE name = 'Admin' AND p = 'conversations:delete'`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("Admin has conversations:delete %d times, want 1", count)
	}
	var agentHas bool
	if err := db.Get(&agentHas, `SELECT 'conversations:delete' = ANY(permissions) FROM roles WHERE name = 'Agent'`); err != nil {
		t.Fatal(err)
	}
	if agentHas {
		t.Fatal("Agent role got conversations:delete")
	}
	var typeExists bool
	if err := db.Get(&typeExists, `SELECT EXISTS (SELECT 1 FROM pg_enum WHERE enumlabel = 'conversation_deleted')`); err != nil {
		t.Fatal(err)
	}
	if !typeExists {
		t.Fatal("activity_log_type has no conversation_deleted value")
	}
}
