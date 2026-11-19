package pylonimport

import (
	"context"
	"fmt"
	"io"

	"github.com/abhinavxd/libredesk/internal/media"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

const nonImportedSQL = `
NOT COALESCE(c.meta ? '` + MetaIssueID + `', FALSE)
AND NOT COALESCE(c.meta ? '` + MetaMergedIssueID + `', FALSE)`

// NonImportedConversation is a Libredesk conversation with no Pylon import meta.
type NonImportedConversation struct {
	ID              int64  `db:"id"`
	ReferenceNumber string `db:"reference_number"`
	Subject         string `db:"subject"`
}

// PurgeNonImportedOptions control removal of conversations that were not created or merged from a Pylon issue.
type PurgeNonImportedOptions struct {
	// Confirm deletes rows; otherwise only counts are printed.
	Confirm bool
}

// PurgeNonImportedReport counts what a purge run did or would do.
type PurgeNonImportedReport struct {
	DryRun bool

	Targets []NonImportedConversation

	Conversations           int
	ConversationMessages    int
	MessageMedia            int
	WidgetDeliveriesCleared int
	SlackThreads            int
	LinearIssueLinks        int
	ConversationTags        int
	UserNotifications       int
	NotificationEmailQueue  int
	MediaStoreDeletes       int
	MediaStoreDeleteErrors  int
}

// NonImportedPurger removes conversations with no pylon_issue_id or pylon_merged_issue_id in meta.
// Import leaves every Pylon issue on a tagged conversation; anything else is pre-cutover Libredesk data.
type NonImportedPurger struct {
	DB    *sqlx.DB
	Media *media.Manager
	Opt   PurgeNonImportedOptions
	Out   io.Writer
}

// Run plans or executes the purge.
func (p *NonImportedPurger) Run(ctx context.Context) (*PurgeNonImportedReport, error) {
	if p.Out == nil {
		p.Out = io.Discard
	}
	rep := &PurgeNonImportedReport{DryRun: !p.Opt.Confirm}

	targets, err := p.listNonImportedConversations()
	if err != nil {
		return rep, err
	}
	rep.Targets = targets
	if len(targets) == 0 {
		return rep, nil
	}
	ids := conversationIDs(targets)

	if err := p.countForIDs(rep, ids); err != nil {
		return rep, err
	}
	if !p.Opt.Confirm {
		return rep, nil
	}

	files, err := p.listMessageMedia(ids)
	if err != nil {
		return rep, err
	}
	rep.MediaStoreDeletes, rep.MediaStoreDeleteErrors, err = deleteStoredMessageMedia(ctx, p.Media, p.Out, files)
	if err != nil {
		return rep, err
	}
	if rep.MediaStoreDeleteErrors > 0 {
		return rep, fmt.Errorf("message media: %d file(s) could not be deleted from the store (database rows not changed)", rep.MediaStoreDeleteErrors)
	}

	tx, err := p.DB.BeginTxx(ctx, nil)
	if err != nil {
		return rep, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE widget_campaign_deliveries d SET conversation_uuid = NULL
		FROM conversations c
		WHERE d.conversation_uuid = c.uuid AND c.id = ANY($1)`, pq.Array(ids)); err != nil {
		return rep, fmt.Errorf("widget campaign deliveries: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		DELETE FROM media m
		WHERE m.model_type = $1
		  AND m.model_id IN (SELECT id FROM conversation_messages WHERE conversation_id = ANY($2))`,
		mmodels.ModelMessages, pq.Array(ids)); err != nil {
		return rep, fmt.Errorf("message media: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM conversations WHERE id = ANY($1)`, pq.Array(ids)); err != nil {
		return rep, fmt.Errorf("conversations: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return rep, err
	}
	return rep, nil
}

func (p *NonImportedPurger) listNonImportedConversations() ([]NonImportedConversation, error) {
	var out []NonImportedConversation
	q := `SELECT c.id, c.reference_number, c.subject FROM conversations c WHERE ` + nonImportedSQL + ` ORDER BY c.id`
	if err := p.DB.Select(&out, q); err != nil {
		return nil, err
	}
	return out, nil
}

func conversationIDs(targets []NonImportedConversation) []int64 {
	ids := make([]int64, len(targets))
	for i, t := range targets {
		ids[i] = t.ID
	}
	return ids
}

func (p *NonImportedPurger) countForIDs(rep *PurgeNonImportedReport, ids []int64) error {
	rep.Conversations = len(ids)
	counts := []struct {
		dest *int
		q    string
	}{
		{&rep.ConversationMessages, `SELECT COUNT(*) FROM conversation_messages WHERE conversation_id = ANY($1)`},
		{&rep.WidgetDeliveriesCleared, `SELECT COUNT(*) FROM widget_campaign_deliveries d
			JOIN conversations c ON d.conversation_uuid = c.uuid WHERE c.id = ANY($1)`},
		{&rep.SlackThreads, `SELECT COUNT(*) FROM slack_threads WHERE conversation_id = ANY($1)`},
		{&rep.LinearIssueLinks, `SELECT COUNT(*) FROM conversation_linear_issues WHERE conversation_id = ANY($1)`},
		{&rep.ConversationTags, `SELECT COUNT(*) FROM conversation_tags WHERE conversation_id = ANY($1)`},
		{&rep.UserNotifications, `SELECT COUNT(*) FROM user_notifications WHERE conversation_id = ANY($1)`},
		{&rep.NotificationEmailQueue, `SELECT COUNT(*) FROM notification_email_queue WHERE conversation_id = ANY($1)`},
	}
	for _, c := range counts {
		if err := p.DB.Get(c.dest, c.q, pq.Array(ids)); err != nil {
			return err
		}
	}
	if err := p.DB.Get(&rep.MessageMedia, `SELECT COUNT(*) FROM media m
		WHERE m.model_type = $1
		  AND m.model_id IN (SELECT id FROM conversation_messages WHERE conversation_id = ANY($2))`,
		mmodels.ModelMessages, pq.Array(ids)); err != nil {
		return err
	}
	return nil
}

func (p *NonImportedPurger) listMessageMedia(ids []int64) ([]mmodels.Media, error) {
	var out []mmodels.Media
	err := p.DB.Select(&out, `SELECT m.id, m.uuid, m.content_type FROM media m
		WHERE m.model_type = $1
		  AND m.model_id IN (SELECT id FROM conversation_messages WHERE conversation_id = ANY($2))
		ORDER BY m.id`, mmodels.ModelMessages, pq.Array(ids))
	return out, err
}

// Print formats the report like the import summary.
func (r *PurgeNonImportedReport) Print(w io.Writer) {
	mode := "PURGE NON-IMPORTED"
	if r.DryRun {
		mode = "DRY RUN (non-imported conversations counted only; no rows deleted)"
	}
	fmt.Fprintf(w, "\n=== Pylon import: %s ===\n", mode)
	fmt.Fprintf(w, "Criterion: conversations.meta lacks %q and %q.\n", MetaIssueID, MetaMergedIssueID)
	fmt.Fprintf(w, "Conversations without Pylon meta: %d\n", r.Conversations)
	if r.Conversations == 0 {
		fmt.Fprintf(w, "Nothing to remove.\n")
		return
	}
	fmt.Fprintf(w, "Conversations to remove:\n")
	for _, t := range r.Targets {
		fmt.Fprintf(w, "  #%s id=%d %s\n", t.ReferenceNumber, t.ID, t.Subject)
	}
	fmt.Fprintf(w, "Messages (cascade): %d\n", r.ConversationMessages)
	fmt.Fprintf(w, "Message media rows: %d\n", r.MessageMedia)
	fmt.Fprintf(w, "Widget deliveries to clear conversation_uuid: %d\n", r.WidgetDeliveriesCleared)
	fmt.Fprintf(w, "Slack thread rows (cascade): %d\n", r.SlackThreads)
	fmt.Fprintf(w, "Linear issue links (cascade): %d\n", r.LinearIssueLinks)
	fmt.Fprintf(w, "Conversation tag links (cascade): %d\n", r.ConversationTags)
	fmt.Fprintf(w, "User notifications (cascade): %d\n", r.UserNotifications)
	fmt.Fprintf(w, "Notification email queue (cascade): %d\n", r.NotificationEmailQueue)
	if !r.DryRun {
		fmt.Fprintf(w, "Media files removed from store: %d (%d errors)\n", r.MediaStoreDeletes, r.MediaStoreDeleteErrors)
	}
	fmt.Fprintf(w, "Unchanged: Pylon-imported conversations, users, organizations, reference sequence.\n")
}
