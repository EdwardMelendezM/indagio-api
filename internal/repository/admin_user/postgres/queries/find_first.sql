SELECT id, email, created_at
FROM   user_adm
WHERE  deleted_at IS NULL
ORDER BY created_at DESC
LIMIT  1