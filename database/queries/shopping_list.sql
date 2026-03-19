-- name: UpsertShoppingList :many
INSERT INTO shopping_list (shopping_item_id, quantity, id, household_id, user_id)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT DO UPDATE
SET shopping_item_id = EXCLUDED.shopping_item_id,
    quantity = EXCLUDED.quantity,
    id = EXCLUDED.id,
    household_id = EXCLUDED.household_id,
    user_id = EXCLUDED.user_id
RETURNING *;

-- name: GetShoppingList :many
SELECT name, item_type, text_id, portions_per_unit, quantity 
FROM shopping_list sl
JOIN shopping_items si ON sl.shopping_item_id = si.id
WHERE household_id = $1 OR user_id = $2;