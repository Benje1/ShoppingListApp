-- name: InsertHousehold :exec
INSERT INTO households (household_id)
VALUES ($1);