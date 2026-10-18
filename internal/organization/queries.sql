-- name: count-organizations
-- $1 = search text on name or external_id ('' for all).
SELECT COUNT(*) FROM organizations o
WHERE ($1 = '' OR o.name ILIKE '%' || $1 || '%' OR o.external_id = $1);

-- name: get-organizations
-- $1 = search text on name or external_id ('' for all), $2 = limit, $3 = offset.
SELECT o.id, o.created_at, o.updated_at, o.name, o.external_id,
    (SELECT COUNT(*) FROM users u WHERE u.organization_id = o.id AND u.deleted_at IS NULL) AS contact_count
FROM organizations o
WHERE ($1 = '' OR o.name ILIKE '%' || $1 || '%' OR o.external_id = $1)
ORDER BY o.name, o.id
LIMIT $2 OFFSET $3;

-- name: get-organization
SELECT o.id, o.created_at, o.updated_at, o.name, o.external_id,
    (SELECT COUNT(*) FROM users u WHERE u.organization_id = o.id AND u.deleted_at IS NULL) AS contact_count
FROM organizations o WHERE o.id = $1;

-- name: get-organization-by-external-id
SELECT o.id, o.created_at, o.updated_at, o.name, o.external_id,
    (SELECT COUNT(*) FROM users u WHERE u.organization_id = o.id AND u.deleted_at IS NULL) AS contact_count
FROM organizations o WHERE o.external_id = $1;

-- name: insert-organization
INSERT INTO organizations (name, external_id) VALUES ($1, $2)
RETURNING id, created_at, updated_at, name, external_id;

-- name: update-organization
UPDATE organizations SET name = $2, external_id = $3, updated_at = NOW() WHERE id = $1
RETURNING id, created_at, updated_at, name, external_id;

-- name: upsert-organization-by-external-id
-- Creates the organization or renames it. An empty name keeps the current name.
INSERT INTO organizations (name, external_id) VALUES (COALESCE(NULLIF($2, ''), $1), $1)
ON CONFLICT (external_id) WHERE external_id IS NOT NULL DO UPDATE
SET name = COALESCE(NULLIF($2, ''), organizations.name),
    updated_at = CASE WHEN NULLIF($2, '') IS NOT NULL AND $2 IS DISTINCT FROM organizations.name THEN NOW() ELSE organizations.updated_at END
RETURNING id, created_at, updated_at, name, external_id;

-- name: delete-organization
DELETE FROM organizations WHERE id = $1;

-- name: get-organization-contacts
SELECT id, created_at, first_name, COALESCE(last_name, '') AS last_name, email, avatar_url, external_user_id
FROM users
WHERE organization_id = $1 AND type IN ('contact', 'visitor') AND deleted_at IS NULL
ORDER BY first_name, id
LIMIT $2 OFFSET $3;

-- name: set-contact-organization
-- Sets the contact's organization. Conversations of the contact that have no organization get it too.
WITH contact AS (
    UPDATE users SET organization_id = $2, updated_at = NOW()
    WHERE id = $1 AND type IN ('contact', 'visitor') AND deleted_at IS NULL
    RETURNING id
), convs AS (
    UPDATE conversations SET organization_id = $2
    WHERE $2::BIGINT IS NOT NULL AND organization_id IS NULL AND contact_id IN (SELECT id FROM contact)
)
SELECT COUNT(*) FROM contact;

-- name: set-contact-organization-if-unset
-- Sets the contact's organization only when it has none. Returns 0 rows changed when it already has one.
WITH contact AS (
    UPDATE users SET organization_id = $2, updated_at = NOW()
    WHERE id = $1 AND type IN ('contact', 'visitor') AND deleted_at IS NULL AND organization_id IS NULL
    RETURNING id
), convs AS (
    UPDATE conversations SET organization_id = $2
    WHERE organization_id IS NULL AND contact_id IN (SELECT id FROM contact)
)
SELECT COUNT(*) FROM contact;
