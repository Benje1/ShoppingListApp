-- name: CreateShoppingItem :one
INSERT INTO shopping_items (name, item_type, portions_per_unit)
VALUES ($1, $2, $3)
RETURNING *;

-- name: UpdateShoppingItemPortions :one
UPDATE shopping_items
SET portions_per_unit = $2
WHERE id = $1
RETURNING *;

-- name: UpdateShoppingItem :one
UPDATE shopping_items
SET
    name             = COALESCE($2, name),
    item_type        = COALESCE($3, item_type),
    portions_per_unit = COALESCE($4, portions_per_unit)
WHERE id = $1
RETURNING *;

-- name: ListShoppingItems :many
SELECT id, name, item_type, portions_per_unit
FROM shopping_items;

-- name: GetAllShoppingItems :many
SELECT id, name, item_type, portions_per_unit
FROM shopping_items;