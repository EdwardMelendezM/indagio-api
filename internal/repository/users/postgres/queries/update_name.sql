UPDATE users
SET    name       = $1,
       updated_at = NOW()
WHERE  id         = $2
  AND  deleted_at IS NULL