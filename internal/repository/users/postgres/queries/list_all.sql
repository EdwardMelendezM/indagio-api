SELECT id, name, email, role, avatar_url, avatar_version, verified, created_at, COALESCE(blocked, false)
FROM users
WHERE deleted_at IS NULL
  AND (
        $1 = ''
        OR name  ILIKE '%' || $1 || '%'
        OR email ILIKE '%' || $1 || '%'
      )
ORDER BY name, email ASC
LIMIT $2 OFFSET $3;