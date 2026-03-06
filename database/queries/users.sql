-- name: InsertUser :one
INSERT INTO users (name, username, password_hash, household)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET name = $1, password_hash = $2
WHERE username = $3
RETURNING *;

-- name: GetUserByUsername :one
SELECT id, name, household, username, password_hash, created_at
FROM users
WHERE username = $1;