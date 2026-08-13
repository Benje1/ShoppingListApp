-- name: CreateShoppingItem :one
INSERT INTO shopping_items (name, item_type, portions_per_unit, allergens)
VALUES (
    sqlc.arg(name), sqlc.arg(item_type), sqlc.arg(portions_per_unit),
    COALESCE(sqlc.arg(allergens)::text[], '{}')
)
RETURNING *;

-- name: UpdateShoppingItemPortions :one
UPDATE shopping_items
SET portions_per_unit = $2
WHERE id = $1
RETURNING *;

-- name: UpdateShoppingItem :one
-- Partial update: pass NULL for any field to leave it unchanged. Allergens
-- follow the same rule — NULL keeps the existing tags, '{}' clears them.
UPDATE shopping_items
SET
    name              = COALESCE(sqlc.narg(name), name),
    item_type         = COALESCE(sqlc.narg(item_type), item_type),
    portions_per_unit = COALESCE(sqlc.narg(portions_per_unit), portions_per_unit),
    allergens         = COALESCE(sqlc.narg(allergens)::text[], allergens)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListShoppingItems :many
SELECT id, name, item_type, portions_per_unit, allergens
FROM shopping_items;

-- name: GetAllShoppingItems :many
SELECT id, name, item_type, portions_per_unit, allergens
FROM shopping_items;
