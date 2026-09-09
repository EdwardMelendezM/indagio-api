SELECT COUNT(*)
FROM users
WHERE deleted_at IS NULL
  AND verified = TRUE
  AND blocked = FALSE
  AND id <> $1
  AND (
        $2 = ''
        OR name  ILIKE '%' || $2 || '%'
        OR email ILIKE '%' || $2 || '%'
      );
