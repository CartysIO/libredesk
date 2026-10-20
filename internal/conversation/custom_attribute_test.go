package conversation

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/zerodha/logf"
)

// TestAIClassifyStoreQueries checks the two queries behind the AI classify action: the merge-safe
// attribute write never overwrites a value, and the first incoming messages come back oldest first.
func TestAIClassifyStoreQueries(t *testing.T) {
	db := testutil.NewDB(t, "conversation_ai_classify")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), settingsStore: utcSettings{}, wsHub: ws.NewHub(&lo, nil)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	var inbox, contact, agent int
	if err := db.Get(&inbox, `INSERT INTO inboxes (name, channel) VALUES ('Mail', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contact, `INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Ada', '', 'ada@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&agent, `INSERT INTO users (type, first_name, last_name, email) VALUES ('agent', 'Eve', '', 'eve@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	newConv := func(attrs string) (int, string) {
		var id int
		var uuid string
		if err := m.q.InsertConversation.QueryRow(contact, models.StatusOpen, inbox, "", time.Now(), models.StatusOpen, "", false, []byte("{}"), []byte(attrs), time.Time{}, 0, "", 0).Scan(&id, &uuid); err != nil {
			t.Fatalf("inserting conversation: %v", err)
		}
		return id, uuid
	}
	attrs := func(uuid string) map[string]any {
		var raw []byte
		if err := db.Get(&raw, `SELECT custom_attributes FROM conversations WHERE uuid = $1`, uuid); err != nil {
			t.Fatal(err)
		}
		out := map[string]any{}
		if err := json.Unmarshal(raw, &out); err != nil {
			t.Fatal(err)
		}
		return out
	}
	set := func(uuid string, value any) bool {
		t.Helper()
		ok, err := m.SetConversationCustomAttributeIfEmpty(uuid, "question_type", value)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}

	// A blank string counts as unset; other keys are kept.
	_, single := newConv(`{"plan": "pro", "question_type": "  "}`)
	if !set(single, "Bug") {
		t.Fatal("blank value must count as unset")
	}
	if got := attrs(single); got["question_type"] != "Bug" || got["plan"] != "pro" {
		t.Fatalf("want question_type=Bug and plan kept, got %v", got)
	}
	if set(single, "User Error") || attrs(single)["question_type"] != "Bug" {
		t.Fatal("an existing value must not be overwritten")
	}

	// Multi-select values are stored as a JSON array; an empty array counts as unset.
	convID, multi := newConv(`{"question_type": []}`)
	if !set(multi, []string{"Bug", "User Error"}) {
		t.Fatal("empty array must count as unset")
	}
	if got := attrs(multi)["question_type"]; !reflect.DeepEqual(got, []any{"Bug", "User Error"}) {
		t.Fatalf("want a JSON array, got %#v", got)
	}
	if set(multi, []string{"Customer Love"}) {
		t.Fatal("an existing array must not be overwritten")
	}

	if set("00000000-0000-0000-0000-000000000000", "Bug") {
		t.Fatal("unknown conversation must report not set")
	}

	// Oldest public customer messages first; agent replies and private notes are left out.
	base := time.Now().Add(-time.Hour)
	insert := func(typ, senderType string, sender int, private bool, text string, offset time.Duration) {
		if _, err := db.Exec(`INSERT INTO conversation_messages (created_at, type, status, private, conversation_id, content_type, content, text_content, sender_id, sender_type)
			VALUES ($1, $2, 'received', $3, $4, 'text', $5, $5, $6, $7)`, base.Add(offset), typ, private, convID, text, sender, senderType); err != nil {
			t.Fatal(err)
		}
	}
	insert("incoming", "contact", contact, false, "third", 3*time.Minute)
	insert("incoming", "contact", contact, false, "first", time.Minute)
	insert("outgoing", "agent", agent, false, "agent reply", 2*time.Minute)
	insert("outgoing", "agent", agent, true, "private note", 150*time.Second)
	insert("incoming", "contact", contact, false, "second", 2*time.Minute)
	insert("incoming", "contact", contact, false, "fourth", 4*time.Minute)

	msgs, err := m.GetFirstIncomingMessages(multi, 3)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, msg := range msgs {
		if msg.SenderType != models.SenderTypeContact {
			t.Fatalf("non-contact message returned: %+v", msg)
		}
		got = append(got, msg.TextContent)
	}
	if want := []string{"first", "second", "third"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
