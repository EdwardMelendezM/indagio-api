-- Wipes the avatar entirely. Bumps avatar_version so any cached
-- URL becomes stale on the next read.
UPDATE users
SET avatar_url = NULL,
    avatar_version = avatar_version + 1
WHERE id = $1
  AND deleted_at IS NULL
RETURNING avatar_version
