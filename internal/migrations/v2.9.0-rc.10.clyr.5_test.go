package migrations

import (
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC10_CLYR5IngestedMessageSourceIDs(t *testing.T) {
	db := testutil.NewDB(t, "mig_clyr_5_ingested_source")

	var inboxID, contactID, convID int
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Email', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, email) VALUES ('contact', 'T', 't@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&convID, `
		INSERT INTO conversations (contact_id, inbox_id, status_id, subject)
		VALUES ($1, $2, 1, 'backfill')`, contactID, inboxID); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`DELETE FROM ingested_message_source_ids`)
	db.MustExec(`INSERT INTO conversation_messages (type, status, conversation_id, content, text_content, sender_id, sender_type, source_id)
		VALUES ('incoming', 'received', $1, 'a', 'a', $2, 'contact', '  mig-backfill@example.com  ')`, convID, contactID)

	for range 2 {
		if err := V2_9_0_RC10_CLYR5(db, nil, nil); err != nil {
			t.Fatal(err)
		}
	}

	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM ingested_message_source_ids WHERE source_id = 'mig-backfill@example.com'`); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("backfill tombstone count = %d, want 1", n)
	}
}
