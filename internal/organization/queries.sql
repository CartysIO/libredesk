-- name: count-organizations
-- $1 = search text on name or external_id ('' for all).
SELECT COUNT(*) FROM organizations o
WHERE ($1 = '' OR o.name ILIKE '%' || $1 || '%' OR o.external_id = $1);

-- name: get-organizations
-- $1 = search text on name or external_id ('' for all), $2 = limit, $3 = offset.
SELECT o.id, o.created_at, o.updated_at, o.name, o.external_id, o.custom_attributes, o.domains,
    (SELECT COUNT(*) FROM contact_organizations co JOIN users u ON u.id = co.contact_id WHERE co.organization_id = o.id AND u.deleted_at IS NULL) AS contact_count
FROM organizations o
WHERE ($1 = '' OR o.name ILIKE '%' || $1 || '%' OR o.external_id = $1)
ORDER BY o.name, o.id
LIMIT $2 OFFSET $3;

-- name: get-organization
SELECT o.id, o.created_at, o.updated_at, o.name, o.external_id, o.custom_attributes, o.domains,
    (SELECT COUNT(*) FROM contact_organizations co JOIN users u ON u.id = co.contact_id WHERE co.organization_id = o.id AND u.deleted_at IS NULL) AS contact_count
FROM organizations o WHERE o.id = $1;

-- name: get-organization-by-external-id
SELECT o.id, o.created_at, o.updated_at, o.name, o.external_id, o.custom_attributes, o.domains,
    (SELECT COUNT(*) FROM contact_organizations co JOIN users u ON u.id = co.contact_id WHERE co.organization_id = o.id AND u.deleted_at IS NULL) AS contact_count
FROM organizations o WHERE o.external_id = $1;

-- name: insert-organization
INSERT INTO organizations (name, external_id) VALUES ($1, $2)
RETURNING id, created_at, updated_at, name, external_id, custom_attributes, domains;

-- name: update-organization
UPDATE organizations SET name = $2, external_id = $3, updated_at = NOW() WHERE id = $1
RETURNING id, created_at, updated_at, name, external_id, custom_attributes, domains;

-- name: upsert-organization-by-external-id
-- Creates the organization or renames it. An empty name keeps the current name.
INSERT INTO organizations (name, external_id) VALUES (COALESCE(NULLIF($2, ''), $1), $1)
ON CONFLICT (external_id) WHERE external_id IS NOT NULL DO UPDATE
SET name = COALESCE(NULLIF($2, ''), organizations.name),
    updated_at = CASE WHEN NULLIF($2, '') IS NOT NULL AND $2 IS DISTINCT FROM organizations.name THEN NOW() ELSE organizations.updated_at END
RETURNING id, created_at, updated_at, name, external_id, custom_attributes, domains;

-- name: set-organization-fields
-- $2 = domains, $3 = custom attributes; NULL keeps the current value.
-- $4 = merge: add the domains and merge the attributes into the current ones, instead of replacing them.
UPDATE organizations SET
    domains = CASE
        WHEN $2::TEXT[] IS NULL THEN domains
        WHEN $4::BOOLEAN THEN ARRAY(SELECT DISTINCT d FROM unnest(domains || $2::TEXT[]) d ORDER BY d)
        ELSE $2::TEXT[] END,
    custom_attributes = CASE
        WHEN $3::JSONB IS NULL THEN custom_attributes
        WHEN $4::BOOLEAN THEN custom_attributes || $3::JSONB
        ELSE $3::JSONB END,
    updated_at = NOW()
WHERE id = $1
RETURNING id;

-- name: assign-organization-by-domain
-- Adds a contact that has no organization to the one organization that has domain $2, and makes it the
-- contact's default. Nothing happens when no organization or more than one has the domain.
-- Conversations of the contact that have no organization get it too.
WITH org AS (
    SELECT id FROM organizations WHERE domains @> ARRAY[$2::TEXT] LIMIT 2
), pick AS (
    SELECT id FROM org WHERE (SELECT COUNT(*) FROM org) = 1
), contact AS (
    SELECT u.id FROM users u
    WHERE u.id = $1 AND u.type IN ('contact', 'visitor') AND u.deleted_at IS NULL AND u.organization_id IS NULL
      AND NOT EXISTS (SELECT 1 FROM contact_organizations co WHERE co.contact_id = u.id)
), member AS (
    INSERT INTO contact_organizations (contact_id, organization_id)
    SELECT contact.id, pick.id FROM contact, pick
    ON CONFLICT DO NOTHING
    RETURNING contact_id, organization_id
), def AS (
    UPDATE users SET organization_id = member.organization_id, updated_at = NOW()
    FROM member WHERE users.id = member.contact_id
    RETURNING users.id
), convs AS (
    UPDATE conversations SET organization_id = member.organization_id
    FROM member WHERE conversations.contact_id = member.contact_id AND conversations.organization_id IS NULL
    RETURNING conversations.uuid
)
SELECT COALESCE((SELECT organization_id FROM member), 0) AS organization_id,
       COALESCE((SELECT array_agg(uuid::TEXT) FROM convs), '{}') AS filed;

-- name: delete-organization
DELETE FROM organizations WHERE id = $1;

-- name: get-organization-contacts
-- is_default is true when this organization is the contact's default organization.
SELECT u.id, u.created_at, u.first_name, COALESCE(u.last_name, '') AS last_name, u.email, u.avatar_url, u.external_user_id,
    COALESCE(u.organization_id = co.organization_id, FALSE) AS is_default
FROM contact_organizations co
JOIN users u ON u.id = co.contact_id
WHERE co.organization_id = $1 AND u.type IN ('contact', 'visitor') AND u.deleted_at IS NULL
ORDER BY u.first_name, u.id
LIMIT $2 OFFSET $3;

-- name: get-contact-organizations
-- The organizations of a contact, the default one first.
SELECT o.id, o.name, o.external_id, COALESCE(u.organization_id = o.id, FALSE) AS is_default
FROM contact_organizations co
JOIN organizations o ON o.id = co.organization_id
JOIN users u ON u.id = co.contact_id
WHERE co.contact_id = $1
ORDER BY is_default DESC, o.name, o.id;

-- name: add-contact-organization
-- Adds the contact to the organization. $3 = make it the default organization.
-- The organization becomes the default also when the contact has none. Conversations of the contact
-- that have no organization get the contact's default organization.
-- Returns whether the membership is new, whether the default changed, and the UUIDs of the conversations it filed.
WITH contact AS (
    SELECT id, organization_id FROM users
    WHERE id = $1 AND type IN ('contact', 'visitor') AND deleted_at IS NULL
), member AS (
    INSERT INTO contact_organizations (contact_id, organization_id)
    SELECT id, $2 FROM contact
    ON CONFLICT DO NOTHING
    RETURNING contact_id
), def AS (
    UPDATE users SET organization_id = $2, updated_at = NOW()
    WHERE id IN (SELECT id FROM contact WHERE organization_id IS NULL OR ($3::BOOLEAN AND organization_id <> $2))
    RETURNING id
), convs AS (
    UPDATE conversations SET organization_id = COALESCE((SELECT $2::BIGINT FROM def), (SELECT organization_id FROM contact))
    WHERE organization_id IS NULL AND contact_id IN (SELECT id FROM contact)
    RETURNING uuid
)
SELECT (SELECT COUNT(*) FROM contact) AS found,
       (SELECT COUNT(*) FROM member) AS added,
       (SELECT COUNT(*) FROM def) AS default_changed,
       COALESCE((SELECT array_agg(uuid::TEXT) FROM convs), '{}') AS filed;

-- name: remove-contact-organization
-- Removes the contact from the organization. When it was the default, the earliest other membership becomes
-- the default, or none. Conversations keep their organization.
WITH member AS (
    DELETE FROM contact_organizations WHERE contact_id = $1 AND organization_id = $2
    RETURNING contact_id
), def AS (
    UPDATE users SET organization_id = (
        SELECT co.organization_id FROM contact_organizations co
        WHERE co.contact_id = $1 AND co.organization_id <> $2
        ORDER BY co.created_at, co.organization_id LIMIT 1
    ), updated_at = NOW()
    WHERE id = $1 AND organization_id = $2
    RETURNING organization_id
)
SELECT (SELECT COUNT(*) FROM member) AS removed,
       (SELECT COUNT(*) FROM def) AS default_changed;

-- name: reassign-default-organization
-- Before an organization is deleted: contacts that have it as default get their earliest other membership.
UPDATE users u SET organization_id = (
    SELECT co.organization_id FROM contact_organizations co
    WHERE co.contact_id = u.id AND co.organization_id <> $1
    ORDER BY co.created_at, co.organization_id LIMIT 1
), updated_at = NOW()
WHERE u.organization_id = $1
RETURNING u.id, u.organization_id;

-- name: set-conversation-organization
-- Files the conversation under one of its contact's organizations, or under none when $2 is NULL.
-- Returns no row when the organization is not one of the contact's.
UPDATE conversations c SET organization_id = $2, updated_at = NOW()
WHERE c.uuid = $1
  AND ($2::BIGINT IS NULL OR EXISTS (
      SELECT 1 FROM contact_organizations co WHERE co.contact_id = c.contact_id AND co.organization_id = $2))
RETURNING c.id;
