INSERT INTO users (id, name, email, password, role, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
    RETURNING id, name, email, role, avatar_url, avatar_version, verified, created_at