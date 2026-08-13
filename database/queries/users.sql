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
    COALESCE(
        JSON_AGG(
            JSON_BUILD_OBJECT('household_id', h.household_id, 'name', COALESCE(h.name, ''))
            ORDER BY h.household_id
        ) FILTER (WHERE h.household_id IS NOT NULL),
        '[]'
    ) AS households
FROM users u
LEFT JOIN household_members hm ON u.id = hm.user_id
LEFT JOIN households h ON h.household_id = hm.household_id
WHERE u.username = $1
GROUP BY u.id;

-- name: UpdateUserHouseholdMemberships :exec
INSERT INTO household_members (household_id, user_id)
VALUES ($2, $1);

-- name: GetPreviousSeen :one
-- The prior-login timestamp; anchors the cook-review lookback window.
SELECT previous_seen_at FROM users WHERE id = $1;

-- name: UpdateLastSeen :one
-- Snapshot the prior last_seen_at into previous_seen_at, then stamp now().
-- previous_seen_at anchors the login cook-review window.
UPDATE users
SET previous_seen_at = last_seen_at,
    last_seen_at     = now()
WHERE id = $1
RETURNING previous_seen_at;
