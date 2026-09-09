INSERT INTO otps (email, code, expires_at, used)
VALUES ($1, $2, $3, false)