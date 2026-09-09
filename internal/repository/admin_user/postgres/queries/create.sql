INSERT INTO user_adm (email)
VALUES ($1)
RETURNING id, created_at