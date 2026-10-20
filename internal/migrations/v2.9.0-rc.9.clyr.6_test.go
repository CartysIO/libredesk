package migrations

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/abhinavxd/libredesk/internal/testutil"
)

func TestRepointClosedRefs(t *testing.T) {
	in := `[{"actions":[{"type":"set_status","value":["4"]},{"type":"set_priority","value":["4"]}],
		"groups":[{"rules":[
			{"field":"status","field_type":"conversation","operator":"equals","value":"4"},
			{"field":"previous_status","operator":"contains","value":"3, 4,14"},
			{"field":"status","field_type":"contact_custom_attribute","operator":"equals","value":"4"},
			{"field":"priority","operator":"equals","value":"4"}]}]},
		{"logic":"AND","rules":[
			{"model":"conversations","field":"status_id","operator":"in","value":"[\"1\",\"4\"]"},
			{"model":"conversation_statuses","field":"name","operator":"equals","value":"Closed"},
			{"model":"conversation_statuses","field":"id","operator":"equals","value":"4"},
			{"model":"conversations","field":"status_id","operator":"equals","value":"14"}]}]`
	want := `[{"actions":[{"type":"set_status","value":["3"]},{"type":"set_priority","value":["4"]}],
		"groups":[{"rules":[
			{"field":"status","field_type":"conversation","operator":"equals","value":"3"},
			{"field":"previous_status","operator":"contains","value":"3,3,14"},
			{"field":"status","field_type":"contact_custom_attribute","operator":"equals","value":"4"},
			{"field":"priority","operator":"equals","value":"4"}]}]},
		{"logic":"AND","rules":[
			{"model":"conversations","field":"status_id","operator":"in","value":"[\"1\",\"3\"]"},
			{"model":"conversation_statuses","field":"name","operator":"equals","value":"Resolved"},
			{"model":"conversation_statuses","field":"id","operator":"equals","value":"4"},
			{"model":"conversations","field":"status_id","operator":"equals","value":"14"}]}]`

	var got, exp any
	if err := json.Unmarshal([]byte(in), &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(want), &exp); err != nil {
		t.Fatal(err)
	}
	if !repointClosedRefs(got, "4", "3") {
		t.Fatal("repointClosedRefs reported no change")
	}
	if !reflect.DeepEqual(got, exp) {
		out, _ := json.Marshal(got)
		t.Fatalf("got %s", out)
	}
	if repointClosedRefs(got, "4", "3") {
		t.Fatal("second pass changed the value again")
	}
}

func TestV2_9_0_RC9_CLYR6RemovesClosedStatus(t *testing.T) {
	db := testutil.NewDB(t, "migration_clyr_remove_closed")
	// A database from before this migration has the Closed status.
	var closedID, resolvedID int
	db.QueryRow(`INSERT INTO conversation_statuses (name, category) VALUES ('Closed', 'resolved') RETURNING id`).Scan(&closedID)
	db.QueryRow(`SELECT id FROM conversation_statuses WHERE name = 'Resolved'`).Scan(&resolvedID)
	var inbox, contact int
	db.QueryRow(`INSERT INTO inboxes (name, channel) VALUES ('Mail', 'email') RETURNING id`).Scan(&inbox)
	db.QueryRow(`INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Ada', '', 'ada@example.com') RETURNING id`).Scan(&contact)
	if closedID == 0 || resolvedID == 0 || inbox == 0 || contact == 0 {
		t.Fatal("seeding failed")
	}
	var closedConv, resolvedBefore int
	db.QueryRow(`INSERT INTO conversations (contact_id, inbox_id, status_id, closed_at) VALUES ($1, $2, $3, '2026-01-02T00:00:00Z') RETURNING id`, contact, inbox, closedID).Scan(&closedConv)
	db.QueryRow(`INSERT INTO conversations (contact_id, inbox_id, status_id, closed_at, resolved_at, last_resolved_at)
		VALUES ($1, $2, $3, '2026-01-02T00:00:00Z', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z') RETURNING id`, contact, inbox, closedID).Scan(&resolvedBefore)

	c := fmt.Sprint(closedID)
	db.MustExec(`INSERT INTO automation_rules (name, type, rules) VALUES ('r', 'conversation_update', $1)`,
		`[{"groups":[{"rules":[{"field":"status","operator":"equals","value":"`+c+`"}]}],"actions":[{"type":"set_status","value":["`+c+`"]}]}]`)
	db.MustExec(`INSERT INTO macros (name, visibility, message_content, actions) VALUES ('m', 'all', '', $1)`,
		`[{"type":"set_status","value":["`+c+`"]}]`)
	db.MustExec(`INSERT INTO views (name, visibility, filters) VALUES ('v', 'all', $1)`,
		`[{"model":"conversations","field":"status_id","operator":"equals","value":"`+c+`"}]`)

	// Running it twice proves it is idempotent, which matters because a fresh install runs every migration.
	for range 2 {
		if err := V2_9_0_RC9_CLYR6(db, nil, nil); err != nil {
			t.Fatalf("running migration: %v", err)
		}
	}

	var n int
	db.Get(&n, `SELECT COUNT(*) FROM conversation_statuses WHERE name = 'Closed'`)
	if n != 0 {
		t.Fatal("Closed status still exists")
	}
	var times []struct {
		StatusID       int    `db:"status_id"`
		ResolvedAt     string `db:"resolved_at"`
		LastResolvedAt string `db:"last_resolved_at"`
	}
	if err := db.Select(&times, `SELECT status_id, to_char(resolved_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS resolved_at,
		to_char(last_resolved_at AT TIME ZONE 'UTC', 'YYYY-MM-DD') AS last_resolved_at FROM conversations WHERE id IN ($1, $2) ORDER BY id`, closedConv, resolvedBefore); err != nil {
		t.Fatal(err)
	}
	if len(times) != 2 {
		t.Fatalf("got %d conversations, want 2", len(times))
	}
	// A conversation that was only closed takes its close time; one resolved earlier keeps its resolve time.
	if times[0].StatusID != resolvedID || times[0].ResolvedAt != "2026-01-02" || times[0].LastResolvedAt != "2026-01-02" {
		t.Fatalf("closed conversation = %+v", times[0])
	}
	if times[1].StatusID != resolvedID || times[1].ResolvedAt != "2026-01-01" || times[1].LastResolvedAt != "2026-01-01" {
		t.Fatalf("resolved-then-closed conversation = %+v", times[1])
	}

	r := fmt.Sprint(resolvedID)
	for _, q := range []string{
		`SELECT COUNT(*) FROM automation_rules WHERE rules->0->'actions'->0->'value'->>0 = $1 AND rules->0->'groups'->0->'rules'->0->>'value' = $1`,
		`SELECT COUNT(*) FROM macros WHERE actions->0->'value'->>0 = $1`,
		`SELECT COUNT(*) FROM views WHERE filters->0->>'value' = $1`,
	} {
		if err := db.Get(&n, q, r); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("%s: got %d rows pointing to Resolved, want 1", q, n)
		}
	}
}
