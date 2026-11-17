package pylonimport

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestSimulateImportedSLAMet(t *testing.T) {
	db := testutil.NewDB(t, "pylon_sla")
	var chatInbox, emailInbox int
	db.Get(&emailInbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('E', 'email', FALSE) RETURNING id`)
	db.Get(&chatInbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('C', 'livechat', FALSE) RETURNING id`)
	var slaID int
	db.Get(&slaID, `SELECT id FROM sla_policies ORDER BY id LIMIT 1`)
	var contact, statusID int
	db.Get(&contact, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'sla@test.io', 'T') RETURNING id`)
	db.Get(&statusID, `SELECT id FROM conversation_statuses WHERE name = 'Resolved'`)
	t0 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	fr := t0.Add(30 * time.Minute)
	res := t0.Add(2 * time.Hour)
	var convID int
	db.Get(&convID, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, created_at, first_reply_at, resolved_at, last_resolved_at)
		VALUES ('900', $1, $2, $3, $4, $5, $6, $6) RETURNING id`, contact, emailInbox, statusID, t0, fr, res)

	im := &Importer{DB: db, emailInboxID: emailInbox, chatInboxID: chatInbox, Opt: Options{SLAPolicyID: slaID}}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	is := Issue{
		CreatedAt:         t0,
		FirstResponseTime: &fr,
		ResolutionTime:    &res,
		State:             "closed",
	}
	if err := im.simulateImportedSLA(tx, convID, is, t0, "resolved"); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var n int
	db.Get(&n, `SELECT COUNT(*) FROM applied_slas WHERE conversation_id = $1 AND status = 'met'`, convID)
	if n != 1 {
		t.Fatalf("applied_slas = %d", n)
	}
}