package pylonimport

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation"
	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
)

// SyncOptions control one --pylon-sync run.
type SyncOptions struct {
	DryRun              bool
	Limit               int
	Since               time.Time
	CutoverEnd          time.Time
	AllowMissingFiles   bool
	VerifyIssuesInPylon bool
	AgentAliases        map[string]string
	MediaStore          string
	ReportPath          string
}

// SyncMessageAction is one planned or applied message change in the sync report.
type SyncMessageAction struct {
	PylonMessageID string    `json:"pylon_message_id"`
	Timestamp      time.Time `json:"timestamp"`
	Private        bool      `json:"private,omitempty"`
}

// SyncConversationReport is the per-conversation section of the JSON report.
type SyncConversationReport struct {
	ConversationID        int                 `json:"conversation_id"`
	ReferenceNumber       string              `json:"reference_number"`
	PylonIssueID          string              `json:"pylon_issue_id"`
	PylonImportedAt       *time.Time          `json:"pylon_imported_at,omitempty"`
	Skipped               bool                `json:"skipped,omitempty"`
	SkipReason            string              `json:"skip_reason,omitempty"`
	Warning               string              `json:"warning,omitempty"`
	Error                 string              `json:"error,omitempty"`
	StatusAction          StatusSyncAction    `json:"status_action"`
	StatusSkippedReason   string              `json:"status_skipped_reason,omitempty"`
	StatusFrom            string              `json:"status_from,omitempty"`
	StatusTo              string              `json:"status_to,omitempty"`
	MessagesInserted      []SyncMessageAction `json:"messages_inserted,omitempty"`
	MessagesTagged        []SyncMessageAction `json:"messages_tagged,omitempty"`
	WouldInsertMessages   []SyncMessageAction `json:"would_insert_messages,omitempty"`
	WouldTagMessages      []SyncMessageAction `json:"would_tag_messages,omitempty"`
	WouldUpdateStatus     bool                `json:"would_update_status,omitempty"`
	WouldUpdateStatusFrom string              `json:"would_update_status_from,omitempty"`
	WouldUpdateStatusTo   string              `json:"would_update_status_to,omitempty"`
}

// SyncSummary counts sync outcomes.
type SyncSummary struct {
	Examined         int `json:"examined"`
	Skipped          int `json:"skipped"`
	StatusUpdated    int `json:"status_updated"`
	StatusUnchanged  int `json:"status_unchanged"`
	StatusSkipped    int `json:"status_skipped"`
	MessagesInserted int `json:"messages_inserted"`
	MessagesTagged   int `json:"messages_tagged"`
	Errors           int `json:"errors"`
}

// SyncReport is written to --pylon-sync-report.
type SyncReport struct {
	StartedAt     time.Time                `json:"started_at"`
	FinishedAt    time.Time                `json:"finished_at"`
	DryRun        bool                     `json:"dry_run"`
	CutoverEnd    time.Time                `json:"pylon_cutover_end"`
	ReportPath    string                   `json:"report_path"`
	Summary       SyncSummary              `json:"summary"`
	Conversations []SyncConversationReport `json:"conversations"`
}

// Syncer updates migrated conversations from live Pylon data without touching non-imported rows.
type Syncer struct {
	DB  *sqlx.DB
	Imp *Importer
	Opt SyncOptions
	Out io.Writer
}

type importedConversation struct {
	ID              int          `db:"id"`
	ReferenceNumber string       `db:"reference_number"`
	StatusID        int          `db:"status_id"`
	StatusName      string       `db:"status_name"`
	IssueID         string       `db:"issue_id"`
	ImportedAt      sql.NullTime `db:"imported_at"`
}

// Run syncs every conversation with meta.pylon_issue_id using the provided Pylon snapshot.
func (s *Syncer) Run(ctx context.Context, data Data) (SyncReport, error) {
	rep := SyncReport{
		StartedAt:  time.Now().UTC(),
		DryRun:     s.Opt.DryRun,
		CutoverEnd: s.Opt.CutoverEnd.UTC(),
		ReportPath: s.Opt.ReportPath,
	}
	convos, err := s.loadImportedConversations()
	if err != nil {
		return rep, err
	}
	issueByID := map[string]Issue{}
	for _, is := range data.Issues {
		issueByID[is.ID] = is
	}
	// agentFor and insertMessagePlan update im.rep counters; import Run always initializes this.
	if s.Imp.rep == nil {
		s.Imp.rep = &Report{DryRun: s.Opt.DryRun}
	}
	if err := s.Imp.load(data); err != nil {
		return rep, err
	}
	nameByID := map[int]string{}
	for name, id := range s.Imp.statusIDs {
		nameByID[id] = name
	}

	selected := convos
	if !s.Opt.Since.IsZero() {
		var filtered []importedConversation
		for _, c := range selected {
			is, ok := issueByID[c.IssueID]
			if !ok || is.CreatedAt.Before(s.Opt.Since.UTC()) {
				continue
			}
			filtered = append(filtered, c)
		}
		selected = filtered
	}
	if s.Opt.Limit > 0 && len(selected) > s.Opt.Limit {
		selected = selected[:s.Opt.Limit]
	}

	for _, c := range selected {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		rep.Summary.Examined++
		entry := SyncConversationReport{
			ConversationID:  c.ID,
			ReferenceNumber: c.ReferenceNumber,
			PylonIssueID:    c.IssueID,
			StatusAction:    StatusActionSkipped,
		}
		if c.ImportedAt.Valid {
			t := c.ImportedAt.Time.UTC()
			entry.PylonImportedAt = &t
		}
		is, ok := issueByID[c.IssueID]
		if !ok {
			entry.Skipped = true
			entry.SkipReason = "issue_not_in_pylon_snapshot"
			rep.Summary.Skipped++
			rep.Conversations = append(rep.Conversations, entry)
			continue
		}
		if err := s.syncConversation(ctx, c, is, orderedMessages(is, data.Messages[c.IssueID]), nameByID, &entry); err != nil {
			entry.Error = err.Error()
			rep.Summary.Errors++
		}
		s.accumulateSummary(&rep.Summary, entry)
		rep.Conversations = append(rep.Conversations, entry)
	}
	rep.FinishedAt = time.Now().UTC()
	if err := s.writeReport(rep); err != nil {
		return rep, err
	}
	if rep.Summary.Errors > 0 {
		return rep, fmt.Errorf("%d conversation(s) failed", rep.Summary.Errors)
	}
	return rep, nil
}

func (s *Syncer) accumulateSummary(sum *SyncSummary, e SyncConversationReport) {
	if e.Skipped {
		return
	}
	switch e.StatusAction {
	case StatusActionUpdated:
		sum.StatusUpdated++
	case StatusActionUnchanged:
		sum.StatusUnchanged++
	case StatusActionSkipped:
		if e.StatusSkippedReason != "" || e.Warning != "" {
			sum.StatusSkipped++
		}
	}
	sum.MessagesInserted += len(e.MessagesInserted)
	sum.MessagesTagged += len(e.MessagesTagged)
	if s.Opt.DryRun {
		sum.MessagesInserted += len(e.WouldInsertMessages)
		sum.MessagesTagged += len(e.WouldTagMessages)
	}
}

func (s *Syncer) loadImportedConversations() ([]importedConversation, error) {
	const q = `SELECT c.id, c.reference_number, c.status_id, cs.name AS status_name,
		c.meta->>'` + MetaIssueID + `' AS issue_id,
		NULLIF(c.meta->>'` + MetaImportedAt + `', '')::timestamptz AS imported_at
		FROM conversations c
		JOIN conversation_statuses cs ON cs.id = c.status_id
		WHERE c.meta ? '` + MetaIssueID + `'
		ORDER BY c.id`
	var rows []importedConversation
	if err := s.DB.Select(&rows, q); err != nil {
		return nil, fmt.Errorf("imported conversations: %w", err)
	}
	return rows, nil
}

func (s *Syncer) syncConversation(ctx context.Context, c importedConversation, is Issue, msgs []Message, nameByID map[int]string, entry *SyncConversationReport) error {
	importedAt, importedOK := time.Time{}, false
	if c.ImportedAt.Valid {
		importedAt, importedOK = c.ImportedAt.Time.UTC(), true
	}
	nativeAfter, err := hasLibredeskNativeActivityAfter(s.DB, c.ID, s.Opt.CutoverEnd.UTC())
	if err != nil {
		return err
	}
	pylonStatus := MapStatus(is.State)
	statusPlan := PlanStatusSync(importedAt, s.Opt.CutoverEnd.UTC(), importedOK, nativeAfter, c.StatusName, pylonStatus)
	entry.StatusAction = statusPlan.Action
	entry.StatusSkippedReason = statusPlan.SkippedReason
	entry.Warning = statusPlan.Warning
	entry.StatusFrom = c.StatusName
	if statusPlan.ShouldApply {
		entry.StatusTo = pylonStatus
	}
	if s.Opt.DryRun {
		return s.planConversation(c, is, msgs, statusPlan, entry)
	}
	return s.applyConversation(ctx, c, is, msgs, statusPlan, entry)
}

func (s *Syncer) planConversation(c importedConversation, is Issue, msgs []Message, statusPlan StatusSyncPlan, entry *SyncConversationReport) error {
	insertPlans, tagMsgs, err := s.prepareSyncMessagePlans(c.ID, msgs)
	if err != nil {
		return err
	}
	for _, p := range insertPlans {
		entry.WouldInsertMessages = append(entry.WouldInsertMessages, syncActionFromPlan(p))
	}
	for _, m := range tagMsgs {
		entry.WouldTagMessages = append(entry.WouldTagMessages, SyncMessageAction{PylonMessageID: m.ID, Timestamp: m.Timestamp, Private: m.IsPrivate})
	}
	if statusPlan.Action == StatusActionUpdated {
		entry.WouldUpdateStatus = true
		entry.WouldUpdateStatusFrom = statusPlan.CurrentStatus
		entry.WouldUpdateStatusTo = statusPlan.PylonStatus
	}
	return nil
}

func (s *Syncer) prepareSyncMessagePlans(convID int, msgs []Message) ([]msgPlan, []Message, error) {
	s.Imp.convFiles = map[string]string{}
	s.Imp.convUploads = nil
	plans, err := s.Imp.prepareMessagePlans(s.DB, convID, msgs, false)
	if err != nil {
		return nil, nil, err
	}
	tagMsgs, err := s.Imp.planTagMetaUpdates(s.DB, convID, msgs)
	if err != nil {
		return nil, nil, err
	}
	return sortInsertPlans(plans), tagMsgs, nil
}

func (s *Syncer) applyConversation(ctx context.Context, c importedConversation, is Issue, msgs []Message, statusPlan StatusSyncPlan, entry *SyncConversationReport) error {
	s.Imp.pendingSourceIDs = map[string]int{}
	insertPlans, tagMsgs, err := s.prepareSyncMessagePlans(c.ID, msgs)
	if err != nil {
		return err
	}

	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	participants := map[int]bool{}
	for _, p := range insertPlans {
		agentID, err := s.Imp.insertMessagePlan(tx, c.ID, p)
		if err != nil {
			return fmt.Errorf("message %s: %w", p.m.ID, err)
		}
		entry.MessagesInserted = append(entry.MessagesInserted, syncActionFromPlan(p))
		if agentID > 0 {
			participants[agentID] = true
		}
	}
	for _, m := range tagMsgs {
		ok, err := s.Imp.tagPylonMessageMeta(tx, c.ID, m)
		if err != nil {
			return err
		}
		if ok {
			entry.MessagesTagged = append(entry.MessagesTagged, SyncMessageAction{PylonMessageID: m.ID, Timestamp: m.Timestamp, Private: m.IsPrivate})
		}
	}
	for id := range participants {
		if _, err := tx.Exec(`INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, c.ID, id); err != nil {
			return err
		}
	}
	if len(insertPlans) > 0 || len(entry.MessagesTagged) > 0 {
		if err := conversation.RefreshConversationSummaryFromMessages(tx, c.ID); err != nil {
			return err
		}
	}
	if statusPlan.ShouldApply && statusPlan.Action == StatusActionUpdated {
		if err := s.applyPylonStatus(tx, c.ID, is, statusPlan.PylonStatus); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Syncer) applyPylonStatus(tx *sqlx.Tx, convID int, is Issue, status string) error {
	statusID := s.Imp.statusIDs[status]
	if statusID == 0 {
		return fmt.Errorf("unknown status %q", status)
	}
	var resolvedAt *time.Time
	if status == StatusResolved {
		resolvedAt = is.ResolutionTime
		if resolvedAt == nil {
			resolvedAt = is.LatestMessageTime
		}
		if resolvedAt == nil {
			t := is.CreatedAt
			resolvedAt = &t
		}
	}
	_, err := tx.Exec(`UPDATE conversations SET
			status_id = $2,
			resolved_at = $3,
			last_resolved_at = $3,
			snoozed_until = NULL,
			updated_at = NOW()
		WHERE id = $1`, convID, statusID, resolvedAt)
	return err
}

func (s *Syncer) writeReport(rep SyncReport) error {
	if s.Opt.ReportPath == "" {
		return nil
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.Opt.ReportPath, b, 0o644); err != nil {
		return err
	}
	if s.Out != nil {
		fmt.Fprintf(s.Out, "pylon-sync: wrote report to %s\n", s.Opt.ReportPath)
	}
	return nil
}

func sortInsertPlans(plans []msgPlan) []msgPlan {
	var inserts []msgPlan
	for _, p := range plans {
		if p.kind == msgPlanInsert {
			inserts = append(inserts, p)
		}
	}
	sort.Slice(inserts, func(i, j int) bool {
		return inserts[i].m.Timestamp.Before(inserts[j].m.Timestamp)
	})
	return inserts
}

func syncActionFromPlan(p msgPlan) SyncMessageAction {
	return SyncMessageAction{PylonMessageID: p.m.ID, Timestamp: p.m.Timestamp, Private: p.m.IsPrivate}
}

func hasLibredeskNativeActivityAfter(db getter, convID int, after time.Time) (bool, error) {
	var ok bool
	err := db.Get(&ok, `SELECT EXISTS (
		SELECT 1 FROM conversation_messages
		WHERE conversation_id = $1
		  AND COALESCE(meta->>'`+MetaMessageID+`', '') = ''
		  AND created_at > $2
	)`, convID, after)
	return ok, err
}

// planTagMetaUpdates lists Pylon messages whose email body is already on the conversation but lacks pylon_message_id.
func (im *Importer) planTagMetaUpdates(db getter, convID int, msgs []Message) ([]Message, error) {
	var out []Message
	for _, m := range msgs {
		var exists bool
		if err := db.Get(&exists, `SELECT EXISTS (SELECT 1 FROM conversation_messages WHERE conversation_id = $1 AND meta->>'`+MetaMessageID+`' = $2)`, convID, m.ID); err != nil {
			return nil, err
		}
		if exists {
			continue
		}
		if m.EmailInfo == nil {
			continue
		}
		sourceID := stringutil.NormalizeMessageID(m.EmailInfo.MessageID)
		if sourceID == "" {
			continue
		}
		var msgID int
		err := db.Get(&msgID, `SELECT id FROM conversation_messages
			WHERE conversation_id = $1
			  AND trim(both '<>' from trim(source_id)) = $2
			  AND COALESCE(meta->>'`+MetaMessageID+`', '') = ''`,
			convID, sourceID)
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

// tagPylonMessageMeta attaches Pylon meta to an existing row matched by email Message-ID.
func (im *Importer) tagPylonMessageMeta(tx *sqlx.Tx, convID int, m Message) (bool, error) {
	if m.EmailInfo == nil {
		return false, nil
	}
	sourceID := stringutil.NormalizeMessageID(m.EmailInfo.MessageID)
	if sourceID == "" {
		return false, nil
	}
	meta := map[string]any{MetaMessageID: m.ID, "pylon_source": m.Source}
	if m.ThreadID != "" {
		meta["pylon_thread_id"] = m.ThreadID
	}
	e := m.EmailInfo
	if e.FromEmail != "" {
		meta["from"] = []string{NormalizeEmail(e.FromEmail)}
	}
	meta["to"] = lower(e.ToEmails)
	meta["cc"] = lower(e.CCEmails)
	meta["bcc"] = lower(e.BCCEmails)
	meta["email_message_id"] = sourceID
	metaJSON, _ := json.Marshal(meta)
	res, err := tx.Exec(`UPDATE conversation_messages SET
			meta = meta || $3::jsonb,
			updated_at = NOW()
		WHERE conversation_id = $1
		  AND trim(both '<>' from trim(source_id)) = $2
		  AND COALESCE(meta->>'`+MetaMessageID+`', '') = ''`,
		convID, sourceID, metaJSON)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// FilterIssuesForSync keeps only issues that exist after optional live verification.
func FilterIssuesForSync(ctx context.Context, fetcher *Fetcher, issues []Issue, verify bool) (present []Issue, missing []Issue, err error) {
	if !verify {
		return slices.Clone(issues), nil, nil
	}
	return fetcher.FilterPresentIssues(ctx, issues)
}
