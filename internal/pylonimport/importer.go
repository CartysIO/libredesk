package pylonimport

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/media"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Meta keys on Libredesk rows that record where a row came from. They make the import idempotent.
const (
	// conversations.meta: the Pylon issue this conversation was created from.
	MetaIssueID = "pylon_issue_id"
	// conversations.meta: the Pylon issue that was merged into this existing conversation.
	MetaMergedIssueID = "pylon_merged_issue_id"
	// conversation_messages.meta: the Pylon message this message was created from.
	MetaMessageID = "pylon_message_id"
	// users.custom_attributes of agents the import created for Pylon users with no Libredesk agent.
	MetaUserID = "pylon_user_id"
	// users.custom_attributes of contacts the import created without an email address.
	MetaContactID = "pylon_contact_id"
	// conversations.meta of a conversation that the import renumbered: its reference number before.
	MetaPreviousReference = "pre_pylon_import_reference_number"
	// conversations.meta: when the import created the conversation. Time triggers skip it until it has newer activity.
	MetaImportedAt = "pylon_imported_at"
)

// getter runs a query that returns one row: *sqlx.DB or *sqlx.Tx.
type getter interface {
	Get(dest any, query string, args ...any) error
}

// Options control one import run.
type Options struct {
	DryRun bool
	// Limit is the most conversations to import (new or merged) in this run, 0 for all.
	Limit int
	// Since skips issues created before it.
	Since time.Time
	// InboxID is the email inbox the conversations go into, 0 to use the only email inbox.
	InboxID int
	// ReferenceFloor is the lowest reference number new conversations get after the import.
	ReferenceFloor int
	// AgentAliases maps a Pylon user email to a Libredesk agent email.
	AgentAliases map[string]string
	// MediaStore is the media.store value of uploaded files ("s3" or "fs").
	MediaStore string
	// AllowMissingFiles imports even when some files could not be downloaded; their URLs stay in message meta.
	AllowMissingFiles bool
}

// Importer writes Pylon data into a Libredesk database. It writes rows directly with SQL, in one transaction per
// conversation, and calls no Libredesk manager: no email, automation, SLA, CSAT, webhook, Slack, notification or
// websocket event is triggered by the rows it writes.
type Importer struct {
	DB    *sqlx.DB
	Media *media.Manager
	// Files returns the bytes of a downloaded Pylon file, os.ErrNotExist when it is not in the cache.
	Files func(url string) ([]byte, error)
	// Download, when set, fetches the files of the issues a real run imports and returns the URLs that failed.
	Download func(urls []string) map[string]error
	Opt      Options
	Out      io.Writer

	statusIDs    map[string]int
	inboxID      int
	agentByEmail map[string]int
	agentByPylon map[string]int
	// Agents this import created, by display name: one person with two Pylon users gets one agent.
	agentByName   map[string]int
	enabledAgents []int
	contactByMail map[string]int
	contactByID   map[string]int
	orgByAccount  map[string]int
	tagIDs        map[string]int
	imported      map[string]int
	usedSourceIDs map[string]int
	// Per conversation being written: Message-IDs not committed yet, and file key -> content id of its uploads.
	pendingSourceIDs map[string]int
	convFiles        map[string]string
	convUploads      []string
	agentEnabled     map[int]bool
	users            map[string]User
	contacts         map[string]Contact
	accounts         map[string]Account
	rep              *Report
}

// Report counts what a run did, or would do in a dry run.
type Report struct {
	DryRun                bool
	Organizations         int
	OrganizationsNew      int
	Contacts              int
	ContactsNew           int
	Memberships           int
	Agents                []string
	AgentsNew             int
	Issues                int
	Conversations         int
	Merged                int
	AlreadyImported       int
	NotSelected           int
	Messages              int
	PublicMessages        int
	Notes                 int
	SkippedMessages       int
	Attachments           int
	InlineImages          int
	MissingFiles          int
	Duplicates            []string
	Renumbers             []string
	SequenceFloor         int
	Samples               []string
	FailedConversations   []string
	TimeTriggerRulesFound int
}

// Run imports d. In a dry run it reads the database and prints what it would write, and writes nothing.
func (im *Importer) Run(ctx context.Context, d Data) (*Report, error) {
	im.rep = &Report{DryRun: im.Opt.DryRun}
	if im.Out == nil {
		im.Out = io.Discard
	}
	if !im.Opt.DryRun && im.Media == nil {
		return nil, errors.New("a media store is required to import files")
	}
	im.index(d)
	if err := im.load(d); err != nil {
		return nil, err
	}

	issues := slices.Clone(d.Issues)
	sort.Slice(issues, func(i, j int) bool { return issues[i].Number < issues[j].Number })
	im.rep.Issues = len(issues)

	// Duplicates are matched over every issue, so a merge target keeps its match on later runs.
	matches, err := im.matchDuplicates(issues, d.Messages)
	if err != nil {
		return nil, err
	}

	var selected []Issue
	for _, is := range issues {
		switch {
		case im.imported[is.ID] > 0:
			im.rep.AlreadyImported++
		case !im.Opt.Since.IsZero() && is.CreatedAt.Before(im.Opt.Since):
			im.rep.NotSelected++
		case im.Opt.Limit > 0 && len(selected) >= im.Opt.Limit:
			im.rep.NotSelected++
		default:
			selected = append(selected, is)
		}
	}

	maxNumber := 0
	numbers := map[string]int{}
	for _, is := range issues {
		maxNumber = max(maxNumber, is.Number)
		numbers[is.ID] = is.Number
	}
	// Every matched conversation takes its issue's number now, also when the issue is imported on a later run.
	mergedInto := map[int]int{}
	isSelected := map[string]bool{}
	for _, is := range selected {
		isSelected[is.ID] = true
	}
	for id, m := range matches {
		if im.imported[id] > 0 {
			continue
		}
		mergedInto[m.conv.ID] = numbers[id]
		when := "merged"
		if !isSelected[id] {
			when = "merged when that issue is imported"
		}
		im.rep.Duplicates = append(im.rep.Duplicates, fmt.Sprintf("Pylon #%d is Libredesk #%s (%s): %s", numbers[id], m.conv.ReferenceNumber, m.by, when))
	}
	sort.Strings(im.rep.Duplicates)

	existing, err := im.existingConversations()
	if err != nil {
		return nil, err
	}
	renumbers := PlanRenumbers(existing, maxNumber, mergedInto)
	im.rep.SequenceFloor = ReferenceFloor(im.Opt.ReferenceFloor, maxNumber)

	if err := im.countTimeTriggerRules(); err != nil {
		return nil, err
	}

	if im.Opt.DryRun {
		for _, r := range renumbers {
			to := r.To
			if to == "" {
				to = fmt.Sprintf("next sequence value (>= %d)", im.rep.SequenceFloor)
			}
			im.rep.Renumbers = append(im.rep.Renumbers, fmt.Sprintf("Libredesk #%s -> #%s", r.From, to))
		}
		im.planDirectory(d)
		for _, is := range selected {
			_, merge := matches[is.ID]
			im.countIssue(is, d.Messages[is.ID], merge)
		}
		im.sample(selected, d.Messages, matches)
		return im.rep, nil
	}

	// Download the selected issues' files first: a conversation is written with all its files or not at all.
	if im.Download != nil {
		seen := map[string]bool{}
		var urls []string
		for _, is := range selected {
			for _, m := range orderedMessages(is, d.Messages[is.ID]) {
				for _, u := range m.FileURLs {
					if k := FileKey(u); !seen[k] {
						seen[k] = true
						urls = append(urls, u)
					}
				}
			}
		}
		failed := im.Download(urls)
		fmt.Fprintf(im.Out, "pylon import: %d files, %d could not be downloaded\n", len(urls), len(failed))
		if len(failed) > 0 && !im.Opt.AllowMissingFiles {
			var sample []string
			for u, err := range failed {
				if len(sample) < 5 {
					base, _, _ := strings.Cut(u, "?")
					sample = append(sample, fmt.Sprintf("%s: %v", base, err))
				}
			}
			return im.rep, fmt.Errorf("%d files could not be downloaded (rerun to retry, or pass --pylon-allow-missing-files to import without them): %s",
				len(failed), strings.Join(sample, "; "))
		}
	}

	if err := im.applyReferences(renumbers); err != nil {
		return im.rep, fmt.Errorf("reference numbers: %w", err)
	}
	if err := im.ensureQuestionTypeAttribute(); err != nil {
		return im.rep, err
	}
	if err := im.writeDirectory(d); err != nil {
		return im.rep, err
	}
	start := time.Now()
	for n, is := range selected {
		if err := ctx.Err(); err != nil {
			return im.rep, err
		}
		var target *ExistingConversation
		if m, ok := matches[is.ID]; ok {
			target = &m.conv
		}
		counts := *im.rep
		if err := im.importIssue(is, d.Messages[is.ID], target); err != nil {
			// The transaction rolled back: so do its counts.
			im.rep.Messages, im.rep.PublicMessages, im.rep.Notes = counts.Messages, counts.PublicMessages, counts.Notes
			im.rep.SkippedMessages, im.rep.Attachments, im.rep.InlineImages, im.rep.MissingFiles = counts.SkippedMessages, counts.Attachments, counts.InlineImages, counts.MissingFiles
			im.rep.FailedConversations = append(im.rep.FailedConversations, fmt.Sprintf("Pylon #%d: %v", is.Number, err))
			fmt.Fprintf(im.Out, "pylon import: #%d failed: %v\n", is.Number, err)
			continue
		}
		if (n+1)%100 == 0 {
			fmt.Fprintf(im.Out, "pylon import: %d of %d conversations (%s)\n", n+1, len(selected), time.Since(start).Round(time.Second))
		}
	}
	im.sample(selected, d.Messages, matches)
	return im.rep, nil
}

// index builds lookups by Pylon id, adding the users and contacts that only appear as message authors.
func (im *Importer) index(d Data) {
	im.users = map[string]User{}
	for _, u := range d.Users {
		im.users[u.ID] = u
	}
	im.contacts = map[string]Contact{}
	for _, c := range d.Contacts {
		im.contacts[c.ID] = c
	}
	im.accounts = map[string]Account{}
	for _, a := range d.Accounts {
		im.accounts[a.ID] = a
	}
	for _, msgs := range d.Messages {
		for _, m := range msgs {
			if m.Author == nil {
				continue
			}
			if u := m.Author.User; u != nil && u.ID != "" {
				if _, ok := im.users[u.ID]; !ok {
					im.users[u.ID] = User{ID: u.ID, Name: m.Author.Name, Email: u.Email}
				}
			}
			if c := m.Author.Contact; c != nil && c.ID != "" {
				if _, ok := im.contacts[c.ID]; !ok {
					im.contacts[c.ID] = Contact{ID: c.ID, Name: m.Author.Name, Email: c.Email}
				}
			}
		}
	}
}

func (im *Importer) load(d Data) error {
	var statuses []struct {
		ID       int    `db:"id"`
		Name     string `db:"name"`
		Category string `db:"category"`
	}
	if err := im.DB.Select(&statuses, `SELECT id, name, category FROM conversation_statuses`); err != nil {
		return fmt.Errorf("statuses: %w", err)
	}
	im.statusIDs = map[string]int{}
	for _, s := range statuses {
		im.statusIDs[s.Name] = s.ID
	}
	for _, n := range []string{StatusOpen, StatusOnYou, StatusOnCustomer, StatusSnoozed, StatusResolved} {
		if im.statusIDs[n] == 0 {
			return fmt.Errorf("conversation status %q is missing", n)
		}
	}

	if im.Opt.InboxID > 0 {
		if err := im.DB.Get(&im.inboxID, `SELECT id FROM inboxes WHERE id = $1 AND channel = 'email' AND deleted_at IS NULL`, im.Opt.InboxID); err != nil {
			return fmt.Errorf("email inbox %d not found: %w", im.Opt.InboxID, err)
		}
	} else {
		var ids []int
		if err := im.DB.Select(&ids, `SELECT id FROM inboxes WHERE channel = 'email' AND deleted_at IS NULL ORDER BY id`); err != nil {
			return err
		}
		if len(ids) != 1 {
			return fmt.Errorf("found %d email inboxes: pass --pylon-inbox <id>", len(ids))
		}
		im.inboxID = ids[0]
	}

	var agents []struct {
		ID      int            `db:"id"`
		Email   sql.NullString `db:"email"`
		Enabled bool           `db:"enabled"`
		PylonID sql.NullString `db:"pylon_id"`
		Name    string         `db:"name"`
	}
	if err := im.DB.Select(&agents, `SELECT id, email, enabled, custom_attributes->>'`+MetaUserID+`' AS pylon_id,
		TRIM(first_name || ' ' || COALESCE(last_name, '')) AS name
		FROM users WHERE type = 'agent' AND deleted_at IS NULL ORDER BY id`); err != nil {
		return fmt.Errorf("agents: %w", err)
	}
	im.agentByEmail = map[string]int{}
	im.agentByPylon = map[string]int{}
	im.agentByName = map[string]int{}
	im.agentEnabled = map[int]bool{}
	for _, a := range agents {
		im.agentEnabled[a.ID] = a.Enabled
		if e := NormalizeEmail(a.Email.String); e != "" {
			im.agentByEmail[e] = a.ID
		}
		if a.PylonID.Valid {
			im.agentByPylon[a.PylonID.String] = a.ID
			im.agentByName[strings.ToLower(a.Name)] = a.ID
		}
		if a.Enabled {
			im.enabledAgents = append(im.enabledAgents, a.ID)
		}
	}

	var contacts []struct {
		ID      int            `db:"id"`
		Email   sql.NullString `db:"email"`
		PylonID sql.NullString `db:"pylon_id"`
	}
	// Contacts the Clyr backfill created (with an external id) win over email-only contacts with the same email.
	if err := im.DB.Select(&contacts, `SELECT id, email, custom_attributes->>'`+MetaContactID+`' AS pylon_id
		FROM users WHERE type = 'contact' AND deleted_at IS NULL ORDER BY (external_user_id IS NULL), id`); err != nil {
		return fmt.Errorf("contacts: %w", err)
	}
	im.contactByMail = map[string]int{}
	im.contactByID = map[string]int{}
	for _, c := range contacts {
		if e := NormalizeEmail(c.Email.String); e != "" {
			if _, ok := im.contactByMail[e]; !ok {
				im.contactByMail[e] = c.ID
			}
		}
		if c.PylonID.Valid {
			im.contactByID[c.PylonID.String] = c.ID
		}
	}

	var orgs []struct {
		ID         int    `db:"id"`
		ExternalID string `db:"external_id"`
	}
	if err := im.DB.Select(&orgs, `SELECT id, external_id FROM organizations WHERE external_id IS NOT NULL`); err != nil {
		return fmt.Errorf("organizations: %w", err)
	}
	byExt := map[string]int{}
	for _, o := range orgs {
		byExt[o.ExternalID] = o.ID
	}
	im.orgByAccount = map[string]int{}
	for _, a := range d.Accounts {
		if id := byExt[OrgExternalID(a)]; id > 0 {
			im.orgByAccount[a.ID] = id
		}
	}

	var tags []struct {
		ID   int    `db:"id"`
		Name string `db:"name"`
	}
	if err := im.DB.Select(&tags, `SELECT id, name FROM tags`); err != nil {
		return err
	}
	im.tagIDs = map[string]int{}
	for _, t := range tags {
		im.tagIDs[t.Name] = t.ID
	}

	var done []struct {
		IssueID string `db:"issue_id"`
		ID      int    `db:"id"`
	}
	if err := im.DB.Select(&done, `SELECT COALESCE(meta->>'`+MetaIssueID+`', meta->>'`+MetaMergedIssueID+`') AS issue_id, id
		FROM conversations WHERE meta ? '`+MetaIssueID+`' OR meta ? '`+MetaMergedIssueID+`'`); err != nil {
		return fmt.Errorf("imported conversations: %w", err)
	}
	im.imported = map[string]int{}
	for _, r := range done {
		im.imported[r.IssueID] = r.ID
	}

	// Message-IDs of Pylon messages that are already in Libredesk, with the conversation that has them.
	var ids []string
	for _, msgs := range d.Messages {
		for _, m := range msgs {
			if m.EmailInfo != nil {
				if id := stringutil.NormalizeMessageID(m.EmailInfo.MessageID); id != "" {
					ids = append(ids, id)
				}
			}
		}
	}
	var used []struct {
		SourceID string `db:"source_id"`
		ConvID   int    `db:"conversation_id"`
	}
	if err := im.DB.Select(&used, `SELECT DISTINCT ON (source_id) source_id, conversation_id FROM conversation_messages
		WHERE source_id = ANY($1) ORDER BY source_id, id`, pq.Array(ids)); err != nil {
		return fmt.Errorf("message ids: %w", err)
	}
	im.usedSourceIDs = map[string]int{}
	for _, u := range used {
		im.usedSourceIDs[u.SourceID] = u.ConvID
	}
	return nil
}

type duplicate struct {
	conv ExistingConversation
	by   string
}

// existingConversations returns the conversations this import did not create (merged ones count as existing).
func (im *Importer) existingConversations() ([]ExistingConversation, error) {
	var rows []struct {
		ID        int            `db:"id"`
		Ref       string         `db:"reference_number"`
		Subject   sql.NullString `db:"subject"`
		Email     sql.NullString `db:"email"`
		CreatedAt time.Time      `db:"created_at"`
		Merged    bool           `db:"merged"`
	}
	if err := im.DB.Select(&rows, `SELECT c.id, c.reference_number, c.subject, u.email, c.created_at, c.meta ? '`+MetaMergedIssueID+`' AS merged
		FROM conversations c JOIN users u ON u.id = c.contact_id
		WHERE NOT (c.meta ? '`+MetaIssueID+`') ORDER BY c.id`); err != nil {
		return nil, fmt.Errorf("existing conversations: %w", err)
	}
	out := make([]ExistingConversation, 0, len(rows))
	for _, r := range rows {
		out = append(out, ExistingConversation{ID: r.ID, ReferenceNumber: r.Ref, Subject: r.Subject.String, ContactEmail: r.Email.String, CreatedAt: r.CreatedAt, Merged: r.Merged})
	}
	return out, nil
}

// matchDuplicates finds the existing Libredesk conversations that hold a Pylon issue already (both tools read
// support@clyr.io). An issue matches a conversation that has one of its email Message-IDs, else a conversation of
// the same contact with the same subject that started within DuplicateWindow. Each conversation matches one issue.
func (im *Importer) matchDuplicates(issues []Issue, msgs map[string][]Message) (map[string]duplicate, error) {
	existing, err := im.existingConversations()
	if err != nil {
		return nil, err
	}
	byID := map[int]ExistingConversation{}
	for _, c := range existing {
		byID[c.ID] = c
	}
	claimed := map[int]bool{}
	out := map[string]duplicate{}
	// An issue merged on an earlier run keeps its conversation.
	for issueID, convID := range im.imported {
		if c, ok := byID[convID]; ok {
			claimed[convID] = true
			out[issueID] = duplicate{conv: c, by: "merged earlier"}
		}
	}
	for _, is := range issues {
		if im.imported[is.ID] > 0 {
			continue
		}
		for _, m := range msgs[is.ID] {
			if m.EmailInfo == nil {
				continue
			}
			convID := im.usedSourceIDs[stringutil.NormalizeMessageID(m.EmailInfo.MessageID)]
			if c, ok := byID[convID]; ok && !claimed[convID] {
				claimed[convID] = true
				out[is.ID] = duplicate{conv: c, by: "same email Message-ID"}
				break
			}
		}
	}
	for _, is := range issues {
		if _, ok := out[is.ID]; ok || im.imported[is.ID] > 0 {
			continue
		}
		email := im.issueContactEmail(is, msgs[is.ID])
		for _, c := range existing {
			if !claimed[c.ID] && SameThread(email, is.Title, is.CreatedAt, c.ContactEmail, c.Subject, c.CreatedAt) {
				claimed[c.ID] = true
				out[is.ID] = duplicate{conv: c, by: "same contact, subject and day"}
				break
			}
		}
	}
	return out, nil
}

func (im *Importer) issueContactEmail(is Issue, msgs []Message) string {
	if is.Requester != nil {
		if c, ok := im.contacts[is.Requester.ID]; ok && c.Email != "" {
			return c.Email
		}
	}
	for _, m := range msgs {
		if m.Author != nil && m.Author.Contact != nil && m.Author.Contact.Email != "" {
			return m.Author.Contact.Email
		}
	}
	return ""
}

func (im *Importer) countTimeTriggerRules() error {
	return im.DB.Get(&im.rep.TimeTriggerRulesFound, `SELECT COUNT(*) FROM automation_rules WHERE type = 'time_trigger' AND enabled`)
}

// applyReferences moves the sequence above the Pylon range and renumbers the existing conversations that would collide.
func (im *Importer) applyReferences(renumbers []Renumber) error {
	tx, err := im.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// is_called = false: the next nextval returns the floor itself.
	if _, err := tx.Exec(`SELECT setval('conversation_reference_number_sequence', $1, false)
		WHERE (SELECT CASE WHEN is_called THEN last_value + 1 ELSE last_value END FROM conversation_reference_number_sequence) < $1`,
		im.rep.SequenceFloor); err != nil {
		return fmt.Errorf("moving the sequence: %w", err)
	}
	// Merge targets first get a temporary number, so no target waits for a number another target still holds.
	for _, r := range renumbers {
		if r.To != "" {
			if _, err := tx.Exec(`UPDATE conversations SET reference_number = 'pylon-tmp-' || id WHERE id = $1`, r.ConversationID); err != nil {
				return err
			}
		}
	}
	for _, r := range renumbers {
		var to string
		// The first number a conversation had is kept: a later run never overwrites it.
		if err := tx.Get(&to, `UPDATE conversations SET
				reference_number = CASE WHEN $2 = '' THEN nextval('conversation_reference_number_sequence')::TEXT ELSE $2 END,
				meta = CASE WHEN meta ? '`+MetaPreviousReference+`' THEN meta
					ELSE jsonb_set(meta, '{`+MetaPreviousReference+`}', to_jsonb($3::TEXT)) END,
				updated_at = NOW()
			WHERE id = $1 RETURNING reference_number`, r.ConversationID, r.To, r.From); err != nil {
			return fmt.Errorf("renumbering #%s: %w", r.From, err)
		}
		im.rep.Renumbers = append(im.rep.Renumbers, fmt.Sprintf("Libredesk #%s -> #%s", r.From, to))
	}
	return tx.Commit()
}

func (im *Importer) ensureQuestionTypeAttribute() error {
	_, err := im.DB.Exec(`INSERT INTO custom_attribute_definitions (name, description, applies_to, key, data_type)
		VALUES ('Question type', 'Question type from Pylon (imported).', 'conversation', 'question_type', 'text')
		ON CONFLICT (key, applies_to) DO NOTHING`)
	return err
}

// agentPlan is how a Pylon user maps to a Libredesk agent: an existing agent, or a disabled one to create.
type agentPlan struct {
	existingID int
	first      string
	last       string
	email      string
}

// planAgent maps a Pylon user to an agent by email (with aliases), else to a disabled agent named after it.
// Created agents have no password and no role and are disabled, so nobody can sign in as them.
func (im *Importer) planAgent(u User) agentPlan {
	emails := append([]string{u.Email}, u.Emails...)
	for _, e := range emails {
		e = NormalizeEmail(e)
		if e == "" {
			continue
		}
		if alias := NormalizeEmail(im.Opt.AgentAliases[e]); alias != "" {
			e = alias
		}
		if id := im.agentByEmail[e]; id > 0 {
			return agentPlan{existingID: id, email: e}
		}
	}
	if id := im.agentByPylon[u.ID]; id > 0 {
		return agentPlan{existingID: id}
	}
	name := Titled(u.Name)
	if name == "" {
		name = "Pylon user"
	}
	p := agentPlan{email: NormalizeEmail(u.Email)}
	p.first, p.last = SplitName(name, "")
	suffix := "(former)"
	if p.email == "" {
		suffix = "(Pylon bot)"
	}
	p.last = strings.TrimSpace(p.last + " " + suffix)
	return p
}

// agentFor returns the agent for a Pylon user, creating a disabled one when needed.
func (im *Importer) agentFor(tx getter, u User) (int, error) {
	if id := im.agentByPylon[u.ID]; id > 0 {
		return id, nil
	}
	p := im.planAgent(u)
	if p.existingID > 0 {
		im.agentByPylon[u.ID] = p.existingID
		return p.existingID, nil
	}
	name := strings.ToLower(strings.TrimSpace(p.first + " " + p.last))
	if id := im.agentByName[name]; id > 0 {
		im.agentByPylon[u.ID] = id
		return id, nil
	}
	var id int
	attrs, _ := json.Marshal(map[string]string{MetaUserID: u.ID})
	if err := tx.Get(&id, `INSERT INTO users (type, enabled, email, first_name, last_name, custom_attributes)
		VALUES ('agent', FALSE, NULLIF($1, ''), $2, $3, $4) RETURNING id`, p.email, p.first, p.last, attrs); err != nil {
		return 0, fmt.Errorf("creating agent for Pylon user %s: %w", u.Name, err)
	}
	im.agentByPylon[u.ID] = id
	im.agentByName[name] = id
	if p.email != "" {
		im.agentByEmail[p.email] = id
	}
	im.rep.AgentsNew++
	im.rep.Agents = append(im.rep.Agents, fmt.Sprintf("%s <%s> -> new disabled agent %q", u.Name, u.Email, strings.TrimSpace(p.first+" "+p.last)))
	return id, nil
}

// authorUser is the Pylon user behind a message, or a stand-in user for an integration author (such as Linear).
func (im *Importer) authorUser(m Message) (User, bool) {
	if m.Author == nil {
		return User{ID: "author:unknown", Name: "Pylon"}, true
	}
	if m.Author.User != nil && m.Author.User.ID != "" {
		u, ok := im.users[m.Author.User.ID]
		if !ok {
			u = User{ID: m.Author.User.ID, Name: m.Author.Name, Email: m.Author.User.Email}
		}
		return u, true
	}
	if m.Author.Contact != nil && (m.Author.Contact.ID != "" || m.Author.Contact.Email != "") {
		return User{}, false
	}
	name := strings.TrimSpace(m.Author.Name)
	if name == "" {
		name = "Pylon"
	}
	if m.Source == "linear" && !strings.Contains(strings.ToLower(name), "linear") {
		name += " via Linear"
	}
	return User{ID: "author:" + strings.ToLower(name), Name: name}, true
}

// planDirectory counts organizations, contacts and memberships for a dry run.
func (im *Importer) planDirectory(d Data) {
	im.rep.Organizations = len(d.Accounts)
	for _, a := range d.Accounts {
		if im.orgByAccount[a.ID] == 0 {
			im.rep.OrganizationsNew++
		}
	}
	members := im.memberships(d)
	im.rep.Contacts = len(im.contacts)
	for _, c := range im.contacts {
		if im.existingContact(c) == 0 {
			im.rep.ContactsNew++
		}
		im.rep.Memberships += len(members[c.ID])
	}
	seen, newNames := map[string]bool{}, map[string]bool{}
	for _, u := range im.users {
		if seen[u.ID] {
			continue
		}
		seen[u.ID] = true
		p := im.planAgent(u)
		if p.existingID > 0 {
			im.rep.Agents = append(im.rep.Agents, fmt.Sprintf("%s <%s> -> agent %d (%s)", u.Name, u.Email, p.existingID, p.email))
		} else {
			name := strings.TrimSpace(p.first + " " + p.last)
			if !newNames[strings.ToLower(name)] && im.agentByName[strings.ToLower(name)] == 0 {
				newNames[strings.ToLower(name)] = true
				im.rep.AgentsNew++
			}
			im.rep.Agents = append(im.rep.Agents, fmt.Sprintf("%s <%s> -> disabled agent %q", u.Name, u.Email, name))
		}
	}
	sort.Strings(im.rep.Agents)
}

// memberships returns, for each Pylon contact id, its account ids: its own account first, then the accounts of
// the issues it opened.
func (im *Importer) memberships(d Data) map[string][]string {
	out := map[string][]string{}
	add := func(contactID, accountID string) {
		if contactID == "" || accountID == "" || slices.Contains(out[contactID], accountID) {
			return
		}
		if _, ok := im.accounts[accountID]; ok {
			out[contactID] = append(out[contactID], accountID)
		}
	}
	for _, c := range im.contacts {
		if c.Account != nil {
			add(c.ID, c.Account.ID)
		}
	}
	for _, is := range d.Issues {
		if is.Requester != nil && is.Account != nil {
			add(is.Requester.ID, is.Account.ID)
		}
	}
	return out
}

func (im *Importer) existingContact(c Contact) int {
	for _, e := range append([]string{c.Email}, c.Emails...) {
		if id := im.contactByMail[NormalizeEmail(e)]; id > 0 {
			return id
		}
	}
	return im.contactByID[c.ID]
}

// writeDirectory creates the organizations, contacts and memberships. It never changes an existing organization's
// name or an existing contact's default organization.
func (im *Importer) writeDirectory(d Data) error {
	tx, err := im.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	im.rep.Organizations = len(d.Accounts)
	for _, a := range d.Accounts {
		if im.orgByAccount[a.ID] > 0 {
			continue
		}
		name := clip(strings.TrimSpace(a.Name), 255)
		if name == "" {
			name = "Unnamed account"
		}
		var id int
		if err := tx.Get(&id, `WITH ins AS (
				INSERT INTO organizations (name, external_id) VALUES ($1, $2)
				ON CONFLICT (external_id) WHERE external_id IS NOT NULL DO NOTHING RETURNING id
			) SELECT id FROM ins UNION ALL SELECT id FROM organizations WHERE external_id = $2 LIMIT 1`,
			name, clip(OrgExternalID(a), 128)); err != nil {
			return fmt.Errorf("organization %s: %w", a.Name, err)
		}
		im.orgByAccount[a.ID] = id
		im.rep.OrganizationsNew++
	}

	members := im.memberships(d)
	ids := make([]string, 0, len(im.contacts))
	for id := range im.contacts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	im.rep.Contacts = len(ids)
	for _, pid := range ids {
		c := im.contacts[pid]
		id, err := im.contactFor(tx, c)
		if err != nil {
			return err
		}
		for i, acc := range members[pid] {
			org := im.orgByAccount[acc]
			res, err := tx.Exec(`INSERT INTO contact_organizations (contact_id, organization_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, id, org)
			if err != nil {
				return fmt.Errorf("membership of %s: %w", c.Email, err)
			}
			if n, _ := res.RowsAffected(); n > 0 {
				im.rep.Memberships++
			}
			if i == 0 {
				if _, err := tx.Exec(`UPDATE users SET organization_id = $2, updated_at = NOW() WHERE id = $1 AND organization_id IS NULL`, id, org); err != nil {
					return err
				}
			}
		}
	}
	return tx.Commit()
}

// contactFor returns the Libredesk contact for a Pylon contact, matched by email, creating it when needed.
func (im *Importer) contactFor(tx getter, c Contact) (int, error) {
	if id := im.existingContact(c); id > 0 {
		return id, nil
	}
	email := NormalizeEmail(c.Email)
	if email == "" && len(c.Emails) > 0 {
		email = NormalizeEmail(c.Emails[0])
	}
	first, last := SplitName(c.Name, email)
	attrs := []byte(`{}`)
	if email == "" {
		attrs, _ = json.Marshal(map[string]string{MetaContactID: c.ID})
	}
	var id int
	if err := tx.Get(&id, `INSERT INTO users (type, email, first_name, last_name, custom_attributes)
		VALUES ('contact', NULLIF($1, ''), $2, NULLIF($3, ''), $4) RETURNING id`, clip(email, 320), first, last, attrs); err != nil {
		return 0, fmt.Errorf("contact %s: %w", email, err)
	}
	if email != "" {
		im.contactByMail[email] = id
	} else {
		im.contactByID[c.ID] = id
	}
	im.rep.ContactsNew++
	return id, nil
}

// requester is the Pylon contact a conversation belongs to: the issue requester, else the first contact author.
func (im *Importer) requester(is Issue, msgs []Message) Contact {
	if is.Requester != nil {
		if c, ok := im.contacts[is.Requester.ID]; ok {
			return c
		}
	}
	for _, m := range msgs {
		if m.Author != nil && m.Author.Contact != nil {
			return im.contacts[m.Author.Contact.ID]
		}
	}
	return Contact{ID: "unknown", Name: "Unknown (Pylon)"}
}

// orderedMessages returns the issue's messages oldest first. An issue without messages gets its body as the first message.
func orderedMessages(is Issue, msgs []Message) []Message {
	out := slices.Clone(msgs)
	if len(out) == 0 && strings.TrimSpace(is.BodyHTML) != "" {
		var author *MessageAuthor
		if is.Requester != nil {
			author = &MessageAuthor{Contact: &Ref{ID: is.Requester.ID}}
		}
		out = append(out, Message{ID: "body:" + is.ID, MessageHTML: is.BodyHTML, Timestamp: is.CreatedAt, Source: is.Source, Author: author})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Timestamp.Before(out[j].Timestamp) })
	// Issue files that no message carries go on the first message.
	if len(out) > 0 && len(is.AttachmentURLs) > 0 {
		have := map[string]bool{}
		for _, m := range out {
			for _, u := range m.FileURLs {
				have[FileKey(u)] = true
			}
		}
		for _, u := range is.AttachmentURLs {
			if !have[FileKey(u)] {
				have[FileKey(u)] = true
				out[0].FileURLs = append(slices.Clone(out[0].FileURLs), u)
			}
		}
	}
	return out
}

func (im *Importer) countIssue(is Issue, msgs []Message, merge bool) {
	if merge {
		im.rep.Merged++
	} else {
		im.rep.Conversations++
	}
	for _, m := range orderedMessages(is, msgs) {
		if merge && m.EmailInfo != nil && im.usedSourceIDs[stringutil.NormalizeMessageID(m.EmailInfo.MessageID)] > 0 {
			im.rep.SkippedMessages++
			continue
		}
		im.rep.Messages++
		if m.IsPrivate {
			im.rep.Notes++
		} else {
			im.rep.PublicMessages++
		}
		im.rep.Attachments += len(m.FileURLs)
		im.rep.InlineImages += len(InlineFileURLs(m.MessageHTML, m.FileURLs))
	}
}

// importIssue writes one issue in one transaction: a new conversation, or new messages in the conversation it duplicates.
func (im *Importer) importIssue(is Issue, msgs []Message, target *ExistingConversation) (err error) {
	msgs = orderedMessages(is, msgs)
	// Message-IDs, content ids and stored files of this transaction; they count only once it commits.
	im.pendingSourceIDs = map[string]int{}
	im.convFiles = map[string]string{}
	im.convUploads = nil
	defer func() {
		if err != nil {
			// The rows rolled back: remove the files stored for them too.
			for _, name := range im.convUploads {
				_ = im.Media.Delete(name)
			}
		}
	}()
	tx, err := im.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	req := im.requester(is, msgs)
	contactID, err := im.contactFor(im.DB, req)
	if err != nil {
		return err
	}
	var orgID sql.NullInt64
	if is.Account != nil && im.orgByAccount[is.Account.ID] > 0 {
		orgID = sql.NullInt64{Int64: int64(im.orgByAccount[is.Account.ID]), Valid: true}
		if _, err := tx.Exec(`INSERT INTO contact_organizations (contact_id, organization_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, contactID, orgID); err != nil {
			return err
		}
	}
	var assignee sql.NullInt64
	if is.Assignee != nil && is.Assignee.ID != "" {
		u, ok := im.users[is.Assignee.ID]
		if !ok {
			u = User{ID: is.Assignee.ID, Name: "Pylon user", Email: is.Assignee.Email}
		}
		id, err := im.agentFor(im.DB, u)
		if err != nil {
			return err
		}
		// An open conversation stays unassigned rather than with a disabled agent, whom no reply would notify.
		if im.agentEnabled[id] || MapStatus(is.State) == StatusResolved {
			assignee = sql.NullInt64{Int64: int64(id), Valid: true}
		}
	}

	status := MapStatus(is.State)
	var resolvedAt *time.Time
	if status == StatusResolved {
		resolvedAt = is.ResolutionTime
		if resolvedAt == nil {
			resolvedAt = is.LatestMessageTime
		}
		if resolvedAt == nil {
			resolvedAt = &is.CreatedAt
		}
	}
	attrs := map[string]string{}
	if qt := QuestionType(is.CustomFields); qt != "" {
		attrs["question_type"] = qt
	}
	attrsJSON, _ := json.Marshal(attrs)
	lastAt := is.CreatedAt
	if n := len(msgs); n > 0 && msgs[n-1].Timestamp.After(lastAt) {
		lastAt = msgs[n-1].Timestamp
	}

	var convID int
	if target == nil {
		meta, _ := json.Marshal(map[string]any{MetaIssueID: is.ID, MetaImportedAt: time.Now().UTC().Format(time.RFC3339),
			"pylon_number": is.Number, "pylon_link": is.Link, "pylon_source": is.Source})
		subject := strings.TrimSpace(is.Title)
		if err := tx.Get(&convID, `INSERT INTO conversations
			(created_at, updated_at, reference_number, contact_id, assigned_user_id, organization_id, inbox_id, status_id,
			 subject, meta, custom_attributes, contact_last_seen_at, resolved_at, last_resolved_at)
			VALUES ($1, $2, $3, $4, $5, COALESCE($6, (SELECT organization_id FROM users WHERE id = $4)), $7, $8,
			 NULLIF($9, ''), $10, $11, $2, $12, $12)
			RETURNING id`,
			is.CreatedAt, lastAt, strconv.Itoa(is.Number), contactID, assignee, orgID, im.inboxID, im.statusIDs[status],
			subject, meta, attrsJSON, resolvedAt); err != nil {
			return fmt.Errorf("conversation: %w", err)
		}
	} else {
		convID = target.ID
		meta, _ := json.Marshal(map[string]any{MetaMergedIssueID: is.ID, "pylon_number": is.Number, "pylon_link": is.Link, "pylon_source": is.Source})
		// Pylon's status and assignee win only when Pylon saw the newest activity; otherwise the live state stays.
		if _, err := tx.Exec(`UPDATE conversations SET
				meta = meta || $2::jsonb,
				custom_attributes = custom_attributes || $3::jsonb,
				organization_id = COALESCE(organization_id, $6),
				status_id = CASE WHEN $8 > COALESCE(last_message_at, 'epoch') THEN $4 ELSE status_id END,
				assigned_user_id = CASE WHEN $8 > COALESCE(last_message_at, 'epoch') THEN COALESCE($5, assigned_user_id) ELSE assigned_user_id END,
				resolved_at = CASE WHEN $8 > COALESCE(last_message_at, 'epoch') THEN COALESCE($7, resolved_at) ELSE resolved_at END,
				last_resolved_at = CASE WHEN $8 > COALESCE(last_message_at, 'epoch') THEN COALESCE($7, last_resolved_at) ELSE last_resolved_at END,
				snoozed_until = CASE WHEN $8 > COALESCE(last_message_at, 'epoch') THEN NULL ELSE snoozed_until END
			WHERE id = $1`, convID, meta, attrsJSON, im.statusIDs[status], assignee, orgID, resolvedAt, lastAt); err != nil {
			return fmt.Errorf("merging into #%s: %w", target.ReferenceNumber, err)
		}
	}

	participants := map[int]bool{}
	if assignee.Valid {
		participants[int(assignee.Int64)] = true
	}
	for _, m := range msgs {
		agentID, err := im.writeMessage(tx, convID, m, target != nil)
		if err != nil {
			return fmt.Errorf("message %s: %w", m.ID, err)
		}
		if agentID > 0 {
			participants[agentID] = true
		}
	}

	for _, t := range is.Tags {
		if !KeepTag(t) {
			continue
		}
		t = clip(strings.TrimSpace(t), 140)
		tagID := im.tagIDs[t]
		if tagID == 0 {
			if err := im.DB.Get(&tagID, `WITH ins AS (INSERT INTO tags (name) VALUES ($1) ON CONFLICT (name) DO NOTHING RETURNING id)
				SELECT id FROM ins UNION ALL SELECT id FROM tags WHERE name = $1 LIMIT 1`, t); err != nil {
				return fmt.Errorf("tag %s: %w", t, err)
			}
		}
		if _, err := tx.Exec(`INSERT INTO conversation_tags (conversation_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, convID, tagID); err != nil {
			return err
		}
		im.tagIDs[t] = tagID
	}
	for id := range participants {
		if _, err := tx.Exec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, convID, id); err != nil {
			return err
		}
	}
	// The history counts as read, so imported conversations do not show as unread to every agent. A merged
	// conversation is live: its unread state stays as it is.
	if target == nil {
		if _, err := tx.Exec(`INSERT INTO conversation_last_seen (conversation_id, user_id, last_seen_at)
			SELECT $1, u, (SELECT COALESCE(MAX(created_at), NOW()) FROM conversation_messages WHERE conversation_id = $1)
			FROM UNNEST($2::BIGINT[]) u
			ON CONFLICT (conversation_id, user_id) DO UPDATE SET last_seen_at = GREATEST(conversation_last_seen.last_seen_at, EXCLUDED.last_seen_at)`,
			convID, pq.Array(im.enabledAgents)); err != nil {
			return err
		}
	}
	if err := refreshSummary(tx, convID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for id, conv := range im.pendingSourceIDs {
		im.usedSourceIDs[id] = conv
	}
	if target != nil {
		im.rep.Merged++
	} else {
		im.rep.Conversations++
	}
	im.imported[is.ID] = convID
	return nil
}

// refreshSummary sets the conversation's last message, reply and inbound timestamps from its messages.
func refreshSummary(tx *sqlx.Tx, convID int) error {
	_, err := tx.Exec(`WITH
		lm AS (SELECT text_content, sender_type, sender_id, created_at FROM conversation_messages
			WHERE conversation_id = $1 AND type != 'activity' ORDER BY created_at DESC, id DESC LIMIT 1),
		li AS (SELECT text_content, sender_type, sender_id, created_at FROM conversation_messages
			WHERE conversation_id = $1 AND type != 'activity' AND private = FALSE ORDER BY created_at DESC, id DESC LIMIT 1)
		UPDATE conversations SET
			last_message = (SELECT text_content FROM lm),
			last_message_sender = (SELECT sender_type FROM lm),
			last_message_sender_id = (SELECT sender_id FROM lm),
			last_message_at = (SELECT created_at FROM lm),
			last_interaction = (SELECT text_content FROM li),
			last_interaction_sender = (SELECT sender_type FROM li),
			last_interaction_sender_id = (SELECT sender_id FROM li),
			last_interaction_at = (SELECT created_at FROM li),
			first_reply_at = (SELECT MIN(created_at) FROM conversation_messages WHERE conversation_id = $1 AND type = 'outgoing' AND private = FALSE),
			last_reply_at = (SELECT MAX(created_at) FROM conversation_messages WHERE conversation_id = $1 AND type = 'outgoing' AND private = FALSE),
			last_inbound_at = (SELECT MAX(created_at) FROM conversation_messages WHERE conversation_id = $1 AND type = 'incoming'),
			waiting_since = CASE WHEN (SELECT category FROM conversation_statuses s WHERE s.id = conversations.status_id) = 'open'
				AND (SELECT sender_type FROM li) = 'contact' THEN (SELECT created_at FROM li) END,
			contact_last_seen_at = COALESCE((SELECT created_at FROM lm), contact_last_seen_at)
		WHERE id = $1`, convID)
	return err
}

// writeMessage inserts one Pylon message with its files. It returns the agent id when an agent wrote it.
// In a merge, a message whose Message-ID is already in the conversation is skipped: it is the same email.
func (im *Importer) writeMessage(tx *sqlx.Tx, convID int, m Message, merge bool) (int, error) {
	var exists bool
	if err := tx.Get(&exists, `SELECT EXISTS (SELECT 1 FROM conversation_messages WHERE conversation_id = $1 AND meta->>'`+MetaMessageID+`' = $2)`, convID, m.ID); err != nil {
		return 0, err
	}
	if exists {
		return 0, nil
	}
	sourceID := ""
	if m.EmailInfo != nil {
		sourceID = stringutil.NormalizeMessageID(m.EmailInfo.MessageID)
	}
	if sourceID != "" && (im.usedSourceIDs[sourceID] > 0 || im.pendingSourceIDs[sourceID] > 0) {
		if merge && im.usedSourceIDs[sourceID] == convID {
			im.rep.SkippedMessages++
			return 0, nil
		}
		// The Message-ID belongs to another conversation: keep it in meta only, so email threading stays unambiguous.
		sourceID = ""
	}
	if m.IsPrivate {
		sourceID = ""
	}

	var (
		msgType    = "incoming"
		status     = "received"
		senderType = "contact"
		senderID   int
		agentID    int
	)
	if u, isAgent := im.authorUser(m); isAgent {
		id, err := im.agentFor(im.DB, u)
		if err != nil {
			return 0, err
		}
		msgType, status, senderType, senderID, agentID = "outgoing", "sent", "agent", id, id
	} else {
		c := im.contacts[m.Author.Contact.ID]
		if c.ID == "" {
			c = Contact{ID: m.Author.Contact.ID, Name: m.Author.Name, Email: m.Author.Contact.Email}
		}
		id, err := im.contactFor(im.DB, c)
		if err != nil {
			return 0, err
		}
		senderID = id
	}

	meta := map[string]any{MetaMessageID: m.ID, "pylon_source": m.Source}
	if m.ThreadID != "" {
		meta["pylon_thread_id"] = m.ThreadID
	}
	if e := m.EmailInfo; e != nil {
		if e.FromEmail != "" {
			meta["from"] = []string{NormalizeEmail(e.FromEmail)}
		}
		meta["to"] = lower(e.ToEmails)
		meta["cc"] = lower(e.CCEmails)
		meta["bcc"] = lower(e.BCCEmails)
		if id := stringutil.NormalizeMessageID(e.MessageID); id != "" {
			meta["email_message_id"] = id
		}
	}

	content := m.MessageHTML
	files, missing, err := im.uploadFiles(m, &content)
	if err != nil {
		return 0, err
	}
	if len(missing) > 0 {
		meta["pylon_missing_files"] = missing
	}
	metaJSON, _ := json.Marshal(meta)

	var msgID int
	if err := tx.Get(&msgID, `INSERT INTO conversation_messages
		(created_at, updated_at, type, status, private, conversation_id, content_type, content, text_content, source_id, sender_id, sender_type, meta)
		VALUES ($1, $1, $2, $3, $4, $5, 'html', $6, $7, NULLIF($8, ''), $9, $10, $11) RETURNING id`,
		m.Timestamp, msgType, status, m.IsPrivate, convID, content, stringutil.HTML2Text(content), sourceID, senderID, senderType, metaJSON); err != nil {
		return 0, err
	}
	if sourceID != "" {
		im.pendingSourceIDs[sourceID] = convID
	}
	for _, f := range files {
		if _, err := tx.Exec(`INSERT INTO media (created_at, updated_at, store, filename, content_type, content_id, model_id, model_type,
				disposition, size, meta, private, uuid)
			VALUES ($1, $1, $2, $3, $4, $5, $6, 'messages', $7, $8, $9, TRUE, $10)`,
			m.Timestamp, im.Opt.MediaStore, f.name, f.contentType, f.contentID, msgID, f.disposition, f.size, f.meta, f.uuid); err != nil {
			return 0, fmt.Errorf("media %s: %w", f.name, err)
		}
	}
	im.rep.Messages++
	if m.IsPrivate {
		im.rep.Notes++
	} else {
		im.rep.PublicMessages++
	}
	return agentID, nil
}

type uploaded struct {
	uuid, name, contentType, contentID, disposition string
	size                                            int
	meta                                            []byte
}

// uploadFiles stores the message's files in the media store. A file shown inline in an <img> becomes an inline
// image: its src is rewritten to cid:ldsk-<uuid>, the form Libredesk resolves to a signed URL when it reads the
// message. Files missing from the cache are returned by URL and left out.
func (im *Importer) uploadFiles(m Message, content *string) ([]uploaded, []string, error) {
	inline := InlineFileURLs(*content, m.FileURLs)
	var (
		out     []uploaded
		missing []string
	)
	for _, u := range m.FileURLs {
		b, err := im.Files(u)
		if errors.Is(err, os.ErrNotExist) || (err == nil && len(b) == 0) {
			missing = append(missing, u)
			im.rep.MissingFiles++
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		id := uuid.NewString()
		name := stringutil.SanitizeFilename(FileName(u))
		_, contentType, err := im.Media.Upload(id, "", bytes.NewReader(b))
		if err != nil {
			return nil, nil, fmt.Errorf("uploading %s: %w", name, err)
		}
		im.convUploads = append(im.convUploads, id)
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
		if slices.Contains(image.Exts, ext) && image.IsImageByContent(bytes.NewReader(b)) {
			if thumb, err := image.CreateThumb(image.DefThumbSize, bytes.NewReader(b)); err == nil {
				if _, _, err := im.Media.Upload(image.ThumbPrefix+id, contentType, thumb); err == nil {
					im.convUploads = append(im.convUploads, image.ThumbPrefix+id)
				}
			}
		}
		meta, _ := json.Marshal(map[string]string{"pylon_file_key": FileKey(u)})
		f := uploaded{uuid: id, name: name, contentType: contentType, disposition: "attachment", size: len(b), meta: meta}
		if src, ok := inline[u]; ok {
			f.disposition = "inline"
			f.contentID = "ldsk-" + id
			*content = strings.ReplaceAll(*content, src, "cid:"+f.contentID)
			im.convFiles[FileKey(u)] = f.contentID
			im.rep.InlineImages++
		}
		im.rep.Attachments++
		out = append(out, f)
	}
	// Quoted history shows images of earlier messages: point them at those messages' media, which Libredesk
	// resolves within the conversation.
	*content = imgSrc.ReplaceAllStringFunc(*content, func(tag string) string {
		sub := imgSrc.FindStringSubmatch(tag)
		src := sub[2] + sub[3]
		if strings.HasPrefix(src, "cid:") {
			return tag
		}
		if cid, ok := im.convFiles[FileKey(html.UnescapeString(src))]; ok {
			return strings.Replace(tag, src, "cid:"+cid, 1)
		}
		return tag
	})
	return out, missing, nil
}

func lower(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s = NormalizeEmail(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// sample describes three conversations of the run: the first, the first merge and the first with internal notes.
func (im *Importer) sample(selected []Issue, msgs map[string][]Message, matches map[string]duplicate) {
	picked := map[string]bool{}
	pick := func(ok func(Issue) bool) {
		for _, is := range selected {
			if !picked[is.ID] && ok(is) {
				picked[is.ID] = true
				im.rep.Samples = append(im.rep.Samples, im.describe(is, msgs[is.ID], matches))
				return
			}
		}
	}
	pick(func(Issue) bool { return true })
	pick(func(is Issue) bool { _, ok := matches[is.ID]; return ok })
	pick(func(is Issue) bool {
		return slices.ContainsFunc(msgs[is.ID], func(m Message) bool { return m.IsPrivate }) && len(msgs[is.ID]) >= 4
	})
}

func (im *Importer) describe(is Issue, msgs []Message, matches map[string]duplicate) string {
	var b strings.Builder
	target := "new conversation"
	if m, ok := matches[is.ID]; ok {
		target = "merged into Libredesk #" + m.conv.ReferenceNumber
	}
	assignee := "unassigned"
	if is.Assignee != nil {
		u := im.users[is.Assignee.ID]
		p := im.planAgent(u)
		assignee = fmt.Sprintf("%s (agent %d)", u.Name, p.existingID)
	}
	req := im.requester(is, msgs)
	// Samples show structure, not content: customer emails are masked and message text is left out.
	fmt.Fprintf(&b, "Pylon #%d: %s, %s -> %s, assignee %s, contact %s, question_type %q, tags %v\n",
		is.Number, target, is.State, MapStatus(is.State), assignee, MaskEmail(req.Email), QuestionType(is.CustomFields), is.Tags)
	for i, m := range orderedMessages(is, msgs) {
		if i == 8 {
			fmt.Fprintf(&b, "    ... %d more\n", len(msgs)-i)
			break
		}
		kind := "customer"
		who := ""
		if u, isAgent := im.authorUser(m); isAgent {
			kind, who = "reply", u.Name
			if m.IsPrivate {
				kind = "note"
			}
		} else if m.Author != nil && m.Author.Contact != nil {
			who = MaskEmail(m.Author.Contact.Email)
		}
		fmt.Fprintf(&b, "    %s %-8s %-28s %d files, %d characters\n", m.Timestamp.Format(time.RFC3339), kind, who, len(m.FileURLs), len(stringutil.HTML2Text(m.MessageHTML)))
	}
	return b.String()
}

// Print writes the report.
func (r *Report) Print(w io.Writer) {
	mode := "IMPORT"
	if r.DryRun {
		mode = "DRY RUN (nothing written)"
	}
	fmt.Fprintf(w, "\n=== Pylon import: %s ===\n", mode)
	fmt.Fprintf(w, "Organizations: %d (%d new)\n", r.Organizations, r.OrganizationsNew)
	fmt.Fprintf(w, "Contacts: %d (%d new), new memberships: %d\n", r.Contacts, r.ContactsNew, r.Memberships)
	fmt.Fprintf(w, "New disabled agents: %d\n", r.AgentsNew)
	fmt.Fprintf(w, "Pylon issues: %d; already imported: %d; not selected (--since/--limit): %d\n", r.Issues, r.AlreadyImported, r.NotSelected)
	fmt.Fprintf(w, "Conversations created: %d; merged into existing duplicates: %d; failed: %d\n", r.Conversations, r.Merged, len(r.FailedConversations))
	fmt.Fprintf(w, "Messages: %d (%d public, %d internal notes); skipped as already present: %d\n", r.Messages, r.PublicMessages, r.Notes, r.SkippedMessages)
	fmt.Fprintf(w, "Files: %d (%d inline images); missing from the cache: %d\n", r.Attachments, r.InlineImages, r.MissingFiles)
	fmt.Fprintf(w, "New conversations get reference numbers from %d up.\n", r.SequenceFloor)
	if r.TimeTriggerRulesFound > 0 {
		fmt.Fprintf(w, "Enabled time-trigger automation rules: %d (they skip imported conversations).\n", r.TimeTriggerRulesFound)
	}
	section := func(title string, lines []string) {
		fmt.Fprintf(w, "%s (%d):\n", title, len(lines))
		for _, l := range lines {
			fmt.Fprintf(w, "  %s\n", strings.TrimRight(l, "\n"))
		}
	}
	section("Agents", r.Agents)
	section("Duplicates", r.Duplicates)
	section("Reference renumbers", r.Renumbers)
	section("Failed", r.FailedConversations)
	section("Samples", r.Samples)
}
