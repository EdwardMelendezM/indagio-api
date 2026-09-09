SELECT code, expires_at
FROM otps
WHERE email = $1 AND used = false
ORDER BY expires_at DESC
    LIMIT 1