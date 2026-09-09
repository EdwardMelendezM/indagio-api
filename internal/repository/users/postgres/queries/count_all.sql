SELECT COUNT(*)
FROM users
WHERE deleted_at IS NULL
  AND (
        $1 = ''
        OR name  ILIKE '%' || $1 || '%'
        OR email ILIKE '%' || $1 || '%'
      );