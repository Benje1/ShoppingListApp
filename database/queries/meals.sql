-- name: CreateMeal :one
INSERT INTO meals (name, description, default_portions, season, household_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetMeal :one
SELECT * FROM meals
WHERE id = $1;

-- name: ListMeals :many
-- Lists all global meals plus meals belonging to the given household.
-- Pass NULL for household_id to get only global meals.
SELECT * FROM meals
WHERE household_id IS NULL
   OR household_id = sqlc.narg('household_id')
ORDER BY name;

-- name: UpdateMeal :one
UPDATE meals
SET name             = $2,
    description      = $3,
    default_portions = $4,
    season           = $5,
    household_id     = $6
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
    m.season,
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
-- Lists all global meals plus meals belonging to the given household.
SELECT
    m.*,
    COUNT(mi.shopping_item_id) AS ingredient_count
FROM meals m
LEFT JOIN meal_ingredients mi ON mi.meal_id = m.id
WHERE m.household_id IS NULL
   OR m.household_id = sqlc.narg('household_id')
GROUP BY m.id
ORDER BY m.name;

-- name: AddMealCook :exec
-- Assign a cook to a meal, optionally scoped to a specific household.
-- Pass NULL for household_id to create a cross-household assignment.
INSERT INTO meal_cooks (meal_id, user_id, household_id)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING;

-- name: RemoveMealCook :exec
-- Remove a cook assignment. household_id must match exactly (NULL removes the
-- cross-household assignment; a specific ID removes that household's assignment).
DELETE FROM meal_cooks
WHERE meal_id = $1 AND user_id = $2
  AND (
      (household_id IS NULL     AND $3::INT IS NULL) OR
      (household_id IS NOT NULL AND household_id = $3)
  );

-- name: GetMealCooks :many
-- Returns all cook assignments for a meal, with their household scope.
SELECT
    u.id,
    u.name,
    u.username,
    mc.household_id,
    h.name AS household_name
FROM meal_cooks mc
JOIN users u       ON u.id = mc.user_id
LEFT JOIN households h ON h.household_id = mc.household_id
WHERE mc.meal_id = $1
ORDER BY u.name;

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
    m.season          AS meal_season,
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
-- Meals that a specific user can cook within a given household context:
--   1. Meal is assigned to this user with this exact household_id, OR
--   2. Meal is assigned to this user with no household restriction (NULL), OR
--   3. Meal has no cook assignments at all (universally available).
-- Additionally, household-specific meals are filtered: if a meal has a
-- household_id set, it is only returned when the caller's household matches.
SELECT m.id, m.name, m.description, m.default_portions, m.season, m.household_id
FROM meals m
WHERE
    -- Respect household-specific meals: only show them to the right household
    (m.household_id IS NULL OR m.household_id = sqlc.narg('household_id'))
    AND
    (
        -- Assigned to this user for this specific household
        EXISTS (
            SELECT 1 FROM meal_cooks mc
            WHERE mc.meal_id = m.id
              AND mc.user_id = $1
              AND mc.household_id = sqlc.narg('household_id')
        )
        OR
        -- Assigned to this user with no household restriction
        EXISTS (
            SELECT 1 FROM meal_cooks mc
            WHERE mc.meal_id = m.id
              AND mc.user_id = $1
              AND mc.household_id IS NULL
        )
        OR
        -- No cook assigned at all (available to everyone)
        NOT EXISTS (SELECT 1 FROM meal_cooks mc WHERE mc.meal_id = m.id)
    )
ORDER BY m.name;
