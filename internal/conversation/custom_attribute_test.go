package conversation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/abhinavxd/libredesk/internal/ws"
	"github.com/zerodha/logf"
)

// TestSetConversationCustomAttributeIfEmpty checks the AI classify write merges one key and never overwrites a value.
func TestSetConversationCustomAttributeIfEmpty(t *testing.T) {
	db := testutil.NewDB(t, "conversation_attr_if_empty")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), settingsStore: utcSettings{}, wsHub: ws.NewHub(&lo, nil)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	var inbox, contact int
	if err := db.Get(&inbox, `INSERT INTO inboxes (name, channel) VALUES ('Mail', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contact, `INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Ada', '', 'ada@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	var id int
	var uuid string
	if err := m.q.InsertConversation.QueryRow(contact, models.StatusOpen, inbox, "", time.Now(), models.StatusOpen, "", false, []byte("{}"), []byte(`{"plan": "pro", "question_type": ""}`), time.Time{}, 0, "", 0).Scan(&id, &uuid); err != nil {
		t.Fatalf("inserting conversation: %v", err)
	}

	attrs := func() map[string]any {
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

	// An empty string counts as unset.
	set, err := m.SetConversationCustomAttributeIfEmpty(uuid, "question_type", "bug")
	if err != nil || !set {
		t.Fatalf("first set: set=%v err=%v", set, err)
	}
	if got := attrs(); got["question_type"] != "bug" || got["plan"] != "pro" {
		t.Fatalf("want question_type=bug and plan kept, got %v", got)
	}

	set, err = m.SetConversationCustomAttributeIfEmpty(uuid, "question_type", "billing")
	if err != nil || set {
		t.Fatalf("second set must be skipped: set=%v err=%v", set, err)
	}
	if got := attrs(); got["question_type"] != "bug" {
		t.Fatalf("existing value overwritten: %v", got)
	}

	set, err = m.SetConversationCustomAttributeIfEmpty("00000000-0000-0000-0000-000000000000", "question_type", "bug")
	if err != nil || set {
		t.Fatalf("unknown conversation: set=%v err=%v", set, err)
	}
}
