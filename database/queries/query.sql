-- name: CreateShoppingItem :exec
INSERT INTO shopping_items (name, item_type)
VALUES ($1, $2);

-- name: ListShoppingItems :many
SELECT id, name, item_type
FROM shopping_items;
