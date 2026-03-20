-- name: GetPantry :many
-- Returns all pantry entries for a user/household, joined with item details.
SELECT
    p.id,
    p.shopping_item_id,
    si.name             AS item_name,
    si.item_type,
    si.portions_per_unit,
    si.shelf_life_days,
    p.portions_remaining,
    p.expires_on,
    p.status,
    p.bought_at,
    p.updated_at
FROM pantry p
JOIN shopping_items si ON si.id = p.shopping_item_id
WHERE p.household_id = $1 OR p.user_id = $2
ORDER BY p.status DESC, p.expires_on ASC NULLS LAST, si.name;

-- name: UpsertPantryItem :one
-- Add or top-up an item in the pantry. On conflict, adds portions and
-- resets expires_on to the new purchase date + shelf life.
INSERT INTO pantry (shopping_item_id, household_id, user_id, portions_remaining, expires_on, status, updated_at)
VALUES ($1, $2, $3, $4, $5, 'fresh', now())
ON CONFLICT (shopping_item_id, household_id) WHERE household_id IS NOT NULL
DO UPDATE SET
    portions_remaining = pantry.portions_remaining + EXCLUDED.portions_remaining,
    expires_on         = CASE WHEN EXCLUDED.expires_on IS NOT NULL THEN EXCLUDED.expires_on ELSE pantry.expires_on END,
    status             = 'fresh',
    updated_at         = now()
RETURNING *;

-- name: DecrementPantryPortions :one
-- Called when a meal is cooked. Decrements portions by the given amount.
-- Clamps to 0 — never goes negative.
UPDATE pantry
SET portions_remaining = GREATEST(0, portions_remaining - $3),
    updated_at         = now()
WHERE shopping_item_id = $1
  AND (household_id = $2 OR user_id = $4)
RETURNING *;

-- name: RemovePantryItem :exec
DELETE FROM pantry WHERE id = $1;

-- name: ExpirePantryItems :many
-- Called by the scheduled job. Marks items as expiring_soon or expired.
UPDATE pantry
SET status     = CASE
                    WHEN expires_on < CURRENT_DATE                    THEN 'expired'
                    WHEN expires_on <= CURRENT_DATE + INTERVAL '2 days' THEN 'expiring_soon'
                    ELSE status
                 END,
    updated_at = now()
WHERE expires_on IS NOT NULL
  AND status <> 'expired'
RETURNING id, shopping_item_id, household_id, user_id, status, expires_on;

-- name: GetExpiredPantryItems :many
SELECT p.id, p.shopping_item_id, si.name AS item_name, p.expires_on, p.status,
       p.household_id, p.user_id
FROM pantry p
JOIN shopping_items si ON si.id = p.shopping_item_id
WHERE p.status IN ('expiring_soon', 'expired')
ORDER BY p.expires_on ASC;
