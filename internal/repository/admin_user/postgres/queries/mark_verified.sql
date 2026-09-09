UPDATE user_adm
SET verified = true
WHERE  email      = $1
  AND  deleted_at IS NULL