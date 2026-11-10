package conversation

import "github.com/jmoiron/sqlx"

// RefreshConversationSummaryFromMessages recomputes conversation preview and timestamp columns from message rows.
// Use after backdated inserts so last_message_at reflects the newest message by created_at, not the last POST.
func RefreshConversationSummaryFromMessages(tx *sqlx.Tx, convID int) error {
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
