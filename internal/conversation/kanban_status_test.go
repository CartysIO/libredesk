package conversation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

// TestShouldMoveToOnCustomer checks which sent messages move a conversation to "On customer".
func TestShouldMoveToOnCustomer(t *testing.T) {
	reply := func(mutate func(*models.Message)) models.Message {
		m := models.Message{Type: models.MessageOutgoing, SenderType: models.SenderTypeAgent, Meta: json.RawMessage(`{}`)}
		if mutate != nil {
			mutate(&m)
		}
		return m
	}
	tests := []struct {
		name    string
		message models.Message
		status  string
		want    bool
	}{
		{"reply from Open", reply(nil), models.StatusOpen, true},
		{"reply from On you", reply(nil), models.StatusOnYou, true},
		{"reply already On customer", reply(nil), models.StatusOnCustomer, false},
		{"reply on Snoozed", reply(nil), models.StatusSnoozed, false},
		{"reply on Resolved", reply(nil), models.StatusResolved, false},
		{"private note", reply(func(m *models.Message) { m.Private = true }), models.StatusOpen, false},
		{"automated reply", reply(func(m *models.Message) { m.Meta = json.RawMessage(`{"is_automated":true}`) }), models.StatusOpen, false},
		{"csat message", reply(func(m *models.Message) { m.Meta = json.RawMessage(`{"is_csat":true}`) }), models.StatusOpen, false},
		{"continuity email", reply(func(m *models.Message) { m.Meta = json.RawMessage(`{"continuity_email":true}`) }), models.StatusOpen, false},
		{"incoming message", reply(func(m *models.Message) { m.Type = models.MessageIncoming }), models.StatusOpen, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldMoveToOnCustomer(tt.message, tt.status); got != tt.want {
				t.Fatalf("shouldMoveToOnCustomer = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReOpenConversationQuery checks the target status of the re-open query after a customer message.
func TestReOpenConversationQuery(t *testing.T) {
	db := testutil.NewDB(t, "conversation_reopen_kanban")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), settingsStore: utcSettings{}}
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
	create := func(status string, replied bool) string {
		var id int
		var uuid string
		if err := m.q.InsertConversation.QueryRow(contact, status, inbox, "", time.Now(), status, "", false, []byte("{}"), []byte("{}"), time.Time{}, 0, "").Scan(&id, &uuid); err != nil {
			t.Fatalf("inserting conversation: %v", err)
		}
		if replied {
			db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, id)
		}
		return uuid
	}
	statusOf := func(uuid string) string {
		var s string
		if err := db.Get(&s, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id = c.status_id WHERE c.uuid = $1`, uuid); err != nil {
			t.Fatal(err)
		}
		return s
	}

	tests := []struct {
		name    string
		status  string
		replied bool
		want    string // "" means the query must not touch the conversation
	}{
		{"Snoozed without reply", models.StatusSnoozed, false, models.StatusOpen},
		{"Resolved without reply", models.StatusResolved, false, models.StatusOpen},
		{"Closed without reply", models.StatusClosed, false, models.StatusOpen},
		{"Snoozed with reply", models.StatusSnoozed, true, models.StatusOnYou},
		{"Resolved with reply", models.StatusResolved, true, models.StatusOnYou},
		{"Closed with reply", models.StatusClosed, true, models.StatusOnYou},
		{"On customer", models.StatusOnCustomer, true, models.StatusOnYou},
		{"On customer without recorded reply", models.StatusOnCustomer, false, models.StatusOnYou},
		{"Open", models.StatusOpen, true, ""},
		{"On you", models.StatusOnYou, true, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uuid := create(tt.status, tt.replied)
			var id int
			var got string
			err := m.q.ReOpenConversation.QueryRow(uuid).Scan(&id, &got)
			if tt.want == "" {
				if err == nil {
					t.Fatalf("query reopened a conversation in %q to %q", tt.status, got)
				}
				if s := statusOf(uuid); s != tt.status {
					t.Fatalf("status = %q, want unchanged %q", s, tt.status)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want || statusOf(uuid) != tt.want {
				t.Fatalf("returned %q, stored %q, want %q", got, statusOf(uuid), tt.want)
			}
		})
	}

	// If an admin removed "On you", the query falls back to Open.
	db.MustExec(`DELETE FROM conversations WHERE status_id = (SELECT id FROM conversation_statuses WHERE name = 'On you')`)
	db.MustExec(`DELETE FROM conversation_statuses WHERE name = 'On you'`)
	uuid := create(models.StatusResolved, true)
	var id int
	var got string
	if err := m.q.ReOpenConversation.QueryRow(uuid).Scan(&id, &got); err != nil || got != models.StatusOpen {
		t.Fatalf("fallback: status %q err %v, want Open", got, err)
	}
}

// TestUnsnoozeAllQuery checks that expired snoozes return to "On you" after an agent reply, else "Open".
func TestUnsnoozeAllQuery(t *testing.T) {
	db := testutil.NewDB(t, "conversation_unsnooze_kanban")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), settingsStore: utcSettings{}}
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
	create := func(replied bool, until string) string {
		var id int
		var uuid string
		if err := m.q.InsertConversation.QueryRow(contact, models.StatusSnoozed, inbox, "", time.Now(), "s", "", false, []byte("{}"), []byte("{}"), time.Time{}, 0, "").Scan(&id, &uuid); err != nil {
			t.Fatal(err)
		}
		db.MustExec(`UPDATE conversations SET snoozed_until = NOW() + $2::interval WHERE id = $1`, id, until)
		if replied {
			db.MustExec(`UPDATE conversations SET first_reply_at = NOW() WHERE id = $1`, id)
		}
		return uuid
	}
	unsnooze := func() map[string]string {
		rows, err := m.q.UnsnoozeAll.Queryx()
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		got := map[string]string{}
		for rows.Next() {
			var u, s string
			if err := rows.Scan(&u, &s); err != nil {
				t.Fatal(err)
			}
			got[u] = s
		}
		return got
	}

	replied, fresh, future := create(true, "-1 minute"), create(false, "-1 minute"), create(true, "1 hour")
	got := unsnooze()
	if len(got) != 2 || got[replied] != models.StatusOnYou || got[fresh] != models.StatusOpen {
		t.Fatalf("unsnoozed %v, want replied=%q fresh=%q and not the future snooze %s", got, models.StatusOnYou, models.StatusOpen, future)
	}

	// If an admin removed "On you", the query falls back to Open.
	db.MustExec(`DELETE FROM conversations WHERE status_id = (SELECT id FROM conversation_statuses WHERE name = 'On you')`)
	db.MustExec(`DELETE FROM conversation_statuses WHERE name = 'On you'`)
	fallback := create(true, "-1 minute")
	if got := unsnooze(); got[fallback] != models.StatusOpen {
		t.Fatalf("fallback: %v, want Open", got)
	}
}
