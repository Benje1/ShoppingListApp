-- name: CreateShoppingItem :one
INSERT INTO shopping_items (name, item_type, portions_per_unit, allergens, base_unit, pack_size, sold_loose)
VALUES (
    sqlc.arg(name), sqlc.arg(item_type), sqlc.arg(portions_per_unit),
    COALESCE(sqlc.arg(allergens)::text[], '{}'),
    sqlc.narg(base_unit),
    COALESCE(sqlc.narg(pack_size)::numeric, 1),
    COALESCE(sqlc.narg(sold_loose)::boolean, false)
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
    allergens         = COALESCE(sqlc.narg(allergens)::text[], allergens),
    base_unit         = COALESCE(sqlc.narg(base_unit), base_unit),
    pack_size         = COALESCE(sqlc.narg(pack_size)::numeric, pack_size),
    sold_loose        = COALESCE(sqlc.narg(sold_loose)::boolean, sold_loose)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: ListShoppingItems :many
SELECT id, name, item_type, portions_per_unit, allergens, base_unit, pack_size, sold_loose
FROM shopping_items;

-- name: GetAllShoppingItems :many
SELECT id, name, item_type, portions_per_unit, allergens, base_unit, pack_size, sold_loose
FROM shopping_items;
