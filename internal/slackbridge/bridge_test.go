package slackbridge

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/testutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const testChannel = "CSUPPORT"

// fakeSlack records Web API calls and answers them like Slack.
type fakeSlack struct {
	mu    sync.Mutex
	calls []slackCall
	next  int
	srv   *httptest.Server
	// failAuth makes auth.test fail; fail5xx makes the next N chat.postMessage calls answer HTTP 500.
	failAuth bool
	fail5xx  int
}

type slackCall struct {
	method string
	form   url.Values
}

func newFakeSlack(t *testing.T) *fakeSlack {
	f := &fakeSlack{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer xoxb-test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/files/") {
			io.WriteString(w, "file-bytes")
			return
		}
		r.ParseForm()
		method := strings.TrimPrefix(r.URL.Path, "/api/")
		f.mu.Lock()
		f.calls = append(f.calls, slackCall{method: method, form: r.PostForm})
		f.next++
		n := f.next
		failAuth := f.failAuth
		fail := method == "chat.postMessage" && f.fail5xx > 0
		if fail {
			f.fail5xx--
		}
		f.mu.Unlock()
		if fail {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch method {
		case "auth.test":
			if failAuth {
				io.WriteString(w, `{"ok":false,"error":"invalid_auth"}`)
				return
			}
			io.WriteString(w, `{"ok":true,"user_id":"UBOT"}`)
		case "chat.postMessage":
			fmt.Fprintf(w, `{"ok":true,"ts":"1700000000.%06d"}`, n)
		case "users.info":
			switch r.PostForm.Get("user") {
			case "UERAN":
				io.WriteString(w, `{"ok":true,"user":{"id":"UERAN","name":"eran","profile":{"real_name":"Eran Artzi","email":"Eran@Clyr.io"}}}`)
			default:
				io.WriteString(w, `{"ok":true,"user":{"id":"UOTHER","name":"x","profile":{"real_name":"Stranger","email":"stranger@example.com"}}}`)
			}
		default:
			io.WriteString(w, `{"ok":true}`)
		}
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeSlack) byMethod(method string) []slackCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []slackCall
	for _, c := range f.calls {
		if c.method == method {
			out = append(out, c)
		}
	}
	return out
}

// fakeConversations stores messages in Postgres and fires message.created synchronously, as the real manager does.
type fakeConversations struct {
	db     *sqlx.DB
	conv   cmodels.Conversation
	bridge *Manager
	agents map[int]umodels.User
	notes  []cmodels.Message
	reply  []cmodels.Message
	to     [][]string
	media  [][]mmodels.Media
	// failGet makes GetConversation fail.
	failGet bool
}

func (f *fakeConversations) GetConversation(id int, uuid, refNum string) (cmodels.Conversation, error) {
	if f.failGet || uuid != f.conv.UUID {
		return cmodels.Conversation{}, errors.New("not found")
	}
	return f.conv, nil
}

func (f *fakeConversations) insert(senderID int, content string, private bool) cmodels.Message {
	var msg cmodels.Message
	f.db.QueryRowx(`INSERT INTO conversation_messages (type, status, private, conversation_id, content_type, content, text_content, sender_id, sender_type)
		VALUES ('outgoing', 'sent', $1, $2, 'html', $3, $3, $4, 'agent') RETURNING id, uuid`, private, f.conv.ID, content, senderID).Scan(&msg.ID, &msg.UUID)
	a := f.agents[senderID]
	msg.ConversationUUID = f.conv.UUID
	msg.Content = content
	msg.TextContent = content
	msg.ContentType = cmodels.ContentTypeHTML
	msg.Private = private
	msg.SenderID = senderID
	msg.SenderType = cmodels.SenderTypeAgent
	msg.Type = cmodels.MessageOutgoing
	msg.Author = cmodels.MessageAuthor{ID: senderID, FirstName: a.FirstName, LastName: a.LastName}
	f.bridge.TriggerEvent(wmodels.EventMessageCreated, &msg)
	return msg
}

func (f *fakeConversations) SendPrivateNote(media []mmodels.Media, senderID int, conversationUUID, content string, _ []cmodels.MentionInput) (cmodels.Message, error) {
	msg := f.insert(senderID, content, true)
	f.notes = append(f.notes, msg)
	f.media = append(f.media, media)
	return msg, nil
}

func (f *fakeConversations) QueueReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, to, cc, bcc []string, sendFrom string, metaMap map[string]interface{}) (cmodels.Message, error) {
	msg := f.insert(senderID, content, false)
	f.reply = append(f.reply, msg)
	f.to = append(f.to, to)
	return msg, nil
}

type fakeUsers struct{ agents map[int]umodels.User }

func (f fakeUsers) GetAgent(id int, email string) (umodels.User, error) {
	for _, a := range f.agents {
		if strings.EqualFold(a.Email.String, email) {
			return a, nil
		}
	}
	return umodels.User{}, errors.New("not found")
}

func (f fakeUsers) GetAgentCachedOrLoad(id int) (umodels.User, error) {
	if a, ok := f.agents[id]; ok {
		return a, nil
	}
	return umodels.User{}, errors.New("not found")
}

type fakeMedia struct{ uploadedBy []null.Int }

func (f *fakeMedia) Upload(fileName, contentType string, content io.ReadSeeker) (string, string, error) {
	return fileName, contentType, nil
}

func (f *fakeMedia) Insert(disposition null.String, fileName, contentType, contentID string, modelType null.String, uuid string, modelID null.Int, fileSize int, meta []byte, private bool, uploadedBy null.Int) (mmodels.Media, error) {
	f.uploadedBy = append(f.uploadedBy, uploadedBy)
	return mmodels.Media{ID: len(f.uploadedBy), UUID: uuid, Filename: fileName}, nil
}

type fakeSettings struct{}

func (fakeSettings) GetAppRootURL() (string, error) { return "https://desk.example.com", nil }

type harness struct {
	m     *Manager
	slack *fakeSlack
	convs *fakeConversations
	media *fakeMedia
	db    *sqlx.DB
	eran  umodels.User
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	db := testutil.NewDB(t, "slackbridge")
	var contactID, eranID, inboxID int
	var conv cmodels.Conversation
	db.QueryRow(`INSERT INTO users (type, first_name, last_name, email) VALUES ('contact', 'Jane', 'Doe', 'jane@acme.test') RETURNING id`).Scan(&contactID)
	db.QueryRow(`INSERT INTO users (type, first_name, last_name, email) VALUES ('agent', 'Eran', 'Artzi', 'eran@clyr.io') RETURNING id`).Scan(&eranID)
	db.QueryRow(`INSERT INTO inboxes (name, channel, config, "from") VALUES ('Chat', 'livechat', '{}', '') RETURNING id`).Scan(&inboxID)
	if err := db.QueryRow(`INSERT INTO conversations (contact_id, inbox_id, status_id, subject)
		VALUES ($1, $2, (SELECT id FROM conversation_statuses WHERE name = 'Open'), 'Cannot export receipts') RETURNING id, uuid, reference_number`,
		contactID, inboxID).Scan(&conv.ID, &conv.UUID, &conv.ReferenceNumber); err != nil {
		t.Fatalf("inserting conversation: %v", err)
	}
	conv.ContactID = contactID
	conv.InboxID = inboxID
	conv.InboxChannel = "livechat"
	conv.Status = null.StringFrom("Open")
	conv.Subject = null.StringFrom("Cannot export receipts")
	conv.Contact = cmodels.ConversationContact{FirstName: "Jane", LastName: "Doe", Email: null.StringFrom("jane@acme.test"),
		OrganizationName: null.StringFrom("Acme Inc"), OrganizationExternalID: null.StringFrom("team_42")}

	eran := umodels.User{ID: eranID, FirstName: "Eran", LastName: "Artzi", Email: null.StringFrom("eran@clyr.io"), Enabled: true}
	agents := map[int]umodels.User{eranID: eran}

	slack := newFakeSlack(t)
	lo := logf.New(logf.Opts{Level: logf.ErrorLevel})
	media := &fakeMedia{}
	m, err := New(Config{BotToken: "xoxb-test", SigningSecret: "s", ChannelID: testChannel, APIURL: slack.srv.URL + "/api/"},
		db, &lo, fakeUsers{agents: agents}, media, fakeSettings{})
	if err != nil {
		t.Fatal(err)
	}
	m.api.wait = func(context.Context, time.Duration) error { return nil }
	convs := &fakeConversations{db: db, conv: conv, bridge: m, agents: agents}
	m.SetConversationStore(convs)
	return &harness{m: m, slack: slack, convs: convs, media: media, db: db, eran: eran}
}

// drain runs every queued job, including jobs that running a job queues.
func (h *harness) drain(t *testing.T) {
	t.Helper()
	for {
		select {
		case j := <-h.m.jobs:
			h.m.process(context.Background(), j)
		default:
			return
		}
	}
}

func (h *harness) contactMessage(t *testing.T, text string) {
	t.Helper()
	var msg cmodels.Message
	h.db.QueryRow(`INSERT INTO conversation_messages (type, status, conversation_id, content_type, content, text_content, sender_id, sender_type)
		VALUES ('incoming', 'received', $1, 'text', $2, $2, $3, 'contact') RETURNING id`, h.convs.conv.ID, text, h.convs.conv.ContactID).Scan(&msg.ID)
	msg.ConversationUUID = h.convs.conv.UUID
	msg.TextContent = text
	msg.ContentType = cmodels.ContentTypeText
	msg.SenderType = cmodels.SenderTypeContact
	msg.Type = cmodels.MessageIncoming
	msg.Author = cmodels.MessageAuthor{FirstName: "Jane", LastName: "Doe"}
	h.m.TriggerEvent(wmodels.EventMessageCreated, &msg)
}

func (h *harness) slackEvent(t *testing.T, ev map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"type": "event_callback", "event": ev})
	if _, err := h.m.HandleSlackEvent(body); err != nil {
		t.Fatal(err)
	}
}

func (h *harness) rootTS(t *testing.T) string {
	t.Helper()
	tr, ok, err := h.m.store.threadByConversation(h.convs.conv.ID)
	if err != nil || !ok {
		t.Fatalf("no thread: ok=%v err=%v", ok, err)
	}
	return tr.ThreadTS
}

func TestContactMessageCreatesThreadOnce(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "The export button does nothing")
	h.contactMessage(t, "Still broken")
	h.drain(t)

	posts := h.slack.byMethod("chat.postMessage")
	if len(posts) != 3 {
		t.Fatalf("want root + 2 messages, got %d posts", len(posts))
	}
	root := posts[0].form
	if root.Get("channel") != testChannel || root.Get("thread_ts") != "" {
		t.Fatalf("root posted wrong: %v", root)
	}
	for _, want := range []string{"Cannot export receipts", "Jane Doe", "jane@acme.test", "Acme Inc", "team_42", "Status: Open", "Channel: chat",
		"https://desk.example.com/inboxes/all/conversation/" + h.convs.conv.UUID} {
		if !strings.Contains(root.Get("text"), want) {
			t.Errorf("root text misses %q: %s", want, root.Get("text"))
		}
	}
	ts := h.rootTS(t)
	for _, p := range posts[1:] {
		if p.form.Get("thread_ts") != ts {
			t.Fatalf("message not in the thread: %v", p.form)
		}
		if !strings.Contains(p.form.Get("text"), "(customer)") {
			t.Fatalf("customer message not marked: %s", p.form.Get("text"))
		}
	}

	// A retried event for the same message is not posted twice.
	var id int
	h.db.Get(&id, `SELECT MIN(id) FROM conversation_messages`)
	h.m.TriggerEvent(wmodels.EventMessageCreated, &cmodels.Message{ID: id, ConversationUUID: h.convs.conv.UUID, Type: cmodels.MessageIncoming})
	h.drain(t)
	if n := len(h.slack.byMethod("chat.postMessage")); n != 3 {
		t.Fatalf("duplicate post: %d posts", n)
	}
}

func TestSlackReplyBecomesNoteAndIsNotEchoed(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "help")
	h.drain(t)
	ts := h.rootTS(t)

	ev := map[string]any{"type": "message", "channel": testChannel, "user": "UERAN", "text": "Looking at it, *bold* claim", "ts": "1700000001.000100", "thread_ts": ts}
	h.slackEvent(t, ev)
	h.slackEvent(t, ev) // Slack retry
	h.drain(t)

	if len(h.convs.notes) != 1 || len(h.convs.reply) != 0 {
		t.Fatalf("want 1 note and no reply, got %d notes, %d replies", len(h.convs.notes), len(h.convs.reply))
	}
	if got := h.convs.notes[0].Content; got != "<p>Looking at it, <strong>bold</strong> claim</p>" {
		t.Fatalf("note content %q", got)
	}
	if h.convs.notes[0].SenderID != h.eran.ID {
		t.Fatalf("note author %d, want %d", h.convs.notes[0].SenderID, h.eran.ID)
	}
	// Root + contact message only: the note from Slack is not posted back.
	if n := len(h.slack.byMethod("chat.postMessage")); n != 2 {
		t.Fatalf("note echoed to Slack: %d posts", n)
	}
	r := h.slack.byMethod("reactions.add")
	if len(r) != 1 || r[0].form.Get("name") != reactionNote {
		t.Fatalf("want one memo reaction, got %v", r)
	}
}

func TestMentionSendsReplyToCustomer(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "help")
	h.drain(t)
	ts := h.rootTS(t)

	text := "<@UBOT> Fixed, please try again"
	// Slack sends both a message and an app_mention event for one mention.
	h.slackEvent(t, map[string]any{"type": "message", "channel": testChannel, "user": "UERAN", "text": text, "ts": "1700000002.000100", "thread_ts": ts})
	h.slackEvent(t, map[string]any{"type": "app_mention", "channel": testChannel, "user": "UERAN", "text": text, "ts": "1700000002.000100", "thread_ts": ts})
	h.drain(t)

	if len(h.convs.reply) != 1 || len(h.convs.notes) != 0 {
		t.Fatalf("want 1 reply and no note, got %d replies, %d notes", len(h.convs.reply), len(h.convs.notes))
	}
	if got := h.convs.reply[0].Content; got != "<p>Fixed, please try again</p>" {
		t.Fatalf("reply content %q", got)
	}
	if h.convs.to[0] != nil {
		t.Fatalf("chat reply got recipients %v", h.convs.to[0])
	}
	r := h.slack.byMethod("reactions.add")
	if len(r) != 1 || r[0].form.Get("name") != reactionReply {
		t.Fatalf("want one outbox reaction, got %v", r)
	}
}

func TestEmailMentionAddressesContact(t *testing.T) {
	h := newHarness(t)
	h.convs.conv.InboxChannel = "email"
	h.contactMessage(t, "help")
	h.drain(t)
	h.slackEvent(t, map[string]any{"type": "app_mention", "channel": testChannel, "user": "UERAN", "text": "<@UBOT> done", "ts": "1700000003.000100", "thread_ts": h.rootTS(t)})
	h.drain(t)
	if len(h.convs.to) != 1 || len(h.convs.to[0]) != 1 || h.convs.to[0][0] != "jane@acme.test" {
		t.Fatalf("email reply recipients %v", h.convs.to)
	}
}

func TestIgnoredSlackMessages(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "help")
	h.drain(t)
	ts := h.rootTS(t)

	for _, ev := range []map[string]any{
		{"type": "message", "channel": testChannel, "user": "UERAN", "text": "top level", "ts": "1700000010.1"},
		{"type": "message", "channel": "COTHER", "user": "UERAN", "text": "other channel", "ts": "1700000010.2", "thread_ts": ts},
		{"type": "message", "channel": testChannel, "bot_id": "BSELF", "text": "our own post", "ts": "1700000010.3", "thread_ts": ts},
		{"type": "message", "subtype": "message_changed", "channel": testChannel, "user": "UERAN", "text": "edit", "ts": "1700000010.4", "thread_ts": ts},
		{"type": "message", "channel": testChannel, "user": "UERAN", "text": "unknown thread", "ts": "1700000010.5", "thread_ts": "1.2"},
	} {
		h.slackEvent(t, ev)
	}
	h.drain(t)
	if len(h.convs.notes)+len(h.convs.reply) != 0 {
		t.Fatalf("ignored events created messages: %d notes, %d replies", len(h.convs.notes), len(h.convs.reply))
	}
}

func TestUnknownSlackUserGetsWarning(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "help")
	h.drain(t)
	ts := h.rootTS(t)
	h.slackEvent(t, map[string]any{"type": "message", "channel": testChannel, "user": "UOTHER", "text": "hi", "ts": "1700000020.1", "thread_ts": ts})
	h.drain(t)
	if len(h.convs.notes) != 0 {
		t.Fatal("note created for a Slack user with no agent")
	}
	posts := h.slack.byMethod("chat.postMessage")
	last := posts[len(posts)-1].form
	if last.Get("thread_ts") != ts || !strings.Contains(last.Get("text"), "stranger@example.com") {
		t.Fatalf("want a warning in the thread naming the email, got %v", last)
	}
}

func TestSlackEmailAliasMapsToAgent(t *testing.T) {
	h := newHarness(t)
	h.m.cfg.AgentEmailAliases = map[string]string{"stranger@example.com": "eran@clyr.io"}
	h.contactMessage(t, "help")
	h.drain(t)
	ts := h.rootTS(t)
	h.slackEvent(t, map[string]any{"type": "message", "channel": testChannel, "user": "UOTHER", "text": "on it", "ts": "1700000021.1", "thread_ts": ts})
	h.drain(t)
	if len(h.convs.notes) != 1 || h.convs.notes[0].SenderID != h.eran.ID {
		t.Fatalf("want one note by the aliased agent, got %+v", h.convs.notes)
	}
}

func TestParseEmailAliases(t *testing.T) {
	got, err := ParseEmailAliases(" Omer.Menashe@gmail.com = omer@clyr.io ,, a@x.com=b@x.com ")
	if err != nil {
		t.Fatal(err)
	}
	if got["omer.menashe@gmail.com"] != "omer@clyr.io" || got["a@x.com"] != "b@x.com" || len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	if got, err := ParseEmailAliases("o'brien@x.com=agent@clyr.io"); err != nil || got["o'brien@x.com"] != "agent@clyr.io" {
		t.Fatalf("apostrophe address: %v %v", got, err)
	}
	if empty, err := ParseEmailAliases(""); err != nil || len(empty) != 0 {
		t.Fatalf("empty input: %v %v", empty, err)
	}
	for _, bad := range []string{"omer@clyr.io", "a@x.com=", "=b@x.com", "nope=b@x.com", "a@x.com=b@x.com=typo", "a@x.com=b@@x.com", "a@x.com=b@x", "a b@x.com=c@x.com"} {
		if _, err := ParseEmailAliases(bad); err == nil {
			t.Fatalf("want an error for %q", bad)
		}
	}
}

func TestSlackFileIsOwnedByAgent(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "help")
	h.drain(t)
	h.slackEvent(t, map[string]any{"type": "message", "subtype": "file_share", "channel": testChannel, "user": "UERAN", "text": "", "ts": "1700000030.1",
		"thread_ts": h.rootTS(t), "files": []map[string]any{{"name": "log.txt", "mimetype": "text/plain", "url_private_download": h.slack.srv.URL + "/files/f1"}}})
	h.drain(t)
	if len(h.convs.notes) != 1 || len(h.convs.media[0]) != 1 {
		t.Fatalf("want a note with one file, got %d notes", len(h.convs.notes))
	}
	if got := h.media.uploadedBy; len(got) != 1 || got[0] != null.IntFrom(h.eran.ID) {
		t.Fatalf("file uploaded_by %v, want agent %d", got, h.eran.ID)
	}
}

func TestLibredeskNoteAndStatusChange(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "help")
	h.drain(t)

	// A note typed in Libredesk goes to Slack.
	h.convs.insert(h.eran.ID, "Checking the logs", true)
	h.drain(t)
	posts := h.slack.byMethod("chat.postMessage")
	if last := posts[len(posts)-1].form.Get("text"); !strings.Contains(last, "Eran Artzi") || !strings.Contains(last, "(note)") || !strings.Contains(last, "Checking the logs") {
		t.Fatalf("note post %q", last)
	}

	// A status change updates the root, not a new message.
	h.convs.conv.Status = null.StringFrom("Resolved")
	h.m.TriggerEvent(wmodels.EventConversationStatusChanged, map[string]any{"conversation_uuid": h.convs.conv.UUID})
	h.drain(t)
	upd := h.slack.byMethod("chat.update")
	if len(upd) != 1 || upd[0].form.Get("ts") != h.rootTS(t) || !strings.Contains(upd[0].form.Get("text"), "Status: Resolved") {
		t.Fatalf("root update %v", upd)
	}
}

func TestURLVerification(t *testing.T) {
	h := newHarness(t)
	got, err := h.m.HandleSlackEvent([]byte(`{"type":"url_verification","challenge":"abc123"}`))
	if err != nil || got != "abc123" {
		t.Fatalf("challenge %q, err %v", got, err)
	}
}

func TestQueueFullIsReported(t *testing.T) {
	h := newHarness(t)
	h.m.jobs = make(chan job, 1)
	body := []byte(`{"type":"event_callback","event":{"type":"message"}}`)
	if _, err := h.m.HandleSlackEvent(body); err != nil {
		t.Fatalf("first event: %v", err)
	}
	if _, err := h.m.HandleSlackEvent(body); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("want ErrQueueFull, got %v", err)
	}
}

func TestUnknownBotIDDoesNotGuess(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "help")
	h.drain(t)
	h.slack.failAuth = true
	ts := h.rootTS(t)
	h.slackEvent(t, map[string]any{"type": "app_mention", "channel": testChannel, "user": "UERAN", "text": "<@UBOT> fixed", "ts": "1700000040.1", "thread_ts": ts})
	h.drain(t)
	if len(h.convs.notes)+len(h.convs.reply) != 0 {
		t.Fatalf("message added without the bot ID: %d notes, %d replies", len(h.convs.notes), len(h.convs.reply))
	}
	posts := h.slack.byMethod("chat.postMessage")
	if last := posts[len(posts)-1].form; last.Get("thread_ts") != ts || !strings.Contains(last.Get("text"), "send it again") {
		t.Fatalf("want a warning in the thread, got %v", last)
	}
}

func TestReplyWarnsWhenConversationFails(t *testing.T) {
	h := newHarness(t)
	h.contactMessage(t, "help")
	h.drain(t)
	ts := h.rootTS(t)
	h.convs.failGet = true
	h.slackEvent(t, map[string]any{"type": "app_mention", "channel": testChannel, "user": "UERAN", "text": "<@UBOT> fixed", "ts": "1700000050.1", "thread_ts": ts})
	h.drain(t)
	if len(h.convs.reply) != 0 {
		t.Fatal("reply sent although the conversation failed to load")
	}
	posts := h.slack.byMethod("chat.postMessage")
	if last := posts[len(posts)-1].form; last.Get("thread_ts") != ts || !strings.Contains(last.Get("text"), "Not sent to the customer") {
		t.Fatalf("want a warning in the thread, got %v", last)
	}
}

func TestSlackServerErrorIsRetried(t *testing.T) {
	h := newHarness(t)
	h.slack.fail5xx = 2
	h.contactMessage(t, "help")
	h.drain(t)
	h.rootTS(t)
	// 2 failed root posts, then root and message.
	if n := len(h.slack.byMethod("chat.postMessage")); n != 4 {
		t.Fatalf("want 4 postMessage calls, got %d", n)
	}
}

func TestLongRetryAfterFailsFast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "600")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	api := newSlackAPI("t", srv.URL)
	waited := false
	api.wait = func(context.Context, time.Duration) error { waited = true; return nil }
	if _, err := api.postMessage(context.Background(), "C", "", "x"); err == nil || waited {
		t.Fatalf("want an immediate error, got err=%v waited=%v", err, waited)
	}
}

func TestRetryStopsWhenContextEnds(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	api := newSlackAPI("t", srv.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	if _, err := api.postMessage(ctx, "C", "", "x"); err == nil {
		t.Fatal("want an error")
	}
	if time.Since(start) > time.Second {
		t.Fatal("retry did not stop on a cancelled context")
	}
}

func TestUnsavedRootIsDeleted(t *testing.T) {
	h := newHarness(t)
	// Another conversation already owns the ts that the fake Slack gives the next post, so saving the root fails.
	var other int
	h.db.QueryRow(`INSERT INTO conversations (contact_id, inbox_id, status_id) SELECT contact_id, inbox_id, status_id FROM conversations WHERE id = $1 RETURNING id`, h.convs.conv.ID).Scan(&other)
	h.db.MustExec(`INSERT INTO slack_threads (conversation_id, channel_id, thread_ts) VALUES ($1, $2, '1700000000.000001')`, other, testChannel)

	h.contactMessage(t, "help")
	h.drain(t)
	del := h.slack.byMethod("chat.delete")
	if len(del) != 1 || del[0].form.Get("ts") != "1700000000.000001" {
		t.Fatalf("want the unsaved root deleted, got %v", del)
	}
	if _, ok, _ := h.m.store.threadByConversation(h.convs.conv.ID); ok {
		t.Fatal("thread saved although the insert failed")
	}
}
