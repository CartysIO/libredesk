package pylonimport

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/abhinavxd/libredesk/internal/media"
	localfs "github.com/abhinavxd/libredesk/internal/media/stores/localfs"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func TestPurgeNonImportedRemovesOnlyUntaggedConversations(t *testing.T) {
	db := testutil.NewDB(t, "pylon_purge_non_imported")

	nativeID := seedBareConversation(t, db, "native", nil)
	importedMeta, _ := json.Marshal(map[string]string{MetaIssueID: "issue-1"})
	importedID := seedBareConversation(t, db, "imported", importedMeta)
	mergedMeta, _ := json.Marshal(map[string]string{MetaMergedIssueID: "issue-2"})
	mergedID := seedBareConversation(t, db, "merged", mergedMeta)

	p := &NonImportedPurger{DB: db, Opt: PurgeNonImportedOptions{Confirm: true}}
	rep, err := p.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Conversations != 1 {
		t.Fatalf("Conversations = %d, want 1", rep.Conversations)
	}
	assertConversationExists(t, db, importedID)
	assertConversationExists(t, db, mergedID)
	if n := count(t, db, `SELECT COUNT(*) FROM conversations WHERE id = $1`, nativeID); n != 0 {
		t.Fatalf("native conversation still present")
	}
}

func TestPurgeNonImportedDryRunLeavesData(t *testing.T) {
	db := testutil.NewDB(t, "pylon_purge_non_imported_dry")
	seedBareConversation(t, db, "native", nil)

	var buf bytes.Buffer
	p := &NonImportedPurger{DB: db, Opt: PurgeNonImportedOptions{Confirm: false}, Out: &buf}
	rep, err := p.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !rep.DryRun || rep.Conversations != 1 {
		t.Fatalf("rep = %+v", rep)
	}
	rep.Print(&buf)
	if n := count(t, db, `SELECT COUNT(*) FROM conversations`); n != 1 {
		t.Fatalf("conversations left: %d", n)
	}
	if !strings.Contains(buf.String(), "DRY RUN") {
		t.Fatalf("output %q", buf.String())
	}
	if !strings.Contains(buf.String(), "#") || !strings.Contains(buf.String(), "native") {
		t.Fatalf("expected listed conversation in output %q", buf.String())
	}
}

func TestPurgeNonImportedDeletesAttachments(t *testing.T) {
	db := testutil.NewDB(t, "pylon_purge_non_imported_media")
	lo := logf.New(logf.Opts{})
	store, err := localfs.New(localfs.Opts{UploadPath: t.TempDir(), UploadURI: media.PublicURI, RootURL: func() string { return "http://x" }, SigningKey: strings.Repeat("k", 32)})
	if err != nil {
		t.Fatal(err)
	}
	mm, err := media.New(media.Opts{Store: store, Lo: &lo, DB: db, I18n: testutil.NewI18n(t), RootURL: func() string { return "http://x" }})
	if err != nil {
		t.Fatal(err)
	}
	nativeID := seedConversationWithAttachment(t, db, mm)
	importedMeta, _ := json.Marshal(map[string]string{MetaIssueID: "x"})
	seedBareConversation(t, db, "imported", importedMeta)

	p := &NonImportedPurger{DB: db, Opt: PurgeNonImportedOptions{Confirm: true}, Media: mm}
	rep, err := p.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rep.MediaStoreDeletes != 1 {
		t.Fatalf("MediaStoreDeletes = %d, want 1", rep.MediaStoreDeletes)
	}
	if n := count(t, db, `SELECT COUNT(*) FROM conversations WHERE id = $1`, nativeID); n != 0 {
		t.Fatalf("native conversation still present")
	}
	if n := count(t, db, `SELECT COUNT(*) FROM media WHERE model_type = 'messages'`); n != 0 {
		t.Fatalf("message media left: %d", n)
	}
}

func seedBareConversation(t *testing.T, db *sqlx.DB, subject string, meta []byte) int {
	t.Helper()
	var inbox, contact int
	db.Get(&inbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Support', 'email', FALSE) RETURNING id`)
	db.Get(&contact, `INSERT INTO users (type, email, first_name) VALUES ('contact', $1, 'C') RETURNING id`, subject+"@example.com")
	var conv int
	metaVal := []byte(`{}`)
	if meta != nil {
		metaVal = meta
	}
	if err := db.Get(&conv, `INSERT INTO conversations (contact_id, inbox_id, status_id, subject, meta)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE name = 'Open'), $3, $4) RETURNING id`,
		contact, inbox, subject, metaVal); err != nil {
		t.Fatal(err)
	}
	db.Exec(`INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type, content)
		VALUES ('incoming', 'received', $1, $2, 'contact', 'hi')`, conv, contact)
	return conv
}

func assertConversationExists(t *testing.T, db *sqlx.DB, id int) {
	t.Helper()
	if n := count(t, db, `SELECT COUNT(*) FROM conversations WHERE id = $1`, id); n != 1 {
		t.Fatalf("conversation %d missing", id)
	}
}
