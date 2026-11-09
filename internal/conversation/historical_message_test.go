package conversation

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/dbutil"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

type noopMessageMedia struct{ mediaStore }

func (noopMessageMedia) LinkMessageMediaTx(*sqlx.Tx, int, []mmodels.Media, []string, int) error {
	return nil
}

func TestInsertHistoricalAgentMessageMidThreadOrder(t *testing.T) {
	db := testutil.NewDB(t, "historical_message_order")
	lo := logf.New(logf.Opts{})
	m := &Manager{lo: &lo, i18n: testutil.NewI18n(t), db: db, mediaStore: noopMessageMedia{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	var contactID, agentID, inboxID, statusID int
	var convUUID string
	if err := db.QueryRow(`INSERT INTO users (type, email, first_name) VALUES ('contact', 'hist@c.test', 'C') RETURNING id`).Scan(&contactID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO users (type, email, first_name) VALUES ('agent', 'hist-a@c.test', 'A') RETURNING id`).Scan(&agentID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`INSERT INTO inboxes (channel, name) VALUES ('email', 'hist') RETURNING id`).Scan(&inboxID); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT id FROM conversation_statuses WHERE category = 'open' LIMIT 1`).Scan(&statusID); err != nil {
		t.Fatal(err)
	}
	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t1 := time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	if err := db.QueryRow(`INSERT INTO conversations (contact_id, inbox_id, status_id, subject, created_at)
		VALUES ($1, $2, $3, 'hist', $4) RETURNING uuid`, contactID, inboxID, statusID, t0).Scan(&convUUID); err != nil {
		t.Fatal(err)
	}
	var convID int
	if err := db.Get(&convID, `SELECT id FROM conversations WHERE uuid = $1`, convUUID); err != nil {
		t.Fatal(err)
	}

	for _, row := range []struct {
		content string
		at      time.Time
	}{
		{"first", t0},
		{"last", t2},
	} {
		if _, err := db.Exec(`INSERT INTO conversation_messages
			(created_at, updated_at, conversation_id, type, status, content, text_content, content_type, sender_id, sender_type, private, meta)
			VALUES ($1, $1, $2, 'outgoing', 'sent', $3, $3, 'text', $4, 'agent', false, '{}')`,
			row.at, convID, row.content, agentID); err != nil {
			t.Fatal(err)
		}
	}

	inserted, err := m.InsertHistoricalAgentMessage(nil, agentID, convUUID, "middle", false, t1, map[string]any{"pylon_message_id": "pylon-mid"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !inserted.IsMigrationBackfill() {
		t.Fatal("expected migration_backfill meta")
	}
	if !inserted.CreatedAt.Equal(t1) {
		t.Fatalf("created_at = %v, want %v", inserted.CreatedAt, t1)
	}

	var lastMessageAt time.Time
	if err := db.Get(&lastMessageAt, `SELECT last_message_at FROM conversations WHERE id = $1`, convID); err != nil {
		t.Fatal(err)
	}
	if !lastMessageAt.Equal(t2) {
		t.Fatalf("last_message_at = %v, want %v", lastMessageAt, t2)
	}

	var bodies []string
	if err := db.Select(&bodies, `SELECT text_content FROM conversation_messages WHERE conversation_id = $1 ORDER BY created_at ASC, id ASC`, convID); err != nil {
		t.Fatal(err)
	}
	want := []string{"first", "middle", "last"}
	if len(bodies) != len(want) {
		t.Fatalf("messages = %v", bodies)
	}
	for i := range want {
		if bodies[i] != want[i] {
			t.Fatalf("order[%d] = %q, want %q; all=%v", i, bodies[i], want[i], bodies)
		}
	}
}
