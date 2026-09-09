-- Persists the avatar JSON document (or NULL to clear) and bumps
-- avatar_version atomically so any cached URL becomes stale.
UPDATE users
SET avatar_url = $2,
    avatar_version = avatar_version + 1
WHERE id = $1
  AND deleted_at IS NULL
RETURNING avatar_version
