-- Find a single user by id. Hydrates the selected avatar border
-- (if any) via LEFT JOIN to avatar_borders so /me, /users/:id, and
-- the feed don't need a second round-trip.
--
-- The JOIN condition includes `ab.is_active = true` so that a
-- deactivated border never bubbles up: a user whose selected_border_id
-- points to a soft-deleted border will get a User with
-- SelectedBorder == nil and SelectedBorderID still set. Callers that
-- need to render "this user had a border that's no longer available"
-- can compare SelectedBorderID != nil && SelectedBorder == nil.
SELECT
    u.id, u.name, u.email, u.role,
    u.avatar_url, u.avatar_version,
    u.verified, u.created_at, COALESCE(u.blocked, false),
    u.selected_border_id,
    ab.id, ab.slug, ab.name, ab.asset_url, ab.thumbnail_url, ab.tier
FROM users u
LEFT JOIN avatar_borders ab
       ON ab.id = u.selected_border_id
      AND ab.is_active = true
WHERE u.id = $1
