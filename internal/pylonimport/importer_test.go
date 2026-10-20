package pylonimport

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/abhinavxd/libredesk/internal/media"
	localfs "github.com/abhinavxd/libredesk/internal/media/stores/localfs"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

func pngBytes(t *testing.T) []byte {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func count(t *testing.T, db *sqlx.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.Get(&n, q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return n
}

func str(t *testing.T, db *sqlx.DB, q string, args ...any) string {
	t.Helper()
	var s string
	if err := db.Get(&s, q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	return s
}

// TestImportEndToEnd imports a small Pylon history twice into a database that already has conversations:
// one that collides with a Pylon number and two that duplicate Pylon issues.
func TestImportEndToEnd(t *testing.T) {
	db := testutil.NewDB(t, "pylon_import")
	lo := logf.New(logf.Opts{})
	store, err := localfs.New(localfs.Opts{UploadPath: t.TempDir(), UploadURI: media.PublicURI, RootURL: func() string { return "http://x" }, SigningKey: strings.Repeat("k", 32), Expiry: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	mm, err := media.New(media.Opts{Store: store, Lo: &lo, DB: db, I18n: testutil.NewI18n(t), RootURL: func() string { return "http://x" }})
	if err != nil {
		t.Fatal(err)
	}

	t0 := time.Date(2026, 10, 19, 15, 0, 0, 0, time.UTC)
	var inbox, eran, cust, other int
	db.Get(&inbox, `INSERT INTO inboxes (name, channel, enabled) VALUES ('Support', 'email', FALSE) RETURNING id`)
	db.Get(&eran, `INSERT INTO users (type, email, first_name) VALUES ('agent', 'eran@clyr.io', 'Eran') RETURNING id`)
	db.Get(&cust, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'vayco@example.com', 'Vayco') RETURNING id`)
	db.Get(&other, `INSERT INTO users (type, email, first_name) VALUES ('contact', 'other@example.com', 'Other') RETURNING id`)
	existing := func(ref string, contact int, subject string, at time.Time) int {
		var id int
		if err := db.Get(&id, `INSERT INTO conversations (reference_number, contact_id, inbox_id, status_id, subject, created_at, last_message_at)
			VALUES ($1, $2, $3, (SELECT id FROM conversation_statuses WHERE name = 'Open'), $4, $5, $5) RETURNING id`, ref, contact, inbox, subject, at); err != nil {
			t.Fatal(err)
		}
		return id
	}
	c100 := existing("100", other, "Unrelated", t0)
	c105 := existing("105", cust, "cancel account", t0)
	c106 := existing("106", cust, "Bank feed broken", t0.Add(time.Hour))
	c107 := existing("107", other, "Other thing", t0)
	// #106 had live activity after Pylon's last message: its status stays.
	db.MustExec(`UPDATE conversations SET last_message_at = $2 WHERE id = $1`, c106, t0.Add(48*time.Hour))
	db.MustExec(`INSERT INTO conversation_messages (type, status, conversation_id, sender_id, sender_type, content, source_id, created_at)
		VALUES ('incoming', 'received', $1, $2, 'contact', 'i need help canceling', 'abc@mail.gmail.com', $3)`, c105, cust, t0)

	img := "https://assets.usepylon.com/w%2F11111111-2222-3333-4444-555555555555-shot.png?Expires=9&Signature=s"
	doc := "https://assets.usepylon.com/w%2F11111111-2222-3333-4444-666666666666-report.pdf?Expires=9&Signature=s"
	files := map[string][]byte{FileKey(img): pngBytes(t), FileKey(doc): []byte("%PDF-1.4 test")}

	contact := &Ref{ID: "pc1", Email: "vayco@example.com"}
	d := Data{
		Users: []User{
			{ID: "u-eran", Name: "eran", Email: "eran@clyr.io"},
			{ID: "u-jaleel", Name: "jaleel", Email: "jaleel@clyr.io"},
			{ID: "u-bot", Name: "Backend Key"},
		},
		Accounts: []Account{
			{ID: "pa1", Name: "Vayco", ExternalIDs: []ExternalID{{Label: "clyr_team_id", ExternalID: "team_1"}}},
			{ID: "pa2", Name: "No Team"},
		},
		Contacts: []Contact{
			{ID: "pc1", Name: "Vayco Stays", Email: "Vayco@Example.com", Account: &Ref{ID: "pa1"}},
			{ID: "pc2", Name: "New Person", Email: "new@example.com", Account: &Ref{ID: "pa2"}},
		},
		Issues: []Issue{
			{ID: "i100", Number: 100, Title: "Old ticket", State: "closed", Account: &Ref{ID: "pa2"}, Requester: &Ref{ID: "pc2"},
				Assignee: &Ref{ID: "u-jaleel"}, Tags: []string{"PR-12", "loop-triaged"}, CreatedAt: t0.AddDate(-1, 0, 0),
				CustomFields: map[string]CustomField{"question_type": {Values: []string{"bug", "user_error"}}}},
			{ID: "i3855", Number: 3855, Title: "cancel account", State: "waiting_on_customer", Account: &Ref{ID: "pa1"}, Requester: &Ref{ID: "pc1"},
				Assignee: &Ref{ID: "u-eran"}, CreatedAt: t0},
			{ID: "i3856", Number: 3856, Title: "Screenshot", State: "on_hold", Account: &Ref{ID: "pa1"}, Requester: &Ref{ID: "pc1"},
				Assignee: &Ref{ID: "u-jaleel"}, CreatedAt: t0},
			{ID: "i3857", Number: 3857, Title: "RE: Bank feed broken", State: "closed", Requester: &Ref{ID: "pc1"}, CreatedAt: t0.Add(2 * time.Hour)},
		},
		Messages: map[string][]Message{
			"i100": {
				{ID: "m1", MessageHTML: "<p>help</p>", Timestamp: t0.AddDate(-1, 0, 0), Author: &MessageAuthor{Contact: &Ref{ID: "pc2", Email: "new@example.com"}},
					EmailInfo: &EmailInfo{FromEmail: "new@example.com", ToEmails: []string{"support@clyr.io"}, MessageID: "<m1@mail>"}},
				{ID: "m2", MessageHTML: "fixed", Timestamp: t0.AddDate(-1, 0, 0).Add(time.Hour), Author: &MessageAuthor{Name: "jaleel", User: &Ref{ID: "u-jaleel"}},
					EmailInfo: &EmailInfo{MessageID: "<m2@service.usepylon.com>"}},
				{ID: "m3", MessageHTML: "slack note", IsPrivate: true, Source: "slack", Timestamp: t0.AddDate(-1, 0, 0).Add(2 * time.Hour), Author: &MessageAuthor{Name: "Backend Key", User: &Ref{ID: "u-bot"}}},
				{ID: "m4", MessageHTML: "linked PR", IsPrivate: true, Source: "linear", Timestamp: t0.AddDate(-1, 0, 0).Add(3 * time.Hour), Author: &MessageAuthor{Name: "Linear"}},
			},
			"i3855": {
				{ID: "m5", MessageHTML: "i need help canceling", Timestamp: t0, Author: &MessageAuthor{Contact: contact}, EmailInfo: &EmailInfo{MessageID: "<abc@mail.gmail.com>"}},
				{ID: "m6", MessageHTML: "Can we help?", Timestamp: t0.Add(time.Minute), Author: &MessageAuthor{User: &Ref{ID: "u-eran"}}, EmailInfo: &EmailInfo{MessageID: "<m6@service.usepylon.com>"}},
			},
			"i3856": {
				{ID: "m7", MessageHTML: `<p>look</p><img src="` + strings.ReplaceAll(img, "&", "&amp;") + `">`, Timestamp: t0, Author: &MessageAuthor{Contact: contact},
					FileURLs: []string{img, doc}},
				// A reply that quotes the screenshot, under another signature.
				{ID: "m9", MessageHTML: `<p>thanks</p><blockquote><img src="` + strings.Replace(img, "Signature=s", "Signature=other", 1) + `"></blockquote>`,
					Timestamp: t0.Add(time.Hour), Author: &MessageAuthor{User: &Ref{ID: "u-eran"}}},
			},
			"i3857": {
				{ID: "m8", MessageHTML: "feed broken again", Timestamp: t0.Add(2 * time.Hour), Author: &MessageAuthor{Contact: contact}},
			},
		},
	}

	run := func(dry bool) *Report {
		im := &Importer{DB: db, Media: mm, Opt: Options{DryRun: dry, ReferenceFloor: 4000, MediaStore: "fs"},
			Files: func(u string) ([]byte, error) {
				if b, ok := files[FileKey(u)]; ok {
					return b, nil
				}
				return nil, os.ErrNotExist
			}}
		rep, err := im.Run(context.Background(), d)
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.FailedConversations) > 0 {
			t.Fatalf("failed: %v", rep.FailedConversations)
		}
		return rep
	}

	dry := run(true)
	if dry.Conversations != 2 || dry.Merged != 2 || count(t, db, `SELECT COUNT(*) FROM conversations`) != 4 {
		t.Fatalf("dry run: %+v", dry)
	}

	rep := run(false)
	if rep.Conversations != 2 || rep.Merged != 2 || rep.SkippedMessages != 1 || rep.Attachments != 2 || rep.InlineImages != 1 {
		t.Fatalf("import: %+v", rep)
	}

	// References: the colliding conversation moved to the sequence, duplicates took their Pylon numbers.
	if ref := str(t, db, `SELECT reference_number FROM conversations WHERE id = $1`, c100); ref != "4000" {
		t.Errorf("colliding #100 became #%s, want #4000", ref)
	}
	if str(t, db, `SELECT reference_number FROM conversations WHERE id = $1`, c105) != "3855" ||
		str(t, db, `SELECT reference_number FROM conversations WHERE id = $1`, c106) != "3857" ||
		str(t, db, `SELECT reference_number FROM conversations WHERE id = $1`, c107) != "4001" {
		t.Error("duplicates must take their Pylon numbers and the rest of the Pylon range must move to the sequence")
	}
	if prev := str(t, db, `SELECT meta->>'`+MetaPreviousReference+`' FROM conversations WHERE id = $1`, c100); prev != "100" {
		t.Errorf("previous reference = %q", prev)
	}
	if n := count(t, db, `SELECT nextval('conversation_reference_number_sequence')`); n != 4002 {
		t.Errorf("next reference = %d", n)
	}

	// The new #100: status, assignee, tags, question_type, authors.
	conv := count(t, db, `SELECT id FROM conversations WHERE reference_number = '100'`)
	if s := str(t, db, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id = c.status_id WHERE c.id = $1`, conv); s != StatusResolved {
		t.Errorf("status = %s", s)
	}
	if str(t, db, `SELECT u.first_name || ' ' || u.last_name || ' ' || u.enabled || ' ' || (u.password IS NULL) FROM conversations c JOIN users u ON u.id = c.assigned_user_id WHERE c.id = $1`, conv) != "Jaleel (former) false true" {
		t.Error("jaleel must be a disabled agent without a password")
	}
	if count(t, db, `SELECT COUNT(*) FROM user_roles ur JOIN users u ON u.id = ur.user_id WHERE u.custom_attributes ? '`+MetaUserID+`'`) != 0 {
		t.Error("created agents must have no role")
	}
	if tags := str(t, db, `SELECT string_agg(t.name, ',') FROM conversation_tags ct JOIN tags t ON t.id = ct.tag_id WHERE ct.conversation_id = $1`, conv); tags != "PR-12" {
		t.Errorf("tags = %s", tags)
	}
	if qt := str(t, db, `SELECT custom_attributes->>'question_type' FROM conversations WHERE id = $1`, conv); qt != "bug, user_error" {
		t.Errorf("question_type = %s", qt)
	}
	if got := str(t, db, `SELECT string_agg(type || ':' || sender_type || ':' || private, ',' ORDER BY created_at) FROM conversation_messages WHERE conversation_id = $1`, conv); got != "incoming:contact:false,outgoing:agent:false,outgoing:agent:true,outgoing:agent:true" {
		t.Errorf("messages = %s", got)
	}
	if got := str(t, db, `SELECT string_agg(COALESCE(source_id, '-'), ',' ORDER BY created_at) FROM conversation_messages WHERE conversation_id = $1`, conv); got != "m1@mail,m2@service.usepylon.com,-,-" {
		t.Errorf("source ids = %s", got)
	}
	if got := str(t, db, `SELECT u.first_name || ' ' || u.last_name FROM conversation_messages m JOIN users u ON u.id = m.sender_id WHERE m.meta->>'`+MetaMessageID+`' = 'm4'`); got != "Linear (Pylon bot)" {
		t.Errorf("integration author = %s", got)
	}
	if count(t, db, `SELECT COUNT(*) FROM conversations WHERE id = $1 AND first_reply_at IS NOT NULL AND resolved_at IS NOT NULL AND last_message = 'linked PR'`, conv) != 1 {
		t.Error("summary columns not set")
	}
	if count(t, db, `SELECT COUNT(*) FROM conversation_last_seen WHERE conversation_id = $1 AND user_id = $2`, conv, eran) != 1 {
		t.Error("history must count as read for enabled agents")
	}

	// Organizations and memberships.
	if org := str(t, db, `SELECT o.external_id FROM conversations c JOIN organizations o ON o.id = c.organization_id WHERE c.id = $1`, conv); org != "pa2" {
		t.Errorf("organization = %s", org)
	}
	if count(t, db, `SELECT COUNT(*) FROM users u JOIN organizations o ON o.id = u.organization_id WHERE u.id = $1 AND o.external_id = 'team_1'`, cust) != 1 {
		t.Error("existing contact must get its Pylon account as default organization")
	}

	// The merge: the duplicate inbound email is not copied again; the agent reply is.
	if got := str(t, db, `SELECT string_agg(content, '|' ORDER BY created_at, id) FROM conversation_messages WHERE conversation_id = $1`, c105); got != "i need help canceling|Can we help?" {
		t.Errorf("merged messages = %s", got)
	}
	if s := str(t, db, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id = c.status_id WHERE c.id = $1`, c105); s != StatusOnCustomer {
		t.Errorf("merged status = %s", s)
	}

	// Files: the inline image points at its media row, the PDF is an attachment.
	m7 := count(t, db, `SELECT id FROM conversation_messages WHERE meta->>'`+MetaMessageID+`' = 'm7'`)
	content := str(t, db, `SELECT content FROM conversation_messages WHERE id = $1`, m7)
	cid := str(t, db, `SELECT content_id FROM media WHERE model_type = 'messages' AND model_id = $1 AND disposition = 'inline'`, m7)
	if !strings.HasPrefix(cid, "ldsk-") || !strings.Contains(content, `src="cid:`+cid+`"`) {
		t.Errorf("inline image: cid %q, content %q", cid, content)
	}
	if name := str(t, db, `SELECT filename FROM media WHERE model_id = $1 AND disposition = 'attachment'`, m7); name != "report.pdf" {
		t.Errorf("attachment = %s", name)
	}
	if s := str(t, db, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id = c.status_id WHERE c.reference_number = '3856'`); s != StatusSnoozed {
		t.Errorf("on_hold = %s", s)
	}
	if count(t, db, `SELECT COUNT(*) FROM conversations WHERE reference_number = '3856' AND assigned_user_id IS NULL`) != 1 {
		t.Error("an open conversation must not be assigned to a disabled agent")
	}
	if quoted := str(t, db, `SELECT content FROM conversation_messages WHERE meta->>'`+MetaMessageID+`' = 'm9'`); !strings.Contains(quoted, `src="cid:`+cid+`"`) {
		t.Errorf("quoted image must point at the earlier message's media: %s", quoted)
	}
	if s := str(t, db, `SELECT s.name FROM conversations c JOIN conversation_statuses s ON s.id = c.status_id WHERE c.id = $1`, c106); s != StatusOpen {
		t.Errorf("a merge must keep the live status when Libredesk has newer activity, got %s", s)
	}

	// Second run: nothing new.
	before := count(t, db, `SELECT (SELECT COUNT(*) FROM conversations) * 1000000 + (SELECT COUNT(*) FROM conversation_messages) * 1000 + (SELECT COUNT(*) FROM media)`)
	// Eran, plus disabled agents for jaleel, Backend Key and the Linear integration.
	if n := count(t, db, `SELECT COUNT(*) FROM users WHERE type = 'agent'`); n != 4 {
		t.Errorf("agents = %d, want 4", n)
	}
	again := run(false)
	if again.Conversations != 0 || again.Merged != 0 || again.Messages != 0 || again.AlreadyImported != 4 || len(again.Renumbers) != 0 {
		t.Fatalf("second run: %+v", again)
	}
	if after := count(t, db, `SELECT (SELECT COUNT(*) FROM conversations) * 1000000 + (SELECT COUNT(*) FROM conversation_messages) * 1000 + (SELECT COUNT(*) FROM media)`); after != before {
		t.Errorf("second run changed row counts: %d -> %d", before, after)
	}
	if count(t, db, `SELECT COUNT(*) FROM users WHERE type = 'agent'`) != 4 {
		t.Error("second run must not create agents again")
	}
}
