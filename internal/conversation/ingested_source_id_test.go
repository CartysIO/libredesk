package conversation

import (
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/lib/pq"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestMessageExistsAfterConversationDelete(t *testing.T) {
	db := testutil.NewDB(t, "ingested_source_id")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t), mediaStore: noopMessageMedia{}}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	var inboxID, contactID, convID int
	var convUUID string
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Email', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, email) VALUES ('contact', 'Amazon', 'spam@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := m.insertConversationRow(m.q.InsertConversation, 0, contactID, inboxID, "", time.Now(), "promo", "", false, []byte("{}"), []byte("{}"), time.Time{}, 0).Scan(&convID, &convUUID); err != nil {
		t.Fatal(err)
	}

	const sourceID = "<dedup-test@example.com>"
	msg := models.Message{
		Type:             models.MessageIncoming,
		Status:           models.MessageStatusReceived,
		ConversationID:   convID,
		ConversationUUID: convUUID,
		Content:          "hello",
		TextContent:      "hello",
		SenderID:         contactID,
		SenderType:       models.SenderTypeContact,
		SourceID:         null.StringFrom(sourceID),
	}
	if err := m.InsertMessage(&msg); err != nil {
		t.Fatal(err)
	}

	exists, err := m.MessageExists(sourceID)
	if err != nil || !exists {
		t.Fatalf("before delete: exists=%v err=%v", exists, err)
	}

	if err := m.DeleteConversation(convUUID); err != nil {
		t.Fatal(err)
	}

	exists, err = m.MessageExists(sourceID)
	if err != nil || !exists {
		t.Fatalf("after delete: exists=%v err=%v, want tombstone", exists, err)
	}

	if _, err := m.messageExistsBySourceID([]string{sourceID}); err == nil {
		t.Fatal("threading lookup should not find deleted conversation message")
	}
}

func TestRecordIngestedSourceIDsForConversationsBackfillOnDelete(t *testing.T) {
	db := testutil.NewDB(t, "ingested_source_id_backfill")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	var inboxID, contactID, convID int
	var convUUID string
	if err := db.Get(&inboxID, `INSERT INTO inboxes (name, channel) VALUES ('Email', 'email') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&contactID, `INSERT INTO users (type, first_name, email) VALUES ('contact', 'A', 'a@example.com') RETURNING id`); err != nil {
		t.Fatal(err)
	}
	if err := m.insertConversationRow(m.q.InsertConversation, 0, contactID, inboxID, "", time.Now(), "t", "", false, []byte("{}"), []byte("{}"), time.Time{}, 0).Scan(&convID, &convUUID); err != nil {
		t.Fatal(err)
	}

	const sourceID = "legacy-msg@example.com"
	db.MustExec(`INSERT INTO conversation_messages (type, status, conversation_id, content, text_content, sender_id, sender_type, source_id)
		VALUES ('incoming', 'received', $1, 'x', 'x', $2, 'contact', $3)`, convID, contactID, sourceID)
	db.MustExec(`DELETE FROM ingested_message_source_ids WHERE source_id = $1`, sourceID)

	if err := m.DeleteConversation(convUUID); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM ingested_message_source_ids WHERE source_id = $1`, sourceID); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("tombstone count = %d, want 1", n)
	}
}

func TestMessageIngestedExistsBySourceIDQuery(t *testing.T) {
	db := testutil.NewDB(t, "ingested_source_id_query")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}

	const id = "only-tombstone@example.com"
	db.MustExec(`INSERT INTO ingested_message_source_ids (source_id) VALUES ($1)`, id)

	var exists bool
	if err := m.q.MessageIngestedExistsBySourceID.QueryRowx(id).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Fatal("expected tombstone-only source id to exist")
	}

	if _, err := m.q.RecordIngestedSourceIDsForConversations.Exec(pq.Array([]string{"00000000-0000-0000-0000-000000000001"})); err != nil {
		t.Fatal(err)
	}
}

func TestMessageExistsTrimsSourceID(t *testing.T) {
	db := testutil.NewDB(t, "ingested_source_id_trim")
	lo := logf.New(logf.Opts{})
	m := &Manager{db: db, lo: &lo, i18n: testutil.NewI18n(t)}
	if err := dbutil.ScanSQLFile("queries.sql", &m.q, db, efs); err != nil {
		t.Fatal(err)
	}
	db.MustExec(`INSERT INTO ingested_message_source_ids (source_id) VALUES ('trim-me@example.com')`)

	exists, err := m.MessageExists("  trim-me@example.com  ")
	if err != nil || !exists {
		t.Fatalf("trimmed lookup: exists=%v err=%v", exists, err)
	}
}
