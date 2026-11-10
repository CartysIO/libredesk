package pylonimport

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/abhinavxd/libredesk/internal/stringutil"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// conversations.meta: sibling conversation absorbed into the merge target of a Pylon issue.
const MetaAbsorbedInto = "pylon_absorbed_into_conversation_id"

// issueEmailMessageIDs returns normalized Message-IDs from a Pylon issue's messages.
func issueEmailMessageIDs(msgs []Message) map[string]bool {
	out := map[string]bool{}
	for _, m := range msgs {
		if m.EmailInfo == nil {
			continue
		}
		if id := stringutil.NormalizeMessageID(m.EmailInfo.MessageID); id != "" {
			out[id] = true
		}
	}
	return out
}

// siblingConversationIDs lists existing Libredesk conversations (other than primaryID) that hold an email
// from this issue, so live inbox fragmentation can be collapsed into one thread.
func (im *Importer) siblingConversationIDs(msgs []Message, primaryID int) []int {
	ids := map[int]bool{}
	for id := range issueEmailMessageIDs(msgs) {
		if convID := im.usedSourceIDs[id]; convID > 0 && convID != primaryID {
			ids[convID] = true
		}
	}
	out := make([]int, 0, len(ids))
	for id := range ids {
		out = append(out, id)
	}
	sort.Ints(out)
	return out
}

func (im *Importer) planSiblingAbsorption(msgs []Message, primaryID int, primaryRef string) {
	for _, sibID := range im.siblingConversationIDs(msgs, primaryID) {
		ref := strconv.Itoa(sibID)
		if r, err := im.conversationReference(sibID); err == nil && r != "" {
			ref = r
		}
		im.rep.Absorbed = append(im.rep.Absorbed, fmt.Sprintf("Libredesk #%s -> merge target #%s (Pylon email fragment)", ref, primaryRef))
	}
	sort.Strings(im.rep.Absorbed)
}

func (im *Importer) conversationReference(id int) (string, error) {
	var ref string
	err := im.DB.Get(&ref, `SELECT reference_number FROM conversations WHERE id = $1`, id)
	return ref, err
}

// lookupExistingConversation loads one conversation for a reconcile re-run.
func (im *Importer) lookupExistingConversation(id int) (*ExistingConversation, error) {
	var row struct {
		ID        int            `db:"id"`
		Ref       string         `db:"reference_number"`
		Subject   sql.NullString `db:"subject"`
		Email     sql.NullString `db:"email"`
		CreatedAt time.Time      `db:"created_at"`
		Merged    bool           `db:"merged"`
	}
	if err := im.DB.Get(&row, `SELECT c.id, c.reference_number, c.subject, u.email, c.created_at, c.meta ? '`+MetaMergedIssueID+`' AS merged
		FROM conversations c JOIN users u ON u.id = c.contact_id WHERE c.id = $1`, id); err != nil {
		return nil, err
	}
	return &ExistingConversation{
		ID: row.ID, ReferenceNumber: row.Ref, Subject: row.Subject.String, ContactEmail: row.Email.String,
		CreatedAt: row.CreatedAt, Merged: row.Merged,
	}, nil
}

// absorbSiblingConversations removes live email copies that Pylon will supply, moves any other messages to the
// target, and resolves empty sibling conversations so one Pylon issue maps to one Libredesk conversation.
func (im *Importer) absorbSiblingConversations(tx *sqlx.Tx, targetID int, msgs []Message) ([]string, error) {
	issueIDs := issueEmailMessageIDs(msgs)
	var storeDeletes []string
	for _, sibID := range im.siblingConversationIDs(msgs, targetID) {
		var imported bool
		if err := tx.Get(&imported, `SELECT (meta ? '`+MetaIssueID+`') FROM conversations WHERE id = $1`, sibID); err != nil {
			return nil, err
		}
		if imported {
			continue
		}
		ref, _ := im.conversationReference(sibID)
		targetRef, _ := im.conversationReference(targetID)
		if ref == "" {
			ref = strconv.Itoa(sibID)
		}
		if targetRef == "" {
			targetRef = strconv.Itoa(targetID)
		}

		var msgRows []struct {
			ID       int            `db:"id"`
			Type     string         `db:"type"`
			SourceID sql.NullString `db:"source_id"`
		}
		if err := tx.Select(&msgRows, `SELECT id, type, source_id FROM conversation_messages WHERE conversation_id = $1 ORDER BY id`, sibID); err != nil {
			return nil, fmt.Errorf("sibling #%s messages: %w", ref, err)
		}
		var deleteIDs, moveIDs []int
		for _, row := range msgRows {
			if row.Type == "activity" {
				deleteIDs = append(deleteIDs, row.ID)
				continue
			}
			sid := stringutil.NormalizeMessageID(row.SourceID.String)
			if sid != "" && issueIDs[sid] {
				deleteIDs = append(deleteIDs, row.ID)
				delete(im.usedSourceIDs, sid)
				continue
			}
			moveIDs = append(moveIDs, row.ID)
		}
		if len(deleteIDs) > 0 {
			uuids, err := im.deleteMessages(tx, deleteIDs)
			if err != nil {
				return nil, fmt.Errorf("sibling #%s delete: %w", ref, err)
			}
			storeDeletes = append(storeDeletes, uuids...)
		}
		for _, msgID := range moveIDs {
			if _, err := tx.Exec(`UPDATE conversation_messages SET conversation_id = $2, updated_at = NOW() WHERE id = $1`, msgID, targetID); err != nil {
				return nil, err
			}
		}
		var remaining int
		if err := tx.Get(&remaining, `SELECT COUNT(*) FROM conversation_messages WHERE conversation_id = $1`, sibID); err != nil {
			return nil, err
		}
		if remaining > 0 {
			continue
		}
		meta, _ := json.Marshal(map[string]any{MetaAbsorbedInto: targetID})
		resolved := im.statusIDs[StatusResolved]
		if _, err := tx.Exec(`UPDATE conversations SET
				status_id = $2,
				meta = meta || $3::jsonb,
				resolved_at = COALESCE(resolved_at, NOW()),
				last_resolved_at = COALESCE(last_resolved_at, NOW()),
				updated_at = NOW()
			WHERE id = $1`, sibID, resolved, meta); err != nil {
			return nil, fmt.Errorf("sibling #%s close: %w", ref, err)
		}
		im.rep.Absorbed = append(im.rep.Absorbed, fmt.Sprintf("Libredesk #%s absorbed into #%s", ref, targetRef))
	}
	sort.Strings(im.rep.Absorbed)
	return storeDeletes, nil
}

func (im *Importer) deleteMessages(tx *sqlx.Tx, msgIDs []int) ([]string, error) {
	if len(msgIDs) == 0 {
		return nil, nil
	}
	var uuids []string
	if err := tx.Select(&uuids, `SELECT uuid FROM media WHERE model_type = 'messages' AND model_id = ANY($1)`, pq.Array(msgIDs)); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM media WHERE model_type = 'messages' AND model_id = ANY($1)`, pq.Array(msgIDs)); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM conversation_messages WHERE id = ANY($1)`, pq.Array(msgIDs)); err != nil {
		return nil, err
	}
	return uuids, nil
}

// reconcileLiveMessage backdates a live email that was already in the merge target and attaches Pylon meta.
func (im *Importer) reconcileLiveMessage(tx *sqlx.Tx, convID int, m Message) error {
	if m.EmailInfo == nil {
		return nil
	}
	sourceID := stringutil.NormalizeMessageID(m.EmailInfo.MessageID)
	if sourceID == "" {
		return nil
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
			created_at = $3, updated_at = $3,
			meta = meta || $4::jsonb
		WHERE conversation_id = $1 AND source_id = $2`,
		convID, sourceID, m.Timestamp, metaJSON)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		im.rep.ReconciledMessages++
	}
	return nil
}
