-- name: UpsertMealPhoto :exec
-- Store an uploaded photo content-addressed by its SHA-256. Idempotent: the same
-- image uploaded twice keeps the single existing row.
INSERT INTO meal_photos (id, content_type, bytes, byte_size)
VALUES ($1, $2, $3, $4)
ON CONFLICT (id) DO NOTHING;

-- name: GetMealPhoto :one
SELECT content_type, bytes FROM meal_photos WHERE id = $1;
