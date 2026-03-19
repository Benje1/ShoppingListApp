-- name: GetShoppingList :many
-- Returns all shopping list items for a user and/or their household,
-- with a scope field so the frontend knows which is which.
SELECT
    sl.id,
    si.id   AS item_id,
    si.name,
    si.item_type,
    si.portions_per_unit,
    sl.quantity,
    sl.updated_at,
    CASE WHEN sl.household_id IS NOT NULL THEN 'household' ELSE 'personal' END AS scope
FROM shopping_list sl
JOIN shopping_items si ON sl.shopping_item_id = si.id
WHERE sl.household_id = $1 OR sl.user_id = $2
ORDER BY si.item_type, si.name;

-- name: GetShoppingListUpdatedAt :one
-- Returns the most recent updated_at across all entries for this user/household.
SELECT MAX(updated_at) AS last_updated
FROM shopping_list
WHERE household_id = $1 OR user_id = $2;

-- name: AddToShoppingList :one
INSERT INTO shopping_list (shopping_item_id, quantity, household_id, user_id, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (shopping_item_id, household_id, user_id)
DO UPDATE SET
    quantity   = shopping_list.quantity + EXCLUDED.quantity,
    updated_at = now()
RETURNING *;

-- name: RemoveFromShoppingList :exec
DELETE FROM shopping_list
WHERE id = $1;

-- name: UpsertShoppingList :many
INSERT INTO shopping_list (shopping_item_id, quantity, id, household_id, user_id, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT DO UPDATE
SET shopping_item_id = EXCLUDED.shopping_item_id,
    quantity         = EXCLUDED.quantity,
    household_id     = EXCLUDED.household_id,
    user_id          = EXCLUDED.user_id,
    updated_at       = now()
RETURNING *;

-- name: GetMealPlan :many
SELECT id, day_name, meal_name, household_id, user_id, updated_at
FROM meal_plan
WHERE household_id = $1 OR user_id = $2
ORDER BY CASE day_name
    WHEN 'Monday'    THEN 1 WHEN 'Tuesday'   THEN 2 WHEN 'Wednesday' THEN 3
    WHEN 'Thursday'  THEN 4 WHEN 'Friday'    THEN 5 WHEN 'Saturday'  THEN 6
    WHEN 'Sunday'    THEN 7 ELSE 8 END;

-- name: GetMealPlanUpdatedAt :one
SELECT MAX(updated_at) AS last_updated
FROM meal_plan
WHERE household_id = $1 OR user_id = $2;

-- name: UpsertMealPlanDay :one
INSERT INTO meal_plan (day_name, meal_name, household_id, user_id, updated_at)
VALUES ($1, $2, $3, $4, now())
ON CONFLICT (day_name, household_id, user_id)
DO UPDATE SET meal_name = EXCLUDED.meal_name, updated_at = now()
RETURNING *;

-- name: GetHaveIt :many
SELECT shopping_item_id, updated_at
FROM shopping_list_have_it
WHERE household_id = $1 OR user_id = $2
ORDER BY updated_at DESC;

-- name: GetHaveItUpdatedAt :one
SELECT MAX(updated_at) AS last_updated
FROM shopping_list_have_it
WHERE household_id = $1 OR user_id = $2;

-- name: MarkHaveIt :one
INSERT INTO shopping_list_have_it (shopping_item_id, household_id, user_id, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (shopping_item_id, household_id) WHERE household_id IS NOT NULL
DO UPDATE SET updated_at = now()
RETURNING *;

-- name: UnmarkHaveIt :exec
DELETE FROM shopping_list_have_it
WHERE shopping_item_id = $1
  AND (household_id = $2 OR user_id = $3);
