-- name: get-links
SELECT id, created_at, conversation_id, issue_id, identifier, url, title, state, created_by
FROM conversation_linear_issues WHERE conversation_id = $1 ORDER BY created_at, id;

-- name: upsert-link
-- Linking an issue that is already linked refreshes its copy instead of adding a second row.
INSERT INTO conversation_linear_issues (conversation_id, issue_id, identifier, url, title, state, created_by)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (conversation_id, issue_id) DO UPDATE SET
	identifier = EXCLUDED.identifier, url = EXCLUDED.url, title = EXCLUDED.title, state = EXCLUDED.state, updated_at = NOW()
RETURNING id, created_at, conversation_id, issue_id, identifier, url, title, state, created_by;

-- name: delete-link
DELETE FROM conversation_linear_issues WHERE id = $1 AND conversation_id = $2;

-- name: get-links-by-issue
SELECT l.conversation_id, l.identifier, c.uuid AS conversation_uuid
FROM conversation_linear_issues l JOIN conversations c ON c.id = l.conversation_id
WHERE l.issue_id = $1 ORDER BY l.id;

-- name: update-issue
-- An empty value leaves the stored one: a webhook may leave a field out.
UPDATE conversation_linear_issues SET
	identifier = COALESCE(NULLIF($2, ''), identifier),
	url = COALESCE(NULLIF($3, ''), url),
	title = COALESCE(NULLIF($4, ''), title),
	state = COALESCE(NULLIF($5, ''), state),
	updated_at = NOW()
WHERE issue_id = $1;

-- name: claim-comment
-- Returns no row when this comment is already in this conversation, which makes webhook retries harmless.
INSERT INTO linear_comment_notes (comment_id, conversation_id) VALUES ($1, $2)
ON CONFLICT (comment_id, conversation_id) DO NOTHING
RETURNING id;

-- name: set-comment-message
UPDATE linear_comment_notes SET message_id = $2 WHERE id = $1;

-- name: release-comment
DELETE FROM linear_comment_notes WHERE id = $1;
