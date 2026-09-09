SELECT id, name, email, password, role, avatar_url, avatar_version, verified, created_at
FROM users
WHERE email = $1