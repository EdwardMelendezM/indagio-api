SELECT id, email, created_at
FROM   user_adm
WHERE  email      = $1
  AND  deleted_at IS NULL
LIMIT  1