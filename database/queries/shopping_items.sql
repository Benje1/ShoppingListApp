-- name: CreateShoppingItem :one
INSERT INTO shopping_items (name, item_type, portions_per_unit)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateShoppingItemPortions :one
UPDATE shopping_items
SET portions_per_unit = $2
WHERE id = $1
RETURNING *;

-- name: ListShoppingItems :many
SELECT id, name, item_type, portions_per_unit
FROM shopping_items;

-- name: GetAllShoppingItems :many
SELECT id, name, item_type, portions_per_unit
FROM shopping_items;
