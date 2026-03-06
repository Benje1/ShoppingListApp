-- name: InsertHousehold :one
INSERT INTO households (household_id)
VALUES ($1)
RETURNING *;

-- name: GetHousehold :one
SELECT household_id
FROM households
WHERE household_id = $1;

-- name: DeleteHousehold :exec
DELETE FROM households
WHERE household_id = $1;