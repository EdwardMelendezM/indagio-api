SELECT id, name, email, role, avatar_url, avatar_version, verified, created_at, COALESCE(blocked, false)
FROM users
WHERE deleted_at IS NULL
  AND verified = TRUE
  AND blocked = FALSE
  AND id <> $1
  AND (
        $2 = ''
        OR name  ILIKE '%' || $2 || '%'
        OR email ILIKE '%' || $2 || '%'
      )
ORDER BY name ASC
LIMIT $3 OFFSET $4;
