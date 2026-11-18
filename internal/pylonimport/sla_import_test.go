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
	frSec, resSec := 1800, 7200
	is := Issue{
		CreatedAt:            t0,
		FirstResponseTime:    &fr,
		ResolutionTime:       &res,
		FirstResponseSeconds: &frSec,
		ResolutionSeconds:    &resSec,
		State:                "closed",
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
	var gotFR, gotRes *int
	if err := db.QueryRow(`SELECT first_response_working_sec, resolution_working_sec FROM applied_slas WHERE conversation_id = $1`, convID).
		Scan(&gotFR, &gotRes); err != nil {
		t.Fatal(err)
	}
	if gotFR == nil || *gotFR != frSec {
		t.Fatalf("first_response_working_sec = %v, want %d", gotFR, frSec)
	}
	if gotRes == nil || *gotRes != resSec {
		t.Fatalf("resolution_working_sec = %v, want %d", gotRes, resSec)
	}
}

func TestSimulateImportedSLARefreshWorkingSec(t *testing.T) {
	db := testutil.NewDB(t, "pylon_sla_refresh")
	var emailInbox int
	db.Get(&emailInbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('E', 'email', FALSE) RETURNING id`)
	var slaID int
	db.Get(&slaID, `SELECT id FROM sla_policies ORDER BY id LIMIT 1`)
	var contact, statusID, convID int
	db.Get(&contact, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'r@test.io', 'T') RETURNING id`)
	db.Get(&statusID, `SELECT id FROM conversation_statuses WHERE name = 'Resolved'`)
	t0 := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	db.Get(&convID, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, created_at)
		VALUES ('901', $1, $2, $3, $4) RETURNING id`, contact, emailInbox, statusID, t0)
	db.Exec(`INSERT INTO applied_slas (conversation_id, sla_policy_id, status, created_at)
		VALUES ($1, $2, 'met', $3)`, convID, slaID, t0)

	im := &Importer{DB: db, emailInboxID: emailInbox, Opt: Options{SLAPolicyID: slaID}}
	tx, _ := db.Beginx()
	frSec := 999
	is := Issue{CreatedAt: t0, FirstResponseSeconds: &frSec, State: "closed"}
	if err := im.simulateImportedSLA(tx, convID, is, t0, "resolved"); err != nil {
		t.Fatal(err)
	}
	tx.Commit()

	var got *int
	db.Get(&got, `SELECT first_response_working_sec FROM applied_slas WHERE conversation_id = $1`, convID)
	if got == nil || *got != frSec {
		t.Fatalf("first_response_working_sec = %v, want %d", got, frSec)
	}
}
