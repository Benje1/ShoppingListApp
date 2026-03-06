-- name: InsertUser :one
INSERT INTO users (name, username, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: AddUserToHousehold :exec
INSERT INTO household_members (household_id, user_id)
VALUES ($1, $2);

-- name: UpdateUserName :one
UPDATE users
SET name = $1
WHERE username = $2
RETURNING *;

-- name: UpdateUserPassword :one
UPDATE users
SET password_hash = $1
WHERE username = $2
RETURNING *;

-- name: GetUserByUsername :one
SELECT 
    u.id, 
    u.name, 
    u.username, 
    u.password_hash, 
    u.created_at,
    ARRAY_AGG(hm.household_id) AS household_ids
FROM users u
LEFT JOIN household_members hm ON u.id = hm.user_id
WHERE u.username = $1
GROUP BY u.id;

-- name: UpdateUserHouseholdMemberships :exec
DELETE FROM household_members WHERE user_id = $1;
INSERT INTO household_members (household_id, user_id)
VALUES ($2, $1);
