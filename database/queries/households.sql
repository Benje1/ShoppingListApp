-- name: InsertHousehold :one
INSERT INTO households (num_people, name)
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

-- name: RenameHousehold :one
UPDATE households
SET name = $2
WHERE household_id = $1
RETURNING *;

-- name: DeleteHousehold :exec
DELETE FROM households
WHERE household_id = $1;

-- name: GetHouseholdMembers :many
SELECT u.id, u.name, u.username
FROM users u
JOIN household_members hm ON hm.user_id = u.id
WHERE hm.household_id = $1;

-- name: CreateInvite :one
INSERT INTO household_invites (household_id, invite_code, requested_by_user_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetInviteByCode :one
SELECT * FROM household_invites
WHERE invite_code = $1;

-- name: GetPendingInvitesForHousehold :many
SELECT hi.id, hi.household_id, hi.invite_code, hi.requested_by_user_id, hi.status, hi.created_at,
       u.name AS requester_name, u.username AS requester_username
FROM household_invites hi
JOIN users u ON u.id = hi.requested_by_user_id
WHERE hi.household_id = $1 AND hi.status = 'pending'
ORDER BY hi.created_at;

-- name: RespondToInvite :one
UPDATE household_invites
SET status = $2
WHERE id = $1
RETURNING *;

-- name: GetInviteByID :one
SELECT * FROM household_invites
WHERE id = $1;
