-- name: CreateMeal :one
INSERT INTO meals (name, description, default_portions)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetMeal :one
SELECT * FROM meals
WHERE id = $1;

-- name: ListMeals :many
SELECT * FROM meals
ORDER BY name;

-- name: UpdateMeal :one
UPDATE meals
SET name             = $2,
    description      = $3,
    default_portions = $4
WHERE id = $1
RETURNING *;

-- name: DeleteMeal :exec
DELETE FROM meals
WHERE id = $1;

-- name: AddMealIngredient :one
INSERT INTO meal_ingredients (meal_id, shopping_item_id, quantity, unit)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateMealIngredient :one
UPDATE meal_ingredients
SET quantity = $3,
    unit     = $4
WHERE meal_id = $1 AND shopping_item_id = $2
RETURNING *;

-- name: RemoveMealIngredient :exec
DELETE FROM meal_ingredients
WHERE meal_id = $1 AND shopping_item_id = $2;

-- name: GetMealWithIngredients :many
-- Returns one row per ingredient; join in application code to build the full meal.
SELECT
    m.id              AS meal_id,
    m.name            AS meal_name,
    m.description     AS meal_description,
    m.default_portions,
    mi.shopping_item_id,
    mi.quantity,
    mi.unit,
    si.name           AS ingredient_name,
    si.item_type      AS ingredient_type,
    si.portions_per_unit
FROM meals m
JOIN meal_ingredients mi ON mi.meal_id = m.id
JOIN shopping_items si   ON si.id = mi.shopping_item_id
WHERE m.id = $1
ORDER BY si.name;

-- name: ListMealsWithIngredientCount :many
SELECT
    m.*,
    COUNT(mi.shopping_item_id) AS ingredient_count
FROM meals m
LEFT JOIN meal_ingredients mi ON mi.meal_id = m.id
GROUP BY m.id
ORDER BY m.name;
