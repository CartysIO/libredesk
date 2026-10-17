package conversation

import (
	"slices"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/zerodha/logf"
)

// TestDeleteSpamConversations checks that marking a conversation as spam deletes it and the contact's
// other unresolved conversations, and keeps the contact's resolved ones and other contacts' conversations.
func TestDeleteSpamConversations(t *testing.T) {
	db := testutil.NewDB(t, "conversation_spam")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), settingsStore: utcSettings{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	var inbox, spammer, other int
	if err := db.Get(&inbox, `INSERT INTO inboxes (name, channel) VALUES ('Mail', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&spammer, `INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Spam', '', 'spam@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&other, `INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Ada', '', 'ada@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}

	insert := func(contactID int, status string) string {
		var id int
		var uuid string
		if err := m.q.InsertConversation.QueryRow(contactID, status, inbox, "", time.Now(), status, "", false, []byte("{}"), []byte("{}"), time.Time{}, 0, "").Scan(&id, &uuid); err != nil {
			t.Fatalf("inserting conversation: %v", err)
		}
		return uuid
	}
	// The marked conversation is deleted even when it is resolved.
	marked := insert(spammer, models.StatusResolved)
	open := insert(spammer, models.StatusOpen)
	snoozed := insert(spammer, models.StatusSnoozed)
	resolved := insert(spammer, models.StatusResolved)
	otherOpen := insert(other, models.StatusOpen)

	deleted, err := m.DeleteSpamConversations(marked, spammer)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, d := range deleted {
		if d.ID == 0 || d.ReferenceNumber == "" {
			t.Fatalf("deleted conversation has no id or reference number: %+v", d)
		}
		got = append(got, d.UUID)
	}
	want := []string{marked, open, snoozed}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("deleted %v, want %v", got, want)
	}

	var left []string
	if err := db.Select(&left, `SELECT uuid::text FROM conversations ORDER BY uuid`); err != nil {
		t.Fatal(err)
	}
	wantLeft := []string{resolved, otherOpen}
	slices.Sort(wantLeft)
	if !slices.Equal(left, wantLeft) {
		t.Fatalf("remaining conversations %v, want %v", left, wantLeft)
	}
}
