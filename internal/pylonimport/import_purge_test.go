package pylonimport

import (
	"context"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

// TestImportThenPurgeNonImported mirrors runPylonImport with --pylon-purge-non-imported:
// Importer.Run, then NonImportedPurger removes pre-cutover conversations.
func TestImportThenPurgeNonImported(t *testing.T) {
	t0 := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	db := testutil.NewDB(t, "pylon_import_then_purge")

	var inbox, cust int
	db.Get(&inbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Support', 'email', FALSE) RETURNING id`)
	db.Get(&cust, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'vayco@example.com', 'Vayco') RETURNING id`)

	var native int
	if err := db.Get(&native, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, subject, created_at, last_message_at)
		VALUES ('999', $1, $2, (SELECT id FROM conversation_statuses WHERE name = 'Open'), 'Pre-cutover', $3, $3) RETURNING id`,
		cust, inbox, t0); err != nil {
		t.Fatal(err)
	}

	d := Data{
		Users:    []User{{ID: "u1", Name: "eran", Email: "eran@clyr.io"}},
		Accounts: []Account{{ID: "a1", Name: "Vayco"}},
		Contacts: []Contact{{ID: "c1", Name: "V", Email: "vayco@example.com", Account: &Ref{ID: "a1"}}},
		Issues: []Issue{{
			ID: "i1", Number: 5000, Title: "From Pylon", State: "open", Source: "email",
			Account: &Ref{ID: "a1"}, Requester: &Ref{ID: "c1"}, Assignee: &Ref{ID: "u1"}, CreatedAt: t0,
		}},
		Messages: map[string][]Message{
			"i1": {{
				ID: "m1", MessageHTML: "hello", Timestamp: t0,
				Author: &MessageAuthor{Contact: &Ref{ID: "c1", Email: "vayco@example.com"}},
			}},
		},
	}

	im := &Importer{DB: db, Opt: Options{ReferenceFloor: 4000, MediaStore: "fs"}}
	if _, err := im.Run(context.Background(), d); err != nil {
		t.Fatal(err)
	}
	if count(t, db, `SELECT COUNT(*) FROM conversations WHERE id = $1`, native) != 1 {
		t.Fatal("pre-cutover conversation should exist after import")
	}
	if n := count(t, db, `SELECT COUNT(*) FROM conversations WHERE meta ? '`+MetaIssueID+`'`); n != 1 {
		t.Fatalf("want 1 imported conversation, got %d", n)
	}

	p := &NonImportedPurger{DB: db, Opt: PurgeNonImportedOptions{Confirm: true}}
	rep, err := p.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Conversations != 1 {
		t.Fatalf("purge report conversations = %d, want 1", rep.Conversations)
	}
	if count(t, db, `SELECT COUNT(*) FROM conversations WHERE id = $1`, native) != 0 {
		t.Fatal("pre-cutover conversation should be removed")
	}
	if n := count(t, db, `SELECT COUNT(*) FROM conversations WHERE meta ? '`+MetaIssueID+`'`); n != 1 {
		t.Fatalf("imported conversation should remain, count %d", n)
	}
}
