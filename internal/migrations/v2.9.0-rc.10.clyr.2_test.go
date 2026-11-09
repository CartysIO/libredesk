package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC10_CLYR2Idempotent(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_backdated_perm")
	for range 2 {
		if err := V2_9_0_RC10_CLYR2(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}
}
