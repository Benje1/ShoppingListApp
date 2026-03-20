-- name: AddMealComponent :exec
INSERT INTO meal_components (parent_meal_id, sub_meal_id, sort_order)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: RemoveMealComponent :exec
DELETE FROM meal_components
WHERE parent_meal_id = $1 AND sub_meal_id = $2;

-- name: GetMealComponents :many
-- Sub-meals that make up a composite meal, with full meal details.
SELECT
    mc.sort_order,
    m.id,
    m.name,
    m.description,
    m.default_portions
FROM meal_components mc
JOIN meals m ON m.id = mc.sub_meal_id
WHERE mc.parent_meal_id = $1
ORDER BY mc.sort_order, m.name;

-- name: GetParentMeals :many
-- Which composite meals use this meal as a component?
SELECT
    mc.parent_meal_id,
    m.name AS parent_name
FROM meal_components mc
JOIN meals m ON m.id = mc.parent_meal_id
WHERE mc.sub_meal_id = $1;

-- name: UpdateShoppingItemShelfLife :one
UPDATE shopping_items
SET shelf_life_days = $2
WHERE id = $1
RETURNING *;
