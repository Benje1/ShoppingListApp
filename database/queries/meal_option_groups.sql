-- name: GetMealOptionGroups :many
-- Returns all option group entries for a meal, ordered by group then sort_order.
SELECT
    e.id,
    e.meal_id,
    e.option_group,
    e.option_type,
    e.sort_order,
    e.shopping_item_id,
    e.sub_meal_id,
    si.name  AS item_name,
    sm.name  AS sub_meal_name
FROM meal_option_group_entries e
LEFT JOIN shopping_items si ON si.id = e.shopping_item_id
LEFT JOIN meals          sm ON sm.id = e.sub_meal_id
WHERE e.meal_id = $1
ORDER BY e.option_group, e.sort_order, e.id;

-- name: AddMealOptionGroupEntry :one
INSERT INTO meal_option_group_entries
    (meal_id, option_group, option_type, sort_order, shopping_item_id, sub_meal_id)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateMealOptionGroupEntry :one
UPDATE meal_option_group_entries
SET option_group     = $2,
    option_type      = $3,
    sort_order       = $4,
    shopping_item_id = $5,
    sub_meal_id      = $6
WHERE id = $1
RETURNING *;

-- name: RemoveMealOptionGroupEntry :exec
DELETE FROM meal_option_group_entries
WHERE id = $1 AND meal_id = $2;
