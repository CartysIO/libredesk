package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/media"
	localfs "github.com/abhinavxd/libredesk/internal/media/stores/localfs"
	"github.com/abhinavxd/libredesk/internal/pylonimport"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
	"github.com/zerodha/logf"
)

// TestRunPylonPurgeNonImportedAfterImport exercises runPylonPurgeNonImported, the hook used
// when --pylon-import runs with --pylon-purge-non-imported after Importer.Run.
func TestRunPylonPurgeNonImportedAfterImport(t *testing.T) {
	t0 := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	db := testutil.NewDB(t, "cmd_pylon_purge_hook")

	var inbox, cust int
	db.Get(&inbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Support', 'email', FALSE) RETURNING id`)
	db.Get(&cust, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'vayco@example.com', 'Vayco') RETURNING id`)

	var native int
	if err := db.Get(&native, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, subject, created_at, last_message_at)
		VALUES ('888', $1, $2, (SELECT id FROM conversation_statuses WHERE name = 'Open'), 'Native', $3, $3) RETURNING id`,
		cust, inbox, t0); err != nil {
		t.Fatal(err)
	}

	d := pylonimport.Data{
		Users:    []pylonimport.User{{ID: "u1", Name: "eran", Email: "eran@clyr.io"}},
		Accounts: []pylonimport.Account{{ID: "a1", Name: "Vayco"}},
		Contacts: []pylonimport.Contact{{ID: "c1", Name: "V", Email: "vayco@example.com", Account: &pylonimport.Ref{ID: "a1"}}},
		Issues: []pylonimport.Issue{{
			ID: "i2", Number: 5001, Title: "Pylon", State: "open", Source: "email",
			Account: &pylonimport.Ref{ID: "a1"}, Requester: &pylonimport.Ref{ID: "c1"},
			Assignee: &pylonimport.Ref{ID: "u1"}, CreatedAt: t0,
		}},
		Messages: map[string][]pylonimport.Message{
			"i2": {{ID: "m1", MessageHTML: "hi", Timestamp: t0, Author: &pylonimport.MessageAuthor{Contact: &pylonimport.Ref{ID: "c1"}}}},
		},
	}
	mm := testPylonImportMedia(t, db)
	im := &pylonimport.Importer{DB: db, Media: mm, Opt: pylonimport.Options{ReferenceFloor: 4000, MediaStore: "fs"}}
	if _, err := im.Run(context.Background(), d); err != nil {
		t.Fatal(err)
	}

	fs, err := stuffbin.NewLocalFS("/", filepath.FromSlash("../i18n/en-US.json:i18n/en-US.json"))
	if err != nil {
		t.Fatal(err)
	}
	setPylonPurgeTestConfig(t)
	settings := initSettings(db)

	ctx := context.Background()
	if err := runPylonPurgeNonImported(ctx, db, fs, settings, true); err != nil {
		t.Fatalf("dry-run purge: %v", err)
	}
	if n := countConversations(t, db, native); n != 1 {
		t.Fatalf("dry-run must not delete native conversation, count %d", n)
	}
	if err := runPylonPurgeNonImported(ctx, db, fs, settings, false); err != nil {
		t.Fatalf("purge: %v", err)
	}
	if n := countConversations(t, db, native); n != 0 {
		t.Fatalf("native conversation should be removed, count %d", n)
	}
	if n := countConversationsWithMeta(t, db, pylonimport.MetaIssueID); n != 1 {
		t.Fatalf("imported conversation should remain, count %d", n)
	}
}

func testPylonImportMedia(t *testing.T, db *sqlx.DB) *media.Manager {
	t.Helper()
	lo := logf.New(logf.Opts{})
	store, err := localfs.New(localfs.Opts{
		UploadPath: t.TempDir(), UploadURI: media.PublicURI,
		RootURL: func() string { return "http://x" }, SigningKey: strings.Repeat("k", 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	mm, err := media.New(media.Opts{Store: store, Lo: &lo, DB: db, I18n: testutil.NewI18n(t), RootURL: func() string { return "http://x" }})
	if err != nil {
		t.Fatal(err)
	}
	return mm
}

func setPylonPurgeTestConfig(t *testing.T) {
	t.Helper()
	keys := map[string]any{
		"upload.provider":       "fs",
		"upload.fs.upload_path": t.TempDir(),
		"app.lang":              "en-US",
		"app.root_url":          "http://localhost",
		"app.encryption_key":    "01234567890123456789012345678901",
	}
	for k, v := range keys {
		if err := ko.Set(k, v); err != nil {
			t.Fatalf("ko.Set %s: %v", k, err)
		}
		t.Cleanup(func() { ko.Delete(k) })
	}
}

func countConversations(t *testing.T, db interface {
	Get(dest any, query string, args ...any) error
}, id int) int {
	t.Helper()
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM conversations WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	return n
}

func countConversationsWithMeta(t *testing.T, db interface {
	Get(dest any, query string, args ...any) error
}, metaKey string) int {
	t.Helper()
	var n int
	if err := db.Get(&n, `SELECT COUNT(*) FROM conversations WHERE meta ? $1`, metaKey); err != nil {
		t.Fatal(err)
	}
	return n
}
