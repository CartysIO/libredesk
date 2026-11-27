package pylonimport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestSyncerInsertsMissingMessageAndUpdatesStatus(t *testing.T) {
	db := testutil.NewDB(t, "pylon_sync")
	t0 := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	cutover := time.Date(2026, 3, 15, 18, 0, 0, 0, time.UTC)
	importedAt := t0.Add(24 * time.Hour)

	var inbox, agent, contact int
	db.Get(&inbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Support', 'email', FALSE) RETURNING id`)
	db.Get(&agent, `INSERT INTO users (type, email, first_name) VALUES ('agent', 'eran@clyr.io', 'Eran') RETURNING id`)
	db.Get(&contact, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'v@example.com', 'V') RETURNING id`)

	meta, _ := json.Marshal(map[string]string{
		MetaIssueID:    "i1",
		MetaImportedAt: importedAt.Format(time.RFC3339),
	})
	var conv int
	if err := db.Get(&conv, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, subject, created_at, last_message_at, meta)
		VALUES ('4001', $1, $2, (SELECT id FROM conversation_statuses WHERE name = 'Open'), 'Help', $3, $3, $4) RETURNING id`,
		contact, inbox, t0, meta); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type, content, created_at, meta)
		VALUES ('incoming', 'received', $1, $2, 'contact', 'old', $3, $4)`,
		conv, contact, t0, `{"pylon_message_id":"m1"}`)

	d := Data{
		Users:    []User{{ID: "u1", Name: "Eran", Email: "eran@clyr.io"}},
		Contacts: []Contact{{ID: "c1", Name: "V", Email: "v@example.com"}},
		Issues: []Issue{{
			ID: "i1", Number: 4001, Title: "Help", State: "closed", Source: "email",
			Requester: &Ref{ID: "c1"}, CreatedAt: t0, ResolutionTime: ptrTime(t0.Add(2 * time.Hour)),
		}},
		Messages: map[string][]Message{
			"i1": {
				{ID: "m1", MessageHTML: "old", Timestamp: t0, Author: &MessageAuthor{Contact: &Ref{ID: "c1"}}},
				{ID: "m2", MessageHTML: "gap reply", Timestamp: t0.Add(time.Hour), Author: &MessageAuthor{User: &Ref{ID: "u1"}}},
			},
		},
	}
	im := &Importer{DB: db, Opt: Options{MediaStore: "fs", AllowMissingFiles: true, AgentAliases: map[string]string{}}}
	syncer := &Syncer{
		DB:  db,
		Imp: im,
		Opt: SyncOptions{CutoverEnd: cutover, ReportPath: t.TempDir() + "/report.json"},
	}
	rep, err := syncer.Run(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.MessagesInserted != 1 || rep.Summary.StatusUpdated != 1 {
		t.Fatalf("summary: %+v", rep.Summary)
	}
	if count(t, db, `SELECT COUNT(*) FROM conversation_messages WHERE conversation_id = $1 AND meta->>'pylon_message_id' = 'm2'`, conv) != 1 {
		t.Fatal("missing m2")
	}
	if got := str(t, db, `SELECT cs.name FROM conversations c JOIN conversation_statuses cs ON cs.id = c.status_id WHERE c.id = $1`, conv); got != StatusResolved {
		t.Fatalf("status = %q", got)
	}
}

func TestSyncerTagsBySourceIDWithoutDuplicateInsert(t *testing.T) {
	db := testutil.NewDB(t, "pylon_sync_tag")
	t0 := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	cutover := time.Date(2026, 3, 15, 18, 0, 0, 0, time.UTC)

	var inbox, contact int
	db.Get(&inbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Support', 'email', FALSE) RETURNING id`)
	db.Get(&contact, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'v@example.com', 'V') RETURNING id`)

	meta, _ := json.Marshal(map[string]string{MetaIssueID: "i1"})
	var conv int
	db.Get(&conv, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, subject, created_at, last_message_at, meta)
		VALUES ('4003', $1, $2, (SELECT id FROM conversation_statuses WHERE name = 'Open'), 'Help', $3, $3, $4) RETURNING id`,
		contact, inbox, t0, meta)
	db.MustExec(`INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type, content, source_id, created_at)
		VALUES ('incoming', 'received', $1, $2, 'contact', 'live email', '<live@mail>', $3)`,
		conv, contact, t0)

	d := Data{
		Contacts: []Contact{{ID: "c1", Email: "v@example.com"}},
		Issues:   []Issue{{ID: "i1", Number: 4003, State: "open", Source: "email", Requester: &Ref{ID: "c1"}, CreatedAt: t0}},
		Messages: map[string][]Message{
			"i1": {{
				ID: "m-live", MessageHTML: "live email", Timestamp: t0,
				Author:    &MessageAuthor{Contact: &Ref{ID: "c1"}},
				EmailInfo: &EmailInfo{MessageID: "<live@mail>"},
			}},
		},
	}
	im := &Importer{DB: db, Opt: Options{MediaStore: "fs", AllowMissingFiles: true}}
	if err := im.load(d); err != nil {
		t.Fatal(err)
	}
	rep, err := (&Syncer{DB: db, Imp: im, Opt: SyncOptions{CutoverEnd: cutover}}).Run(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.MessagesInserted != 0 || rep.Summary.MessagesTagged != 1 {
		t.Fatalf("summary: %+v", rep.Summary)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM conversation_messages WHERE conversation_id = $1`, conv); n != 1 {
		t.Fatalf("message count = %d", n)
	}
	if got := str(t, db, `SELECT meta->>'pylon_message_id' FROM conversation_messages WHERE conversation_id = $1`, conv); got != "m-live" {
		t.Fatalf("pylon_message_id = %q", got)
	}
}

func TestSyncerSkipsStatusWhenNativeAfterCutover(t *testing.T) {
	db := testutil.NewDB(t, "pylon_sync_native")
	t0 := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	cutover := time.Date(2026, 3, 15, 18, 0, 0, 0, time.UTC)
	importedAt := t0.Add(24 * time.Hour)

	var inbox, contact int
	db.Get(&inbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Support', 'email', FALSE) RETURNING id`)
	db.Get(&contact, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'v@example.com', 'V') RETURNING id`)

	meta, _ := json.Marshal(map[string]string{MetaIssueID: "i1", MetaImportedAt: importedAt.Format(time.RFC3339)})
	var conv int
	db.Get(&conv, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, subject, created_at, last_message_at, meta)
		VALUES ('4002', $1, $2, (SELECT id FROM conversation_statuses WHERE name = 'Open'), 'Help', $3, $3, $4) RETURNING id`,
		contact, inbox, t0, meta)
	db.MustExec(`INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type, content, created_at)
		VALUES ('outgoing', 'sent', $1, $2, 'agent', 'after cutover', $3)`,
		conv, contact, cutover.Add(time.Hour))

	d := Data{
		Contacts: []Contact{{ID: "c1", Email: "v@example.com"}},
		Issues:   []Issue{{ID: "i1", Number: 4002, State: "closed", Source: "email", Requester: &Ref{ID: "c1"}, CreatedAt: t0}},
		Messages: map[string][]Message{"i1": {{ID: "m1", MessageHTML: "x", Timestamp: t0, Author: &MessageAuthor{Contact: &Ref{ID: "c1"}}}}},
	}
	im := &Importer{DB: db, Opt: Options{MediaStore: "fs", AllowMissingFiles: true}}
	rep, err := (&Syncer{DB: db, Imp: im, Opt: SyncOptions{CutoverEnd: cutover}}).Run(context.Background(), d)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.StatusSkipped != 1 {
		t.Fatalf("summary: %+v", rep.Summary)
	}
	if got := str(t, db, `SELECT cs.name FROM conversations c JOIN conversation_statuses cs ON cs.id = c.status_id WHERE c.id = $1`, conv); got != StatusOpen {
		t.Fatalf("status should stay open, got %q", got)
	}
}
