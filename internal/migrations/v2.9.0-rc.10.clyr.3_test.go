package migrations

import (
	"encoding/json"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestV2_9_0_RC10_CLYR3WorkingSeconds(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_sla_working_sec")
	meta, _ := json.Marshal(map[string]any{
		"pylon_first_response_seconds": 518400,
		"pylon_resolution_seconds":     69300,
	})
	var contactID, inboxID, statusID, convID int
	db.Get(&contactID, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'migrate-fr@test.io', 'T') RETURNING id`)
	db.Get(&inboxID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('E', 'email', FALSE) RETURNING id`)
	db.Get(&statusID, `SELECT id FROM conversation_statuses LIMIT 1`)
	if err := db.Get(&convID, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, created_at, meta)
		VALUES ('1', $1, $2, $3, NOW(), $4::jsonb) RETURNING id`, contactID, inboxID, statusID, meta); err != nil {
		t.Fatal(err)
	}
	var slaID int
	db.Get(&slaID, `SELECT id FROM sla_policies LIMIT 1`)
	var appliedID int
	if err := db.Get(&appliedID, `INSERT INTO applied_slas (conversation_id, sla_policy_id, status, created_at)
		VALUES ($1, $2, 'met', NOW()) RETURNING id`, convID, slaID); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := V2_9_0_RC10_CLYR3(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}
	var fr, res *int
	if err := db.QueryRow(`SELECT first_response_working_sec, resolution_working_sec FROM applied_slas WHERE id = $1`, appliedID).
		Scan(&fr, &res); err != nil {
		t.Fatal(err)
	}
	if fr == nil || *fr != 518400 {
		t.Fatalf("first_response_working_sec = %v, want 518400", fr)
	}
	if res == nil || *res != 69300 {
		t.Fatalf("resolution_working_sec = %v, want 69300", res)
	}
}

func TestV2_9_0_RC10_CLYR3BackfillNestedPylonMeta(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_sla_working_sec_nested")
	meta, _ := json.Marshal(map[string]any{
		"pylon": map[string]any{
			"first_response_seconds": 3600,
			"resolution_seconds":     7200,
		},
	})
	var contactID, inboxID, statusID, convID int
	db.Get(&contactID, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'nested@test.io', 'T') RETURNING id`)
	db.Get(&inboxID, `INSERT INTO inboxes (name, channel, enabled) VALUES ('E', 'email', FALSE) RETURNING id`)
	db.Get(&statusID, `SELECT id FROM conversation_statuses LIMIT 1`)
	db.Get(&convID, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, created_at, meta)
		VALUES ('2', $1, $2, $3, NOW(), $4::jsonb)`, contactID, inboxID, statusID, meta)
	var slaID int
	db.Get(&slaID, `SELECT id FROM sla_policies LIMIT 1`)
	var appliedID int
	db.Get(&appliedID, `INSERT INTO applied_slas (conversation_id, sla_policy_id, status, created_at)
		VALUES ($1, $2, 'met', NOW()) RETURNING id`, convID, slaID)
	if err := V2_9_0_RC10_CLYR3(db, nil, nil); err != nil {
		t.Fatal(err)
	}
	var fr, res *int
	db.QueryRow(`SELECT first_response_working_sec, resolution_working_sec FROM applied_slas WHERE id = $1`, appliedID).
		Scan(&fr, &res)
	if fr == nil || *fr != 3600 {
		t.Fatalf("nested first_response_working_sec = %v", fr)
	}
	if res == nil || *res != 7200 {
		t.Fatalf("nested resolution_working_sec = %v", res)
	}
}