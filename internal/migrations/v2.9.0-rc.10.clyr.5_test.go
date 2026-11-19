package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC10_CLYR5IngestedMessageSourceIDs(t *testing.T) {
	db := testutil.NewDB(t, "mig_clyr_5_ingested_source")
	if err := V2_9_0_RC10_CLYR5(db, nil, nil); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'ingested_message_source_ids'`); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("ingested_message_source_ids table missing")
	}
}