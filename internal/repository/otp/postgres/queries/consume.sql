UPDATE otps
SET used = true
WHERE email = $1 AND used = false