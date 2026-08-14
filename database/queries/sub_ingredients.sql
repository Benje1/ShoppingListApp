-- name: AddSubIngredient :one
-- Attach a lightweight sub-ingredient (name + allergens) to a shopping item.
INSERT INTO sub_ingredients (shopping_item_id, name, allergens, sort_order)
VALUES (
    sqlc.arg(shopping_item_id), sqlc.arg(name),
    COALESCE(sqlc.arg(allergens)::text[], '{}'), sqlc.arg(sort_order)
)
RETURNING *;

-- name: UpdateSubIngredient :one
UPDATE sub_ingredients
SET name       = sqlc.arg(name),
    allergens  = COALESCE(sqlc.arg(allergens)::text[], '{}'),
    sort_order = sqlc.arg(sort_order)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: RemoveSubIngredient :exec
DELETE FROM sub_ingredients
WHERE id = $1;

-- name: ListSubIngredientsForItem :many
-- All sub-ingredients belonging to a single shopping item.
SELECT id, shopping_item_id, name, allergens, sort_order
FROM sub_ingredients
WHERE shopping_item_id = $1
ORDER BY sort_order, name;

-- name: ListSubIngredientsForMeal :many
-- Every sub-ingredient reachable from a meal via its ingredients, so the
-- meal view can render each ingredient's breakdown in one round trip.
SELECT
    sub.id,
    sub.shopping_item_id,
    sub.name,
    sub.allergens,
    sub.sort_order
FROM sub_ingredients sub
JOIN meal_ingredients mi ON mi.shopping_item_id = sub.shopping_item_id
WHERE mi.meal_id = $1
ORDER BY sub.shopping_item_id, sub.sort_order, sub.name;
