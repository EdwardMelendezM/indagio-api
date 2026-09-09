UPDATE users
SET    blocked    = true,
       updated_at = NOW()
WHERE  id         = $1
  AND  deleted_at IS NULL
