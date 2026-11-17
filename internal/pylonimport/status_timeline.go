package pylonimport

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// insertPylonStatusActivities replays time_in_status_seconds as private status_change activities.
func (im *Importer) insertPylonStatusActivities(tx *sqlx.Tx, convID, agentID int, is Issue) error {
	if len(is.TimeInStatusSeconds) == 0 {
		return im.insertFinalStatusActivity(tx, convID, agentID, is.CreatedAt, MapStatus(is.State))
	}
	at := is.CreatedAt.UTC()
	seen := map[string]bool{}
	for _, key := range pylonStatusSegmentOrder {
		sec, ok := is.TimeInStatusSeconds[key]
		if !ok || sec <= 0 {
			continue
		}
		status := MapPylonStatusKey(key)
		if seen[status] {
			continue
		}
		seen[status] = true
		if err := im.insertStatusActivity(tx, convID, agentID, at, status); err != nil {
			return err
		}
		at = at.Add(time.Duration(sec) * time.Second)
	}
	if !seen[MapStatus(is.State)] {
		return im.insertStatusActivity(tx, convID, agentID, at, MapStatus(is.State))
	}
	return nil
}

func (im *Importer) insertFinalStatusActivity(tx *sqlx.Tx, convID, agentID int, at time.Time, status string) error {
	if status == "" {
		return nil
	}
	return im.insertStatusActivity(tx, convID, agentID, at, status)
}

func (im *Importer) insertStatusActivity(tx *sqlx.Tx, convID, agentID int, at time.Time, status string) error {
	if agentID == 0 && len(im.enabledAgents) > 0 {
		agentID = im.enabledAgents[0]
	}
	if agentID == 0 {
		return nil
	}
	meta, _ := json.Marshal(map[string]string{
		"activity_type": "status_change",
		"new_value":     status,
	})
	content := fmt.Sprintf("Pylon import marked the conversation as %s", status)
	if _, err := tx.Exec(`INSERT INTO conversation_messages
		(created_at, updated_at, type, status, private, conversation_id, content_type, content, text_content,
		 sender_id, sender_type, meta)
		VALUES ($1, $1, 'activity', 'sent', TRUE, $2, 'text', $3, $3, $4, 'agent', $5)`,
		at, convID, content, agentID, meta); err != nil {
		return fmt.Errorf("status activity %s: %w", status, err)
	}
	return nil
}

// applyPylonTimestamps sets conversation columns from Pylon after messages are written.
func (im *Importer) applyPylonTimestamps(tx *sqlx.Tx, convID int, is Issue, statusCategory string) error {
	var firstReply *time.Time
	if is.FirstResponseTime != nil {
		t := is.FirstResponseTime.UTC()
		firstReply = &t
	}
	var resolved *time.Time
	if statusCategory == "resolved" {
		resolved = is.ResolutionTime
		if resolved == nil {
			resolved = is.LatestMessageTime
		}
		if resolved == nil {
			t := is.CreatedAt
			resolved = &t
		}
	}
	_, err := tx.Exec(`UPDATE conversations SET
			first_reply_at = COALESCE($2, first_reply_at),
			last_reply_at = CASE WHEN $2 IS NOT NULL THEN GREATEST(COALESCE(last_reply_at, $2), $2) ELSE last_reply_at END,
			resolved_at = CASE WHEN $3 = 'resolved' THEN COALESCE($4, resolved_at) ELSE resolved_at END,
			last_resolved_at = CASE WHEN $3 = 'resolved' THEN COALESCE($4, last_resolved_at) ELSE last_resolved_at END
		WHERE id = $1`,
		convID, firstReply, statusCategory, resolved)
	return err
}