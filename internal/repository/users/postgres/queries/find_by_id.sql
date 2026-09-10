-- Find a single user by id.
--
-- NOTE: avatar_borders is not part of the current schema set, so this
-- query must not reference that table. We still return a fixed shape
-- for the repository scanner by projecting NULL placeholders for the
-- border payload columns.
SELECT
    u.id, u.name, u.email, u.role,
    u.avatar_url, u.avatar_version,
    u.verified, u.created_at, COALESCE(u.blocked, false),
    u.selected_border_id,
    NULL::uuid AS border_id,
    NULL::text AS border_slug,
    NULL::text AS border_name,
    NULL::text AS border_asset_url,
    NULL::text AS border_thumbnail_url,
    NULL::text AS border_tier
FROM users u
WHERE u.id = $1
