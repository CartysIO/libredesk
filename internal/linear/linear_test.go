package linear

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	"github.com/jmoiron/sqlx"
	"github.com/zerodha/logf"
)

const testSecret = "lin_wh_test"

func sign(body []byte) string {
	mac := hmac.New(sha256.New, []byte(testSecret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	body := []byte(`{"type":"Comment"}`)
	if !VerifySignature(testSecret, body, sign(body)) {
		t.Fatal("good signature rejected")
	}
	for name, sig := range map[string]string{
		"wrong":   sign([]byte(`{"type":"Issue"}`)),
		"not hex": "zz",
		"empty":   "",
	} {
		if VerifySignature(testSecret, body, sig) {
			t.Errorf("%s signature accepted", name)
		}
	}
	if VerifySignature("", body, sign(body)) {
		t.Error("accepted with no secret")
	}
}

func TestFreshTimestamp(t *testing.T) {
	now := time.UnixMilli(1_760_000_000_000)
	cases := map[string]struct {
		ms   int64
		want bool
	}{
		"now":             {now.UnixMilli(), true},
		"30s old":         {now.Add(-30 * time.Second).UnixMilli(), true},
		"2 minutes old":   {now.Add(-2 * time.Minute).UnixMilli(), false},
		"2 minutes ahead": {now.Add(2 * time.Minute).UnixMilli(), false},
		"missing":         {0, false},
	}
	for name, c := range cases {
		if got := freshTimestamp(c.ms, now); got != c.want {
			t.Errorf("%s: got %v, want %v", name, got, c.want)
		}
	}
}

func TestParseIssueRef(t *testing.T) {
	good := map[string]string{
		"ENG-123": "ENG-123",
		" eng-7 ": "ENG-7",
		"https://linear.app/clyr/issue/ENG-123/fix-it": "ENG-123",
		"https://linear.app/clyr/issue/ENG-123":        "ENG-123",
		"linear.app/clyr/issue/ab2-9?foo=bar#c":        "AB2-9",
		"https://www.linear.app/clyr/issue/OPS-1/x":    "OPS-1",
	}
	for in, want := range good {
		got, ok := ParseIssueRef(in)
		if !ok || got != want {
			t.Errorf("ParseIssueRef(%q) = %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, in := range []string{
		"", "ENG", "ENG-", "ENG-0", "123", "-12", "ENG 123",
		"https://evil.example/clyr/issue/ENG-1",
		"https://linear.app/clyr/project/ENG-1",
		"https://linear.app/clyr",
		"javascript:alert(1)",
	} {
		if got, ok := ParseIssueRef(in); ok {
			t.Errorf("ParseIssueRef(%q) = %q, want rejection", in, got)
		}
	}
}

func TestNoteHTML(t *testing.T) {
	got := noteHTML("ENG-123", "Dana <Dev>", "Fixed in **v2**.\n\n<script>x</script>\n\nMail {{ .Contact.Email }}", "https://linear.app/clyr/issue/ENG-123#comment-1")
	for _, want := range []string{
		`<a href="https://linear.app/clyr/issue/ENG-123#comment-1">Linear ENG-123</a>`,
		"Dana &lt;Dev&gt;:",
		"<strong>v2</strong>",
		"&#123;&#123; .Contact.Email }}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("note %q is missing %q", got, want)
		}
	}
	if strings.Contains(got, "<script>") || strings.Contains(got, "{{") {
		t.Errorf("note keeps raw HTML or template code: %q", got)
	}
	if got := noteHTML("ENG-1", "", "hi", "javascript:alert(1)"); strings.Contains(got, "href") || !strings.Contains(got, "Linear:") {
		t.Errorf("unsafe url linked or empty author not defaulted: %q", got)
	}
}

// fakeLinear answers the GraphQL issue query like Linear.
func fakeLinear(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "lin_api_test" {
			w.WriteHeader(http.StatusUnauthorized)
			io.WriteString(w, `{"errors":[{"message":"Authentication required, not authenticated"}]}`)
			return
		}
		var req struct {
			Query     string            `json:"query"`
			Variables map[string]string `json:"variables"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Variables["id"] {
		case "ENG-123":
			io.WriteString(w, `{"data":{"issue":{"id":"issue-uuid-1","identifier":"ENG-123","url":"https://linear.app/clyr/issue/ENG-123/fix-export","title":"Fix export","state":{"name":"In Progress"}}}}`)
		case "ENG-500":
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, `{"errors":[{"message":"Internal error"}]}`)
		default:
			io.WriteString(w, `{"errors":[{"message":"Entity not found: Issue","extensions":{"type":"invalid input"}}],"data":null}`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchIssue(t *testing.T) {
	srv := fakeLinear(t)
	m := &Manager{cfg: Config{APIKey: "lin_api_test", APIURL: srv.URL}, http: srv.Client()}
	ctx := context.Background()

	got, err := m.fetchIssue(ctx, "ENG-123")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "issue-uuid-1" || got.Title != "Fix export" || got.State.Name != "In Progress" {
		t.Fatalf("unexpected issue %+v", got)
	}
	if _, err := m.fetchIssue(ctx, "ENG-9"); !errors.Is(err, ErrIssueNotFound) {
		t.Fatalf("missing issue: got %v, want ErrIssueNotFound", err)
	}
	if _, err := m.fetchIssue(ctx, "ENG-500"); err == nil || errors.Is(err, ErrIssueNotFound) {
		t.Fatalf("server error: got %v", err)
	}
	m.cfg.APIKey = "wrong"
	if _, err := m.fetchIssue(ctx, "ENG-123"); err == nil || errors.Is(err, ErrIssueNotFound) {
		t.Fatalf("bad key: got %v", err)
	}
}

// fakeNotes stores notes in Postgres like the conversation manager.
type fakeNotes struct {
	db    *sqlx.DB
	notes []string
	fail  bool
}

func (f *fakeNotes) SendPrivateNote(_ []mmodels.Media, senderID int, conversationUUID, content string, _ []cmodels.MentionInput) (cmodels.Message, error) {
	if f.fail {
		return cmodels.Message{}, errors.New("db down")
	}
	var msg cmodels.Message
	if err := f.db.QueryRow(`INSERT INTO conversation_messages (type, status, private, conversation_id, content_type, content, text_content, sender_id, sender_type)
		SELECT 'outgoing', 'sent', true, id, 'html', $2, $2, $3, 'agent' FROM conversations WHERE uuid = $1 RETURNING id`,
		conversationUUID, content, senderID).Scan(&msg.ID); err != nil {
		return msg, err
	}
	f.notes = append(f.notes, conversationUUID+"|"+content)
	return msg, nil
}

type harness struct {
	m       *Manager
	db      *sqlx.DB
	notes   *fakeNotes
	agentID int
	convs   []int
	uuids   []string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	db := testutil.NewDB(t, "linear")
	h := &harness{db: db, notes: &fakeNotes{db: db}}
	var contactID, inboxID int
	db.QueryRow(`INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Jane', 'Doe', 'jane@acme.test') RETURNING id`).Scan(&contactID)
	db.QueryRow(`INSERT INTO users (type, first_name, last_name, email) VALUES ('agent', 'Omer', 'M', 'omer@clyr.io') RETURNING id`).Scan(&h.agentID)
	db.QueryRow(`INSERT INTO inboxes (name, channel, config, "from") VALUES ('Chat', 'livechat', '{}', '') RETURNING id`).Scan(&inboxID)
	for range 2 {
		var id int
		var uuid string
		if err := db.QueryRow(`INSERT INTO conversations (contact_id, inbox_id, status_id)
			VALUES ($1, $2, (SELECT id FROM conversation_statuses ORDER BY id LIMIT 1)) RETURNING id, uuid`, contactID, inboxID).Scan(&id, &uuid); err != nil {
			t.Fatalf("inserting conversation: %v", err)
		}
		h.convs = append(h.convs, id)
		h.uuids = append(h.uuids, uuid)
	}
	srv := fakeLinear(t)
	lo := logf.New(logf.Opts{Level: logf.ErrorLevel})
	m, err := New(Config{APIKey: "lin_api_test", WebhookSecret: testSecret, APIURL: srv.URL}, db, &lo, h.notes, h.agentID)
	if err != nil {
		t.Fatal(err)
	}
	h.m = m
	return h
}

func (h *harness) webhook(t *testing.T, payload map[string]any, now time.Time) error {
	t.Helper()
	payload["webhookTimestamp"] = now.UnixMilli()
	body, _ := json.Marshal(payload)
	return h.m.HandleWebhook(body, sign(body), now)
}

func comment(id, issueID, body string) map[string]any {
	return map[string]any{
		"action": "create", "type": "Comment",
		"url":   "https://linear.app/clyr/issue/ENG-123#comment-" + id,
		"actor": map[string]any{"name": "Actor Name"},
		"data":  map[string]any{"id": id, "body": body, "issueId": issueID, "user": map[string]any{"name": "Dana Dev"}},
	}
}

func TestLinkAndUnlink(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	l, err := h.m.Link(ctx, h.convs[0], "https://linear.app/clyr/issue/eng-123/fix-export", h.agentID)
	if err != nil {
		t.Fatal(err)
	}
	if l.IssueID != "issue-uuid-1" || l.Identifier != "ENG-123" || l.State != "In Progress" || l.CreatedBy.Int != h.agentID {
		t.Fatalf("unexpected link %+v", l)
	}
	// Linking the same issue again keeps one row.
	if _, err := h.m.Link(ctx, h.convs[0], "ENG-123", h.agentID); err != nil {
		t.Fatal(err)
	}
	links, _ := h.m.Links(h.convs[0])
	if len(links) != 1 {
		t.Fatalf("want 1 link, got %d", len(links))
	}
	if _, err := h.m.Link(ctx, h.convs[0], "not an issue", h.agentID); !errors.Is(err, ErrInvalidRef) {
		t.Fatalf("got %v, want ErrInvalidRef", err)
	}
	if _, err := h.m.Link(ctx, h.convs[0], "ENG-9", h.agentID); !errors.Is(err, ErrIssueNotFound) {
		t.Fatalf("got %v, want ErrIssueNotFound", err)
	}
	// Another conversation's link ID does not unlink this one.
	if err := h.m.Unlink(h.convs[1], l.ID); err != nil {
		t.Fatal(err)
	}
	if links, _ := h.m.Links(h.convs[0]); len(links) != 1 {
		t.Fatal("unlink with the wrong conversation removed the link")
	}
	if err := h.m.Unlink(h.convs[0], l.ID); err != nil {
		t.Fatal(err)
	}
	if links, _ := h.m.Links(h.convs[0]); len(links) != 0 {
		t.Fatal("link not removed")
	}
}

func TestCommentWebhookAddsNoteOnce(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	for _, c := range h.convs {
		if _, err := h.m.Link(ctx, c, "ENG-123", h.agentID); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Now()

	if err := h.webhook(t, comment("c1", "issue-uuid-1", "Shipped in **v2**"), now); err != nil {
		t.Fatal(err)
	}
	if len(h.notes.notes) != 2 {
		t.Fatalf("want a note in each of 2 conversations, got %d", len(h.notes.notes))
	}
	for i, n := range h.notes.notes {
		if !strings.HasPrefix(n, h.uuids[i]+"|") || !strings.Contains(n, "Linear ENG-123</a></strong> · Dana Dev:") || !strings.Contains(n, "<strong>v2</strong>") {
			t.Errorf("unexpected note %q", n)
		}
	}

	// Linear retries the same delivery: nothing new.
	if err := h.webhook(t, comment("c1", "issue-uuid-1", "Shipped in **v2**"), now); err != nil {
		t.Fatal(err)
	}
	if len(h.notes.notes) != 2 {
		t.Fatalf("retry added notes: %d", len(h.notes.notes))
	}
	var recorded int
	h.db.Get(&recorded, `SELECT COUNT(*) FROM linear_comment_notes WHERE comment_id = 'c1' AND message_id IS NOT NULL`)
	if recorded != 2 {
		t.Fatalf("want 2 recorded notes, got %d", recorded)
	}

	// A comment on an issue no conversation links to does nothing.
	if err := h.webhook(t, comment("c2", "other-issue", "hello"), now); err != nil || len(h.notes.notes) != 2 {
		t.Fatalf("unlinked issue: err %v, notes %d", err, len(h.notes.notes))
	}

	// A failed note is released so the retry adds it.
	h.notes.fail = true
	if err := h.webhook(t, comment("c3", "issue-uuid-1", "later"), now); err == nil {
		t.Fatal("want an error so Linear retries")
	}
	h.notes.fail = false
	if err := h.webhook(t, comment("c3", "issue-uuid-1", "later"), now); err != nil {
		t.Fatal(err)
	}
	if len(h.notes.notes) != 4 {
		t.Fatalf("retry after failure: want 4 notes, got %d", len(h.notes.notes))
	}
}

func TestPendingCommentClaim(t *testing.T) {
	h := newHarness(t)
	if _, err := h.m.Link(context.Background(), h.convs[0], "ENG-123", h.agentID); err != nil {
		t.Fatal(err)
	}
	// A claim with no note, as a crash between claim and note leaves it.
	h.db.MustExec(`INSERT INTO linear_comment_notes (comment_id, conversation_id) VALUES ('c9', $1)`, h.convs[0])

	// While the claim is fresh, the webhook fails so Linear retries, and adds nothing.
	if err := h.webhook(t, comment("c9", "issue-uuid-1", "hi"), time.Now()); err == nil {
		t.Fatal("want an error while another request holds the claim")
	}
	if len(h.notes.notes) != 0 {
		t.Fatal("note added under a fresh claim")
	}
	// Once the claim is stale, the retry takes it over and adds the note.
	h.db.MustExec(`UPDATE linear_comment_notes SET created_at = NOW() - INTERVAL '2 minutes' WHERE comment_id = 'c9'`)
	if err := h.webhook(t, comment("c9", "issue-uuid-1", "hi"), time.Now()); err != nil {
		t.Fatal(err)
	}
	if len(h.notes.notes) != 1 {
		t.Fatalf("want 1 note after takeover, got %d", len(h.notes.notes))
	}
}

func TestWebhookRejectsBadSignatureAndStaleTimestamp(t *testing.T) {
	h := newHarness(t)
	if _, err := h.m.Link(context.Background(), h.convs[0], "ENG-123", h.agentID); err != nil {
		t.Fatal(err)
	}
	now := time.Now()

	p := comment("c1", "issue-uuid-1", "hi")
	p["webhookTimestamp"] = now.UnixMilli()
	body, _ := json.Marshal(p)
	if err := h.m.HandleWebhook(body, sign([]byte("other")), now); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("bad signature: got %v", err)
	}
	// Correctly signed, but sent five minutes ago: a replay.
	p["webhookTimestamp"] = now.Add(-5 * time.Minute).UnixMilli()
	body, _ = json.Marshal(p)
	if err := h.m.HandleWebhook(body, sign(body), now); !errors.Is(err, ErrBadSignature) {
		t.Fatalf("stale timestamp: got %v", err)
	}
	if len(h.notes.notes) != 0 {
		t.Fatal("stale webhook added a note")
	}
	bad := []byte("not json")
	if err := h.m.HandleWebhook(bad, sign(bad), now); !errors.Is(err, ErrBadPayload) {
		t.Fatalf("bad json: got %v", err)
	}
}

func TestIssueWebhookUpdatesState(t *testing.T) {
	h := newHarness(t)
	if _, err := h.m.Link(context.Background(), h.convs[0], "ENG-123", h.agentID); err != nil {
		t.Fatal(err)
	}
	err := h.webhook(t, map[string]any{
		"action": "update", "type": "Issue",
		"data": map[string]any{"id": "issue-uuid-1", "identifier": "ENG-123", "title": "Fix CSV export", "state": map[string]any{"name": "Done"}},
	}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	links, _ := h.m.Links(h.convs[0])
	if len(links) != 1 || links[0].State != "Done" || links[0].Title != "Fix CSV export" || links[0].URL == "" {
		t.Fatalf("issue not updated: %+v", links)
	}
}
