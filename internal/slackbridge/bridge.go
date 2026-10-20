// Package slackbridge mirrors Libredesk conversations into Slack threads (Clyr fork addition).
//
// Each conversation gets one thread in a Slack channel. Customer messages, private notes and agent
// replies are posted into it, and the thread root shows the conversation's status. A plain reply in the
// thread becomes a private note by the agent whose email matches the Slack user. A reply that mentions
// the bridge's bot becomes a reply to the customer.
//
// All work runs on one worker goroutine, in the order it was queued. A note that the bridge creates
// queues its own message.created event, and that event runs only after the bridge has recorded the
// note as coming from Slack, so the bridge does not post it back.
package slackbridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/attachment"
	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	umodels "github.com/abhinavxd/libredesk/internal/user/models"
	wmodels "github.com/abhinavxd/libredesk/internal/webhook/models"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

const (
	defaultQueueSize = 1000
	userCacheTTL     = time.Hour
	jobTimeout       = 2 * time.Minute

	reactionNote  = "memo"
	reactionReply = "outbox_tray"
)

// ErrQueueFull means a Slack event was not queued. The HTTP handler answers with an error, so Slack retries.
var ErrQueueFull = errors.New("slack bridge queue is full")

// Config holds the bridge settings from the [slack] config section.
type Config struct {
	BotToken      string
	SigningSecret string
	// ChannelID is the Slack channel that holds the conversation threads.
	ChannelID string
	// AIBotID is the bot_id of the Slack AI agent. Its messages are handled as the agent AIAgentEmail.
	AIBotID      string
	AIAgentEmail string
	// AgentEmailAliases maps a Slack account email to the Libredesk agent email, for agents whose
	// Slack account uses another address. Keys and values are lowercase. See ParseEmailAliases.
	AgentEmailAliases map[string]string
	// APIURL overrides the Slack Web API base URL (tests).
	APIURL    string
	QueueSize int
}

// ParseEmailAliases reads "slack@example.com=agent@example.com, other@x.com=agent2@x.com" into a
// lowercase map. Malformed pairs are returned as an error so a typo is not silently ignored.
func ParseEmailAliases(s string) (map[string]string, error) {
	out := map[string]string{}
	for _, pair := range strings.Split(s, ",") {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}
		from, to, ok := strings.Cut(pair, "=")
		from, to = strings.ToLower(strings.TrimSpace(from)), strings.ToLower(strings.TrimSpace(to))
		if !ok || !plainEmail(from) || !plainEmail(to) {
			return nil, fmt.Errorf("invalid slack agent email alias %q: want slack-email=agent-email", pair)
		}
		out[from] = to
	}
	return out, nil
}

// plainEmail reports whether s is one bare address: a single @ with text on both sides, no spaces,
// separators, quotes or brackets (an apostrophe, as in o'brien@, is allowed).
func plainEmail(s string) bool {
	if strings.ContainsAny(s, "=, <>\"\t") || strings.Count(s, "@") != 1 {
		return false
	}
	local, domain, _ := strings.Cut(s, "@")
	return local != "" && strings.Contains(domain, ".") && !strings.HasSuffix(domain, ".")
}

// Enabled reports whether the bridge has the settings it needs.
func (c Config) Enabled() bool {
	return c.BotToken != "" && c.SigningSecret != "" && c.ChannelID != ""
}

type conversationStore interface {
	GetConversation(id int, uuid, refNum string) (cmodels.Conversation, error)
	SendPrivateNote(media []mmodels.Media, senderID int, conversationUUID, content string, mentions []cmodels.MentionInput) (cmodels.Message, error)
	QueueReply(media []mmodels.Media, inboxID, senderID, contactID int, conversationUUID, content string, to, cc, bcc []string, sendFrom string, metaMap map[string]interface{}) (cmodels.Message, error)
}

type userStore interface {
	GetAgent(id int, email string) (umodels.User, error)
	GetAgentCachedOrLoad(id int) (umodels.User, error)
}

type mediaStore interface {
	Upload(fileName, contentType string, content io.ReadSeeker) (string, string, error)
	Insert(disposition null.String, fileName, contentType, contentID string, modelType null.String, uuid string, modelID null.Int, fileSize int, meta []byte, private bool, uploadedBy null.Int) (mmodels.Media, error)
}

type settingStore interface {
	GetAppRootURL() (string, error)
}

// job is one unit of work for the worker.
type job struct {
	event wmodels.WebhookEvent
	data  any
	slack *slackEvent
}

// Manager runs the bridge.
type Manager struct {
	cfg      Config
	lo       *logf.Logger
	api      *slackAPI
	store    *store
	users    userStore
	media    mediaStore
	settings settingStore
	convs    conversationStore
	jobs     chan job

	botUserID string

	mu        sync.Mutex
	userCache map[string]cachedUser
}

type cachedUser struct {
	user    slackUser
	fetched time.Time
}

// New creates the bridge. Call SetConversationStore before Run.
func New(cfg Config, db *sqlx.DB, lo *logf.Logger, users userStore, media mediaStore, settings settingStore) (*Manager, error) {
	if !cfg.Enabled() {
		return nil, errors.New("slack bridge needs bot_token, signing_secret and channel_id")
	}
	st, err := newStore(db)
	if err != nil {
		return nil, err
	}
	size := cfg.QueueSize
	if size <= 0 {
		size = defaultQueueSize
	}
	return &Manager{
		cfg:       cfg,
		lo:        lo,
		api:       newSlackAPI(cfg.BotToken, cfg.APIURL),
		store:     st,
		users:     users,
		media:     media,
		settings:  settings,
		jobs:      make(chan job, size),
		userCache: map[string]cachedUser{},
	}, nil
}

// SetConversationStore sets the conversation manager. The bridge is built before it, because the
// conversation manager sends its events to the bridge.
func (m *Manager) SetConversationStore(c conversationStore) {
	m.convs = c
}

// SigningSecret returns the secret that signs Slack requests.
func (m *Manager) SigningSecret() string {
	return m.cfg.SigningSecret
}

// TriggerEvent queues a Libredesk event. It never blocks: when the queue is full the event is dropped
// and logged, the same as webhooks. The conversation manager calls it on the request path.
func (m *Manager) TriggerEvent(event wmodels.WebhookEvent, data any) {
	switch event {
	case wmodels.EventMessageCreated, wmodels.EventConversationCreated, wmodels.EventConversationStatusChanged,
		wmodels.EventConversationAssigned, wmodels.EventConversationUnassigned, wmodels.EventConversationTagsChanged:
	default:
		return
	}
	m.enqueue(job{event: event, data: data})
}

// HandleSlackEvent parses a verified Slack Events API body. It returns the challenge for a
// url_verification request; other events are queued.
func (m *Manager) HandleSlackEvent(body []byte) (challenge string, err error) {
	var env struct {
		Type      string     `json:"type"`
		Challenge string     `json:"challenge"`
		Event     slackEvent `json:"event"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return "", fmt.Errorf("decoding slack event: %w", err)
	}
	switch env.Type {
	case "url_verification":
		return env.Challenge, nil
	case "event_callback":
		ev := env.Event
		if !m.enqueue(job{slack: &ev}) {
			return "", ErrQueueFull
		}
	}
	return "", nil
}

// enqueue queues a job. It returns false when the queue is full and the job was dropped.
func (m *Manager) enqueue(j job) bool {
	select {
	case m.jobs <- j:
		return true
	default:
		m.lo.Error("slack bridge queue is full, dropping event", "event", j.event)
		return false
	}
}

// Run processes queued work until ctx ends.
func (m *Manager) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case j := <-m.jobs:
			jctx, cancel := context.WithTimeout(ctx, jobTimeout)
			m.process(jctx, j)
			cancel()
		}
	}
}

func (m *Manager) process(ctx context.Context, j job) {
	defer func() {
		if r := recover(); r != nil {
			m.lo.Error("slack bridge job panicked", "event", j.event, "panic", r)
		}
	}()
	if m.convs == nil {
		m.lo.Error("slack bridge has no conversation store")
		return
	}
	var err error
	if j.slack != nil {
		err = m.handleSlack(ctx, *j.slack)
	} else {
		err = m.handleLibredesk(ctx, j.event, j.data)
	}
	if err != nil {
		m.lo.Error("slack bridge error", "event", j.event, "error", err)
	}
}

// ensureBotUserID loads the bot's Slack user ID once. It is needed to detect @mentions of the bot.
func (m *Manager) ensureBotUserID(ctx context.Context) string {
	if m.botUserID == "" {
		id, err := m.api.authTest(ctx)
		if err != nil {
			m.lo.Error("slack auth.test failed", "error", err)
			return ""
		}
		m.botUserID = id
	}
	return m.botUserID
}

// slackUser returns a Slack user's profile, cached for an hour.
func (m *Manager) slackUser(ctx context.Context, id string) (slackUser, error) {
	m.mu.Lock()
	c, ok := m.userCache[id]
	m.mu.Unlock()
	if ok && time.Since(c.fetched) < userCacheTTL {
		return c.user, nil
	}
	u, err := m.api.userInfo(ctx, id)
	if err != nil {
		return slackUser{}, err
	}
	m.mu.Lock()
	m.userCache[id] = cachedUser{user: u, fetched: time.Now()}
	m.mu.Unlock()
	return u, nil
}

// ---- Libredesk -> Slack ----

func (m *Manager) handleLibredesk(ctx context.Context, event wmodels.WebhookEvent, data any) error {
	switch event {
	case wmodels.EventMessageCreated:
		var msg cmodels.Message
		switch v := data.(type) {
		case *cmodels.Message:
			msg = *v
		case cmodels.Message:
			msg = v
		default:
			return fmt.Errorf("unexpected message payload %T", data)
		}
		return m.postMessage(ctx, msg)
	case wmodels.EventConversationCreated:
		conv, ok := data.(cmodels.Conversation)
		if !ok {
			return fmt.Errorf("unexpected conversation payload %T", data)
		}
		_, err := m.ensureThread(ctx, conv)
		return err
	default:
		uuid := conversationUUIDFromPayload(data)
		if uuid == "" {
			return nil
		}
		return m.updateRoot(ctx, uuid)
	}
}

func conversationUUIDFromPayload(data any) string {
	if p, ok := data.(map[string]any); ok {
		if s, ok := p["conversation_uuid"].(string); ok {
			return s
		}
	}
	return ""
}

// ensureThread returns the conversation's thread and posts its root when it has none.
func (m *Manager) ensureThread(ctx context.Context, conv cmodels.Conversation) (thread, error) {
	t, ok, err := m.store.threadByConversation(conv.ID)
	if err != nil || ok {
		return t, err
	}
	ts, err := m.api.postMessage(ctx, m.cfg.ChannelID, "", m.rootText(conv))
	if err != nil {
		return t, err
	}
	insertErr := m.store.insertThread(conv.ID, m.cfg.ChannelID, ts)
	saved, ok, err := m.store.threadByConversation(conv.ID)
	if insertErr == nil && err == nil && ok && saved.ThreadTS == ts {
		return thread{ConversationID: conv.ID, ChannelID: m.cfg.ChannelID, ThreadTS: ts, ConversationUUID: conv.UUID}, nil
	}
	// The root was not saved as this conversation's thread. Delete it, so no thread exists in Slack
	// that the bridge does not know about.
	if delErr := m.api.deleteMessage(ctx, m.cfg.ChannelID, ts); delErr != nil {
		m.lo.Error("deleting unsaved slack thread root", "conversation_id", conv.ID, "ts", ts, "error", delErr)
	}
	switch {
	case insertErr != nil:
		return t, insertErr
	case err != nil:
		return t, err
	case ok:
		return saved, nil
	}
	return t, fmt.Errorf("slack thread for conversation %d was not saved", conv.ID)
}

// updateRoot refreshes the thread root after a status, assignee or tag change.
func (m *Manager) updateRoot(ctx context.Context, conversationUUID string) error {
	conv, err := m.convs.GetConversation(0, conversationUUID, "")
	if err != nil {
		return err
	}
	t, ok, err := m.store.threadByConversation(conv.ID)
	if err != nil || !ok {
		return err
	}
	return m.api.updateMessage(ctx, t.ChannelID, t.ThreadTS, m.rootText(conv))
}

func (m *Manager) postMessage(ctx context.Context, msg cmodels.Message) error {
	if msg.Type == cmodels.MessageActivity {
		return nil
	}
	bridged, err := m.store.isBridged(msg.ID)
	if err != nil || bridged {
		return err
	}
	conv, err := m.convs.GetConversation(0, msg.ConversationUUID, "")
	if err != nil {
		return err
	}
	t, err := m.ensureThread(ctx, conv)
	if err != nil {
		return err
	}
	ts, err := m.api.postMessage(ctx, t.ChannelID, t.ThreadTS, m.messageText(msg))
	if err != nil {
		return err
	}
	return m.store.insertOutgoing(t.ChannelID, ts, msg.ID)
}

// rootText is the thread root: who wrote in, their organization, and the conversation's state.
func (m *Manager) rootText(conv cmodels.Conversation) string {
	c := conv.Contact
	name := strings.TrimSpace(c.FirstName + " " + c.LastName)
	if name == "" {
		name = "Unknown contact"
	}
	subject := conv.Subject.String
	if subject == "" {
		subject = "(no subject)"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "*#%s %s*\n", escapeMrkdwn(conv.ReferenceNumber), escapeMrkdwn(subject))
	b.WriteString(escapeMrkdwn(name))
	if c.Email.String != "" {
		fmt.Fprintf(&b, " · %s", escapeMrkdwn(c.Email.String))
	}
	if c.OrganizationName.String != "" {
		fmt.Fprintf(&b, " · *%s*", escapeMrkdwn(c.OrganizationName.String))
		if c.OrganizationExternalID.String != "" {
			fmt.Fprintf(&b, " (teamId `%s`)", escapeMrkdwn(c.OrganizationExternalID.String))
		}
	}
	b.WriteString("\n")

	parts := []string{"Channel: " + channelLabel(conv.InboxChannel)}
	if conv.Status.String != "" {
		parts = append(parts, "Status: "+escapeMrkdwn(conv.Status.String))
	}
	assignee := "unassigned"
	if conv.AssignedUserID.Valid {
		if u, err := m.users.GetAgentCachedOrLoad(conv.AssignedUserID.Int); err == nil {
			assignee = u.FullName()
		}
	}
	parts = append(parts, "Assignee: "+escapeMrkdwn(assignee))
	if tags := tagNames(conv.Tags); len(tags) > 0 {
		parts = append(parts, "Tags: "+escapeMrkdwn(strings.Join(tags, ", ")))
	}
	b.WriteString(strings.Join(parts, " · "))

	if root, err := m.settings.GetAppRootURL(); err == nil && root != "" {
		fmt.Fprintf(&b, "\n<%s/inboxes/all/conversation/%s|Open in Libredesk>", strings.TrimRight(root, "/"), conv.UUID)
	}
	b.WriteString("\n_Reply in this thread to add a note. Mention the bot to send your reply to the customer._")
	return b.String()
}

func channelLabel(channel string) string {
	switch channel {
	case "livechat":
		return "chat"
	case "":
		return "unknown"
	}
	return channel
}

func tagNames(tags null.JSON) []string {
	if !tags.Valid {
		return nil
	}
	var names []string
	if err := json.Unmarshal(tags.JSON, &names); err != nil {
		return nil
	}
	return names
}

// messageText is a message as posted into the thread: who wrote it, what kind it is, and its text.
func (m *Manager) messageText(msg cmodels.Message) string {
	author := strings.TrimSpace(msg.Author.FirstName + " " + msg.Author.LastName)
	var header string
	switch {
	case msg.SenderType == cmodels.SenderTypeContact:
		header = ":bust_in_silhouette: *" + escapeMrkdwn(orDefault(author, "Customer")) + "* (customer)"
	case msg.Private:
		header = ":memo: *" + escapeMrkdwn(orDefault(author, "Agent")) + "* (note)"
	default:
		header = ":outbox_tray: *" + escapeMrkdwn(orDefault(author, "Agent")) + "* (sent to customer)"
	}

	text := msg.TextContent
	if msg.ContentType == cmodels.ContentTypeHTML {
		if t := stringutil.HTML2TextNoQuotes(msg.Content); t != "" {
			text = t
		}
	}
	var b strings.Builder
	b.WriteString(header)
	if q := quote(escapeMrkdwn(text)); q != "" {
		b.WriteString("\n" + q)
	}
	root, _ := m.settings.GetAppRootURL()
	for _, a := range msg.Attachments {
		if a.Disposition == attachment.DispositionInline {
			continue
		}
		link := a.URL
		if strings.HasPrefix(link, "/") {
			link = strings.TrimRight(root, "/") + link
		}
		if link == "" {
			fmt.Fprintf(&b, "\n:paperclip: %s", escapeMrkdwn(a.Name))
			continue
		}
		fmt.Fprintf(&b, "\n:paperclip: <%s|%s>", link, escapeMrkdwn(a.Name))
	}
	return b.String()
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// ---- Slack -> Libredesk ----

// slackEvent is the part of a Slack message or app_mention event the bridge reads.
type slackEvent struct {
	Type     string      `json:"type"`
	Subtype  string      `json:"subtype"`
	Channel  string      `json:"channel"`
	User     string      `json:"user"`
	BotID    string      `json:"bot_id"`
	Text     string      `json:"text"`
	TS       string      `json:"ts"`
	ThreadTS string      `json:"thread_ts"`
	Files    []slackFile `json:"files"`
}

type slackFile struct {
	Name               string `json:"name"`
	Mimetype           string `json:"mimetype"`
	Size               int    `json:"size"`
	URLPrivateDownload string `json:"url_private_download"`
}

func (m *Manager) handleSlack(ctx context.Context, ev slackEvent) error {
	if ev.Type != "message" && ev.Type != "app_mention" {
		return nil
	}
	switch ev.Subtype {
	case "", "file_share", "thread_broadcast", "bot_message":
	default:
		return nil
	}
	// Only replies inside a thread of the bridge's channel.
	if ev.Channel != m.cfg.ChannelID || ev.ThreadTS == "" || ev.ThreadTS == ev.TS {
		return nil
	}
	// Bot messages: only the Slack AI agent. This also skips the bridge's own posts.
	isAI := m.cfg.AIBotID != "" && ev.BotID == m.cfg.AIBotID
	if ev.BotID != "" && !isAI {
		return nil
	}
	if !isAI && ev.User == "" {
		return nil
	}

	t, ok, err := m.store.threadByTS(ev.Channel, ev.ThreadTS)
	if err != nil || !ok {
		return err
	}
	claimID, ok, err := m.store.claim(ev.Channel, ev.TS)
	if err != nil || !ok {
		return err
	}
	// Slack does not retry an event that was answered with 200, and the event is now claimed.
	// Every failure from here on tells the Slack user, so they can send the message again.
	if err := m.addFromSlack(ctx, ev, t, claimID, isAI); err != nil {
		m.replyInThread(ctx, ev, err.userText)
		return err.err
	}
	return nil
}

// slackError is a failure to add a Slack message, with the text shown to the Slack user.
type slackError struct {
	userText string
	err      error
}

func failed(userText string, err error) *slackError {
	return &slackError{userText: userText + " Please send it again.", err: err}
}

// addFromSlack adds a claimed Slack message to the conversation as a note or a customer reply.
func (m *Manager) addFromSlack(ctx context.Context, ev slackEvent, t thread, claimID int, isAI bool) *slackError {
	agent, email, err := m.agentFor(ctx, ev, isAI)
	if err != nil {
		if email != "" {
			return &slackError{userText: fmt.Sprintf("Not added to Libredesk: no enabled Libredesk agent has the email %s.", email), err: err}
		}
		return failed("Not added to Libredesk: the Slack user's email could not be read.", err)
	}

	// Without the bot's own ID a mention cannot be told apart from a note, so stop instead of guessing.
	botID := m.ensureBotUserID(ctx)
	if botID == "" {
		return failed("Not added to Libredesk: the bridge could not identify its Slack bot.", errors.New("slack bot user ID unknown"))
	}
	respond := strings.Contains(ev.Text, "<@"+botID+">")
	content := mrkdwnToHTML(stripMention(ev.Text, botID), func(id string) string {
		if u, err := m.slackUser(ctx, id); err == nil {
			return u.Name
		}
		return ""
	})

	media, err := m.uploadFiles(ctx, ev.Files, agent.ID)
	if err != nil {
		return failed("Not added to Libredesk: a file could not be copied.", err)
	}
	if content == "" && len(media) == 0 {
		return nil
	}

	var msg cmodels.Message
	if respond {
		conv, err := m.convs.GetConversation(0, t.ConversationUUID, "")
		if err != nil {
			return failed("Not sent to the customer: the conversation could not be loaded.", err)
		}
		var to []string
		if conv.InboxChannel == "email" && conv.Contact.Email.String != "" {
			to = []string{conv.Contact.Email.String}
		}
		msg, err = m.convs.QueueReply(media, conv.InboxID, agent.ID, conv.ContactID, conv.UUID, content, to, nil, nil, "", map[string]any{"slack_ts": ev.TS})
		if err != nil {
			return failed("Not sent to the customer: "+err.Error()+".", err)
		}
	} else {
		msg, err = m.convs.SendPrivateNote(media, agent.ID, t.ConversationUUID, content, nil)
		if err != nil {
			return failed("Not added to Libredesk: "+err.Error()+".", err)
		}
	}
	// The message exists in Libredesk; a failure here only risks posting it back to Slack.
	if err := m.store.setMessageID(claimID, msg.ID); err != nil {
		m.lo.Error("recording slack message", "message_id", msg.ID, "error", err)
	}
	reaction := reactionNote
	if respond {
		reaction = reactionReply
	}
	if err := m.api.addReaction(ctx, ev.Channel, ev.TS, reaction); err != nil {
		m.lo.Warn("adding slack reaction", "error", err)
	}
	return nil
}

// agentFor returns the Libredesk agent for a Slack message: the agent with the Slack user's email,
// or the configured AI agent for the Slack AI bot.
func (m *Manager) agentFor(ctx context.Context, ev slackEvent, isAI bool) (umodels.User, string, error) {
	email := m.cfg.AIAgentEmail
	if !isAI {
		u, err := m.slackUser(ctx, ev.User)
		if err != nil {
			return umodels.User{}, "", err
		}
		email = u.Email
		if alias, ok := m.cfg.AgentEmailAliases[strings.ToLower(email)]; ok {
			email = alias
		}
	}
	if email == "" {
		return umodels.User{}, "", errors.New("slack user has no email")
	}
	agent, err := m.users.GetAgent(0, email)
	if err != nil {
		return umodels.User{}, email, err
	}
	if !agent.Enabled {
		return umodels.User{}, email, fmt.Errorf("agent %d is disabled", agent.ID)
	}
	return agent, email, nil
}

// uploadFiles copies Slack files into Libredesk media, owned by the agent so they link to the agent's message.
func (m *Manager) uploadFiles(ctx context.Context, files []slackFile, agentID int) ([]mmodels.Media, error) {
	var out []mmodels.Media
	for _, f := range files {
		if f.URLPrivateDownload == "" {
			continue
		}
		data, err := m.api.download(ctx, f.URLPrivateDownload)
		if err != nil {
			return nil, err
		}
		id := uuid.New().String()
		_, contentType, err := m.media.Upload(id, f.Mimetype, bytes.NewReader(data))
		if err != nil {
			return nil, err
		}
		med, err := m.media.Insert(null.StringFrom(attachment.DispositionAttachment), orDefault(f.Name, "file"), contentType, "", null.String{}, id, null.Int{}, len(data), []byte("{}"), false, null.IntFrom(agentID))
		if err != nil {
			return nil, err
		}
		out = append(out, med)
	}
	return out, nil
}

// replyInThread tells the Slack user why their message did not reach Libredesk.
func (m *Manager) replyInThread(ctx context.Context, ev slackEvent, text string) {
	if _, err := m.api.postMessage(ctx, ev.Channel, ev.ThreadTS, ":warning: "+escapeMrkdwn(text)); err != nil {
		m.lo.Error("posting slack bridge error to thread", "error", err)
	}
}
