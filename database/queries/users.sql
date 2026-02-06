-- name: InsertUser :exec
INSERT INTO users (name, username, password_hash, household)
VALUES ($1, $2, $3, $4);

-- name: UpdateUser :exec
UPDATE users
SET name = $1, password_hash = $2
WHERE username = $3;

-- name: GetUserByUsername :one
SELECT id, name, household, username, password_hash, created_at
FROM users
WHERE username = $1;