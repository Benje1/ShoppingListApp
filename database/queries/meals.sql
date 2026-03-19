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

-- name: AddMealCook :exec
INSERT INTO meal_cooks (meal_id, user_id) VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveMealCook :exec
DELETE FROM meal_cooks WHERE meal_id = $1 AND user_id = $2;

-- name: GetMealCooks :many
SELECT u.id, u.name, u.username
FROM meal_cooks mc
JOIN users u ON u.id = mc.user_id
WHERE mc.meal_id = $1;

-- name: GetMealPlanFull :many
-- Returns the week plan with full meal details joined in.
SELECT
    mp.id,
    mp.day_name,
    mp.meal_id,
    mp.cook_user_id,
    mp.updated_at,
    m.name            AS meal_name,
    m.default_portions,
    m.description     AS meal_description,
    cu.name           AS cook_name,
    cu.username       AS cook_username
FROM meal_plan mp
LEFT JOIN meals m ON m.id = mp.meal_id
LEFT JOIN users cu ON cu.id = mp.cook_user_id
WHERE mp.household_id = $1 OR mp.user_id = $2
ORDER BY CASE mp.day_name
    WHEN 'Monday' THEN 1 WHEN 'Tuesday' THEN 2 WHEN 'Wednesday' THEN 3
    WHEN 'Thursday' THEN 4 WHEN 'Friday' THEN 5 WHEN 'Saturday' THEN 6
    WHEN 'Sunday' THEN 7 ELSE 8 END;

-- name: SetMealPlanDay :one
INSERT INTO meal_plan (day_name, meal_id, cook_user_id, household_id, user_id, updated_at)
VALUES ($1, $2, $3, $4, $5, now())
ON CONFLICT (day_name, household_id, user_id)
DO UPDATE SET
    meal_id      = EXCLUDED.meal_id,
    cook_user_id = EXCLUDED.cook_user_id,
    updated_at   = now()
RETURNING *;

-- name: ClearMealPlanDay :exec
DELETE FROM meal_plan
WHERE day_name = $1
  AND (household_id = $2 OR user_id = $3);

-- name: GetMealsForCook :many
-- Meals that a specific user can cook (they appear in meal_cooks).
SELECT m.id, m.name, m.description, m.default_portions
FROM meals m
JOIN meal_cooks mc ON mc.meal_id = m.id
WHERE mc.user_id = $1
ORDER BY m.name;
