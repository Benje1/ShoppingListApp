-- name: CreateShoppingItem :one
INSERT INTO shopping_items (name, item_type)
VALUES ($1, $2)
RETURNING *;

-- name: ListShoppingItems :many
SELECT id, name, item_type
FROM shopping_items;

-- name: GetAllShoppingItems :many
SELECT id, name, item_type
FROM shopping_items;