-- Find a single user by id.
SELECT
    u.id, u.name, u.email, u.role,
    u.avatar_url, u.avatar_version,
    u.verified, u.created_at, COALESCE(u.blocked, false)
FROM users u
WHERE u.id = $1
