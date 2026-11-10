package pylonimport

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/abhinavxd/libredesk/internal/image"
	"github.com/abhinavxd/libredesk/internal/media"
	mmodels "github.com/abhinavxd/libredesk/internal/media/models"
	"github.com/jmoiron/sqlx"
)

// WipeOptions control a conversation wipe run.
type WipeOptions struct {
	// Confirm writes changes; otherwise only counts are printed.
	Confirm bool
}

// WipeReport counts what a wipe run did or would do.
type WipeReport struct {
	DryRun bool

	Conversations              int
	ConversationMessages       int
	MessageMedia               int
	WidgetDeliveriesCleared    int
	SlackThreads               int
	LinearIssueLinks           int
	ConversationTags           int
	UserNotifications          int
	NotificationEmailQueue     int
	ReferenceSequenceRestartedAt int
	MediaStoreDeletes          int
	MediaStoreDeleteErrors     int
}

// Wiper removes every conversation and related Libredesk rows so --pylon-import can run on a clean ticket store.
// It does not change users, organizations, tags, settings, or external integrations.
type Wiper struct {
	DB    *sqlx.DB
	Media *media.Manager
	Opt   WipeOptions
	Out   io.Writer
}

// Run plans or executes the wipe.
func (w *Wiper) Run(ctx context.Context) (*WipeReport, error) {
	if w.Out == nil {
		w.Out = io.Discard
	}
	rep := &WipeReport{DryRun: !w.Opt.Confirm}

	if err := w.count(rep); err != nil {
		return rep, err
	}
	if !w.Opt.Confirm {
		return rep, nil
	}

	files, err := w.listMessageMedia()
	if err != nil {
		return rep, err
	}
	for _, mm := range files {
		if ctx.Err() != nil {
			return rep, ctx.Err()
		}
		if w.Media == nil {
			return rep, fmt.Errorf("a media manager is required to delete message attachments")
		}
		if err := w.Media.Delete(mm.UUID); err != nil {
			rep.MediaStoreDeleteErrors++
			fmt.Fprintf(w.Out, "warning: delete media %s: %v\n", mm.UUID, err)
			continue
		}
		rep.MediaStoreDeletes++
		if strings.HasPrefix(mm.ContentType, "image/") {
			thumb := image.ThumbPrefix + mm.UUID
			if err := w.Media.Delete(thumb); err != nil {
				fmt.Fprintf(w.Out, "warning: delete thumb %s: %v\n", thumb, err)
			}
		}
	}

	tx, err := w.DB.BeginTxx(ctx, nil)
	if err != nil {
		return rep, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `UPDATE widget_campaign_deliveries SET conversation_uuid = NULL WHERE conversation_uuid IS NOT NULL`); err != nil {
		return rep, fmt.Errorf("widget campaign deliveries: %w", err)
	}

	res, err := tx.ExecContext(ctx, `DELETE FROM conversations`)
	if err != nil {
		return rep, fmt.Errorf("conversations: %w", err)
	}
	if n, _ := res.RowsAffected(); n >= 0 {
		// Already counted in dry run; keep store deletes as the live metric.
		_ = n
	}

	// Message media rows left without a store object (or skipped above) and draft uploads.
	if _, err := tx.ExecContext(ctx, `DELETE FROM media WHERE model_type = $1`, mmodels.ModelMessages); err != nil {
		return rep, fmt.Errorf("message media: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `SELECT setval('conversation_reference_number_sequence', 100, false)`); err != nil {
		return rep, fmt.Errorf("reference sequence: %w", err)
	}
	rep.ReferenceSequenceRestartedAt = 100

	if err := tx.Commit(); err != nil {
		return rep, err
	}
	return rep, nil
}

func (w *Wiper) count(rep *WipeReport) error {
	counts := []struct {
		dest *int
		q    string
	}{
		{&rep.Conversations, `SELECT COUNT(*) FROM conversations`},
		{&rep.ConversationMessages, `SELECT COUNT(*) FROM conversation_messages`},
		{&rep.MessageMedia, `SELECT COUNT(*) FROM media WHERE model_type = 'messages'`},
		{&rep.WidgetDeliveriesCleared, `SELECT COUNT(*) FROM widget_campaign_deliveries WHERE conversation_uuid IS NOT NULL`},
		{&rep.SlackThreads, `SELECT COUNT(*) FROM slack_threads`},
		{&rep.LinearIssueLinks, `SELECT COUNT(*) FROM conversation_linear_issues`},
		{&rep.ConversationTags, `SELECT COUNT(*) FROM conversation_tags`},
		{&rep.UserNotifications, `SELECT COUNT(*) FROM user_notifications WHERE conversation_id IS NOT NULL`},
		{&rep.NotificationEmailQueue, `SELECT COUNT(*) FROM notification_email_queue WHERE conversation_id IS NOT NULL`},
	}
	for _, c := range counts {
		if err := w.DB.Get(c.dest, c.q); err != nil {
			return err
		}
	}
	return nil
}

func (w *Wiper) listMessageMedia() ([]mmodels.Media, error) {
	var out []mmodels.Media
	err := w.DB.Select(&out, `SELECT id, uuid, content_type FROM media WHERE model_type = $1 ORDER BY id`, mmodels.ModelMessages)
	return out, err
}

// Print formats the report like the import summary.
func (r *WipeReport) Print(w io.Writer) {
	mode := "WIPE"
	if r.DryRun {
		mode = "DRY RUN (nothing deleted; pass --confirm with --pylon-wipe)"
	}
	fmt.Fprintf(w, "\n=== Pylon wipe: %s ===\n", mode)
	fmt.Fprintf(w, "Conversations to remove: %d\n", r.Conversations)
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
		fmt.Fprintf(w, "Reference sequence restarted at: %d\n", r.ReferenceSequenceRestartedAt)
	} else {
		fmt.Fprintf(w, "Reference sequence would restart at: 100\n")
	}
	fmt.Fprintf(w, "Unchanged: users, organizations, contact memberships, tag definitions, inboxes, settings, automations.\n")
}
