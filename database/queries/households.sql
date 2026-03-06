-- name: InsertHousehold :one
INSERT INTO households (household_id)
VALUES ($1)
RETURNING *;