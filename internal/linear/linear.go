// Package linear links Linear issues to conversations and adds Linear comments to them as private
// notes (Clyr fork addition).
//
// An agent links an issue by URL or identifier (ENG-123); the issue is looked up through the Linear
// GraphQL API. Linear sends Comment and Issue webhooks to POST /webhooks/linear. A new comment on a
// linked issue becomes a private note, by the system user, in every conversation the issue is linked
// to. An Issue update refreshes the stored title and state.
package linear

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strings"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/dbutil"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

//go:embed queries.sql
var efs embed.FS

var (
	// ErrInvalidRef means the text is not a Linear issue URL or identifier.
	ErrInvalidRef = errors.New("not a Linear issue URL or identifier")
	// ErrIssueNotFound means Linear has no issue with that identifier, or the API key cannot see it.
	ErrIssueNotFound = errors.New("linear issue not found")
	// ErrBadSignature means the webhook signature or timestamp did not check out.
	ErrBadSignature = errors.New("invalid linear webhook signature")
	// ErrBadPayload means a signed webhook body is not valid JSON.
	ErrBadPayload = errors.New("invalid linear webhook payload")
)

// Config holds the settings from the [linear] config section.
type Config struct {
	APIKey        string
	WebhookSecret string
	// APIURL overrides the GraphQL endpoint (tests).
	APIURL string
}

// Enabled reports whether both keys are set.
func (c Config) Enabled() bool {
	return c.APIKey != "" && c.WebhookSecret != ""
}

// Link is a Linear issue linked to a conversation.
type Link struct {
	ID             int       `db:"id" json:"id"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	ConversationID int       `db:"conversation_id" json:"conversation_id"`
	IssueID        string    `db:"issue_id" json:"issue_id"`
	Identifier     string    `db:"identifier" json:"identifier"`
	URL            string    `db:"url" json:"url"`
	Title          string    `db:"title" json:"title"`
	State          string    `db:"state" json:"state"`
	CreatedBy      null.Int  `db:"created_by" json:"created_by"`
}

type noteSender interface {
	SendPrivateNote(media []mmodels.Media, senderID int, conversationUUID, content string, mentions []cmodels.MentionInput) (cmodels.Message, error)
}

type queries struct {
	GetLinks        *sqlx.Stmt `query:"get-links"`
	UpsertLink      *sqlx.Stmt `query:"upsert-link"`
	DeleteLink      *sqlx.Stmt `query:"delete-link"`
	GetLinksByIssue *sqlx.Stmt `query:"get-links-by-issue"`
	UpdateIssue     *sqlx.Stmt `query:"update-issue"`
	ClaimComment    *sqlx.Stmt `query:"claim-comment"`
	CommentNoted    *sqlx.Stmt `query:"comment-noted"`
	SetCommentMsg   *sqlx.Stmt `query:"set-comment-message"`
	ReleaseComment  *sqlx.Stmt `query:"release-comment"`
}

// Manager links issues and handles Linear webhooks.
type Manager struct {
	cfg          Config
	q            queries
	lo           *logf.Logger
	http         *http.Client
	notes        noteSender
	systemUserID int
}

// New creates the manager. Notes from Linear are sent by systemUserID.
func New(cfg Config, db *sqlx.DB, lo *logf.Logger, notes noteSender, systemUserID int) (*Manager, error) {
	if !cfg.Enabled() {
		return nil, errors.New("linear needs api_key and webhook_secret")
	}
	if cfg.APIURL == "" {
		cfg.APIURL = defaultAPIURL
	}
	var q queries
	if err := dbutil.ScanSQLFile("queries.sql", &q, db, efs); err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg, q: q, lo: lo, http: &http.Client{Timeout: 10 * time.Second}, notes: notes, systemUserID: systemUserID}, nil
}

// Links returns the issues linked to a conversation.
func (m *Manager) Links(conversationID int) ([]Link, error) {
	links := []Link{}
	err := m.q.GetLinks.Select(&links, conversationID)
	return links, err
}

// Link looks the issue up in Linear and links it to the conversation. ref is an issue URL or identifier.
func (m *Manager) Link(ctx context.Context, conversationID int, ref string, userID int) (Link, error) {
	identifier, ok := ParseIssueRef(ref)
	if !ok {
		return Link{}, ErrInvalidRef
	}
	issue, err := m.fetchIssue(ctx, identifier)
	if err != nil {
		return Link{}, err
	}
	var l Link
	err = m.q.UpsertLink.Get(&l, conversationID, issue.ID, issue.Identifier, issue.URL, issue.Title, issue.State.Name, null.NewInt(userID, userID > 0))
	return l, err
}

// Unlink removes a link from the conversation.
func (m *Manager) Unlink(conversationID, linkID int) error {
	_, err := m.q.DeleteLink.Exec(linkID, conversationID)
	return err
}

// webhookPayload is the envelope Linear sends for every data change webhook.
type webhookPayload struct {
	Action           string          `json:"action"`
	Type             string          `json:"type"`
	URL              string          `json:"url"`
	WebhookTimestamp int64           `json:"webhookTimestamp"`
	Actor            *namedEntity    `json:"actor"`
	Data             json.RawMessage `json:"data"`
}

type namedEntity struct {
	Name string `json:"name"`
}

type commentData struct {
	ID      string       `json:"id"`
	Body    string       `json:"body"`
	IssueID string       `json:"issueId"`
	User    *namedEntity `json:"user"`
}

type issueData struct {
	ID         string       `json:"id"`
	Identifier string       `json:"identifier"`
	Title      string       `json:"title"`
	URL        string       `json:"url"`
	State      *namedEntity `json:"state"`
}

// HandleWebhook checks the signature and handles one Linear webhook. A returned error other than
// ErrBadSignature means the caller should answer non-2xx so Linear sends it again.
func (m *Manager) HandleWebhook(body []byte, signature string, now time.Time) error {
	if !VerifySignature(m.cfg.WebhookSecret, body, signature) {
		return ErrBadSignature
	}
	var p webhookPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return ErrBadPayload
	}
	if !freshTimestamp(p.WebhookTimestamp, now) {
		return ErrBadSignature
	}
	switch {
	case p.Type == "Comment" && p.Action == "create":
		var c commentData
		if err := json.Unmarshal(p.Data, &c); err != nil {
			return ErrBadPayload
		}
		author := ""
		if c.User != nil {
			author = c.User.Name
		} else if p.Actor != nil {
			author = p.Actor.Name
		}
		return m.addComment(c, author, p.URL)
	case p.Type == "Issue" && p.Action == "update":
		var i issueData
		if err := json.Unmarshal(p.Data, &i); err != nil {
			return ErrBadPayload
		}
		state := ""
		if i.State != nil {
			state = i.State.Name
		}
		_, err := m.q.UpdateIssue.Exec(i.ID, i.Identifier, i.URL, i.Title, state)
		return err
	}
	return nil
}

// addComment adds the comment as a note to every conversation its issue is linked to, once each.
func (m *Manager) addComment(c commentData, author, commentURL string) error {
	if c.ID == "" || c.IssueID == "" || strings.TrimSpace(c.Body) == "" {
		return nil
	}
	var links []struct {
		ConversationID   int    `db:"conversation_id"`
		Identifier       string `db:"identifier"`
		ConversationUUID string `db:"conversation_uuid"`
	}
	if err := m.q.GetLinksByIssue.Select(&links, c.IssueID); err != nil {
		return err
	}
	marker := noteMarker(commentURL)
	for _, l := range links {
		var claimID int
		err := m.q.ClaimComment.Get(&claimID, c.ID, l.ConversationID, marker)
		if errors.Is(err, sql.ErrNoRows) {
			var noted bool
			if err := m.q.CommentNoted.Get(&noted, c.ID, l.ConversationID, marker); err != nil {
				return err
			}
			if !noted {
				// Another request holds a fresh claim. Fail so Linear retries; by then the note is there,
				// or the claim is stale and the retry takes it over.
				return fmt.Errorf("linear comment %s is being added to conversation %d", c.ID, l.ConversationID)
			}
			continue
		}
		if err != nil {
			return err
		}
		msg, err := m.notes.SendPrivateNote(nil, m.systemUserID, l.ConversationUUID, noteHTML(l.Identifier, author, c.Body, commentURL), nil)
		if err != nil {
			// Release the claim so Linear's retry adds the note.
			if _, rerr := m.q.ReleaseComment.Exec(claimID); rerr != nil {
				m.lo.Error("releasing linear comment claim", "comment_id", c.ID, "error", rerr)
			}
			return fmt.Errorf("adding linear comment to conversation %d: %w", l.ConversationID, err)
		}
		if _, err := m.q.SetCommentMsg.Exec(claimID, msg.ID); err != nil {
			m.lo.Error("recording linear comment note", "comment_id", c.ID, "message_id", msg.ID, "error", err)
		}
	}
	return nil
}

// noteMarker returns the text that only this comment's note contains: the link to the comment
// (Linear's comment URLs end in #comment-<id>). It is "" when the URL does not name one comment.
func noteMarker(commentURL string) string {
	if !strings.HasPrefix(commentURL, "https://") || !strings.Contains(commentURL, "#comment-") {
		return ""
	}
	return strings.ReplaceAll(`href="`+html.EscapeString(commentURL)+`"`, "{{", "&#123;&#123;")
}

// noteHTML renders "Linear ENG-123 · author:" followed by the comment's markdown as HTML.
func noteHTML(identifier, author, body, commentURL string) string {
	label := html.EscapeString("Linear " + identifier)
	if strings.HasPrefix(commentURL, "https://") {
		label = `<a href="` + html.EscapeString(commentURL) + `">` + label + `</a>`
	}
	if author == "" {
		author = "Linear"
	}
	out := "<p><strong>" + label + "</strong> · " + html.EscapeString(author) + ":</p>" + stringutil.Markdown2HTML(body)
	// SendPrivateNote renders the content as a template; a comment must not run template code.
	return strings.ReplaceAll(out, "{{", "&#123;&#123;")
}
