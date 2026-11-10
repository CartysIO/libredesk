-- name: get-thread-by-conversation
SELECT conversation_id, channel_id, thread_ts FROM slack_threads WHERE conversation_id = $1;

-- name: get-thread-by-ts
-- $1 = channel ID, $2 = thread root ts. Returns the conversation UUID too.
SELECT st.conversation_id, st.channel_id, st.thread_ts, c.uuid AS conversation_uuid
FROM slack_threads st JOIN conversations c ON c.id = st.conversation_id
WHERE st.channel_id = $1 AND st.thread_ts = $2;

-- name: insert-thread
INSERT INTO slack_threads (conversation_id, channel_id, thread_ts) VALUES ($1, $2, $3)
ON CONFLICT (conversation_id) DO NOTHING;

-- name: claim-slack-message
-- Records a Slack message before the bridge acts on it. Returns no row when it was already recorded,
-- which makes Slack's retries and the double delivery of a mention (message + app_mention) harmless.
INSERT INTO slack_messages (channel_id, slack_ts, direction) VALUES ($1, $2, $3)
ON CONFLICT (channel_id, slack_ts) DO NOTHING
RETURNING id;

-- name: set-slack-message-message-id
UPDATE slack_messages SET message_id = $2 WHERE id = $1;

-- name: insert-outgoing-slack-message
INSERT INTO slack_messages (channel_id, slack_ts, direction, message_id) VALUES ($1, $2, 'out', $3)
ON CONFLICT DO NOTHING;

-- name: message-is-bridged
-- True when the Libredesk message is already in Slack, or came from Slack.
SELECT EXISTS (SELECT 1 FROM slack_messages WHERE message_id = $1);
