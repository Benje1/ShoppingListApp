-- name: InsertHousehold :one
INSERT INTO households (household_id, num_people)
VALUES ($1, $2)
RETURNING *;

-- name: GetHousehold :one
SELECT * FROM households
WHERE household_id = $1;

-- name: UpdateHouseholdNumPeople :one
UPDATE households
SET num_people = $2
WHERE household_id = $1
RETURNING *;

-- name: DeleteHousehold :exec
DELETE FROM households
WHERE household_id = $1;
