package conversation

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

// Conversations created by the Pylon import are history: time-trigger automations must not see them until they
// get a message after the import.
func TestTimeTriggersSkipPylonImports(t *testing.T) {
	db := testutil.NewDB(t, "pylon_time_triggers")
	var contactID, inboxID int
	db.Get(&contactID, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'c@example.com', 'C') RETURNING id`)
	db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Mail', 'email') RETURNING id`)
	importedAt := time.Now().UTC().Add(-10 * time.Minute)
	insert := func(meta string, lastMessageAt time.Time) int {
		var id int
		if err := db.Get(&id, `INSERT INTO conversations (contact_id, inbox_id, status_id, meta, last_message_at) VALUES ($1, $2, (SELECT id FROM conversation_statuses LIMIT 1), $3, $4) RETURNING id`, contactID, inboxID, meta, lastMessageAt); err != nil {
			t.Fatal(err)
		}
		return id
	}
	stamp := `"pylon_imported_at": "` + importedAt.Format(time.RFC3339) + `"`
	live := insert(`{}`, importedAt)
	insert(`{"pylon_issue_id": "abc", `+stamp+`}`, importedAt.Add(-time.Hour))
	merged := insert(`{"pylon_merged_issue_id": "def"}`, importedAt)
	replied := insert(`{"pylon_issue_id": "ghi", `+stamp+`}`, importedAt.Add(time.Minute))

	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	refs, err := m.GetConversationsCreatedAfter(time.Now().Add(-time.Hour), 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs) != 3 || refs[0].ID != live || refs[1].ID != merged || refs[2].ID != replied {
		t.Fatalf("refs = %+v, want the live, the merged and the replied-to imported conversation", refs)
	}
}
