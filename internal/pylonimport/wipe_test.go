package pylonimport

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/media"
	localfs "github.com/abhinavxd/libredesk/internal/media/stores/localfs"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

func TestWipeRemovesConversationsAndMessageMedia(t *testing.T) {
	db := testutil.NewDB(t, "pylon_wipe")
	lo := logf.New(logf.Opts{})
	dir := t.TempDir()
	store, err := localfs.New(localfs.Opts{UploadPath: dir, UploadURI: media.PublicURI, RootURL: func() string { return "http://x" }, SigningKey: strings.Repeat("k", 32)})
	if err != nil {
		t.Fatal(err)
	}
	mm, err := media.New(media.Opts{Store: store, Lo: &lo, DB: db, I18n: testutil.NewI18n(t), RootURL: func() string { return "http://x" }})
	if err != nil {
		t.Fatal(err)
	}

	seedConversationWithAttachment(t, db, mm)

	w := &Wiper{DB: db, Opt: WipeOptions{Confirm: true}, Media: mm}
	rep, err := w.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Conversations == 0 {
		t.Fatal("expected seeded conversations in report")
	}
	if rep.MediaStoreDeletes != 1 {
		t.Fatalf("MediaStoreDeletes = %d, want 1", rep.MediaStoreDeletes)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM conversations`); n != 0 {
		t.Fatalf("conversations left: %d", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM conversation_messages`); n != 0 {
		t.Fatalf("messages left: %d", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM media WHERE model_type = 'messages'`); n != 0 {
		t.Fatalf("message media left: %d", n)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM users WHERE email = 'wipe@example.com'`); n != 1 {
		t.Fatalf("contact should remain: count %d", n)
	}
	seq := str(t, db, `SELECT last_value FROM conversation_reference_number_sequence`)
	if seq != "100" {
		t.Fatalf("sequence last_value = %q, want 100", seq)
	}
}

func TestWipeDryRunLeavesData(t *testing.T) {
	db := testutil.NewDB(t, "pylon_wipe_dry")
	lo := logf.New(logf.Opts{})
	store, err := localfs.New(localfs.Opts{UploadPath: t.TempDir(), UploadURI: media.PublicURI, RootURL: func() string { return "http://x" }, SigningKey: strings.Repeat("k", 32)})
	if err != nil {
		t.Fatal(err)
	}
	mm, err := media.New(media.Opts{Store: store, Lo: &lo, DB: db, I18n: testutil.NewI18n(t), RootURL: func() string { return "http://x" }})
	if err != nil {
		t.Fatal(err)
	}
	seedConversationWithAttachment(t, db, mm)

	var buf bytes.Buffer
	w := &Wiper{DB: db, Opt: WipeOptions{Confirm: false}, Out: &buf}
	rep, err := w.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !rep.DryRun {
		t.Fatal("expected dry run")
	}
	rep.Print(&buf)
	if n := count(t, db, `SELECT COUNT(*) FROM conversations`); n != 1 {
		t.Fatalf("dry run deleted conversations: %d left", n)
	}
	if !strings.Contains(buf.String(), "DRY RUN") {
		t.Fatalf("output %q", buf.String())
	}
}

func seedConversationWithAttachment(t *testing.T, db *sqlx.DB, mm *media.Manager) int {
	t.Helper()
	var inbox, contact int
	db.Get(&inbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Support', 'email', FALSE) RETURNING id`)
	db.Get(&contact, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'wipe@example.com', 'Wipe') RETURNING id`)
	var conv int
	if err := db.Get(&conv, `INSERT INTO conversations (contact_id, inbox_id, status_id, subject)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE name = 'Open'), 'test') RETURNING id`, contact, inbox); err != nil {
		t.Fatal(err)
	}
	var msg int
	if err := db.Get(&msg, `INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type, content)
		VALUES ('incoming', 'received', $1, $2, 'contact', 'hi') RETURNING id`, conv, contact); err != nil {
		t.Fatal(err)
	}
	body := strings.NewReader("hello")
	_, err := mm.UploadAndInsert("note.txt", "text/plain", "", null.StringFrom(mmodels.ModelMessages), null.IntFrom(msg),
		body, 5, null.StringFrom(mmodels.DispositionAttachment), []byte(`{}`), true)
	if err != nil {
		t.Fatal(err)
	}
	return conv
}
