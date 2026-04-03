-- name: GetMealPlanFullV2 :many
-- Returns the week plan with all four cook/meal slots plus resolved effective values.
SELECT
    mp.id,
    mp.day_name,

    -- Effective (resolved) values — what actually runs this week
    mp.meal_id,
    mp.cook_user_id,

    -- Repeating defaults
    mp.repeating_meal_id,
    mp.repeating_cook_user_id,

    -- One-off overrides for the coming week
    mp.temp_meal_id,
    mp.temp_cook_user_id,

    mp.updated_at,

    -- Effective meal details
    em.name            AS meal_name,
    em.default_portions,
    em.description     AS meal_description,

    -- Effective cook details
    ecu.name           AS cook_name,
    ecu.username       AS cook_username,

    -- Repeating meal name
    rm.name            AS repeating_meal_name,

    -- Temp meal name
    tm.name            AS temp_meal_name,

    -- Repeating cook name
    rcu.name           AS repeating_cook_name,
    rcu.username       AS repeating_cook_username,

    -- Temp cook name
    tcu.name           AS temp_cook_name,
    tcu.username       AS temp_cook_username

FROM meal_plan mp
LEFT JOIN meals  em  ON em.id  = mp.meal_id
LEFT JOIN users  ecu ON ecu.id = mp.cook_user_id
LEFT JOIN meals  rm  ON rm.id  = mp.repeating_meal_id
LEFT JOIN meals  tm  ON tm.id  = mp.temp_meal_id
LEFT JOIN users  rcu ON rcu.id = mp.repeating_cook_user_id
LEFT JOIN users  tcu ON tcu.id = mp.temp_cook_user_id
WHERE mp.household_id = $1 OR mp.user_id = $2
ORDER BY CASE mp.day_name
    WHEN 'Monday'    THEN 1
    WHEN 'Tuesday'   THEN 2
    WHEN 'Wednesday' THEN 3
    WHEN 'Thursday'  THEN 4
    WHEN 'Friday'    THEN 5
    WHEN 'Saturday'  THEN 6
    WHEN 'Sunday'    THEN 7
    ELSE 8 END;

-- name: SetMealPlanDayV2 :one
-- Upserts a day's entry, including the four repeating/temp slots.
-- Pass 0 for any int field to store NULL.
INSERT INTO meal_plan (
    day_name,
    meal_id,          cook_user_id,
    repeating_meal_id, repeating_cook_user_id,
    temp_meal_id,      temp_cook_user_id,
    household_id, user_id, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, now())
ON CONFLICT (day_name, household_id) WHERE household_id IS NOT NULL
DO UPDATE SET
    meal_id                 = EXCLUDED.meal_id,
    cook_user_id            = EXCLUDED.cook_user_id,
    repeating_meal_id       = EXCLUDED.repeating_meal_id,
    repeating_cook_user_id  = EXCLUDED.repeating_cook_user_id,
    temp_meal_id            = EXCLUDED.temp_meal_id,
    temp_cook_user_id       = EXCLUDED.temp_cook_user_id,
    updated_at              = now()
RETURNING *;

-- name: ClearTempOverridesForDay :exec
-- Called at week rollover: wipes temp fields so the repeating values take effect.
UPDATE meal_plan
SET temp_meal_id      = NULL,
    temp_cook_user_id = NULL,
    -- Promote repeating values into the effective slots
    meal_id           = repeating_meal_id,
    cook_user_id      = repeating_cook_user_id,
    updated_at        = now()
WHERE day_name = $1
  AND (household_id = $2 OR user_id = $3);

-- name: ClearAllTempOverrides :exec
-- Week-rollover bulk clear for a household or user scope.
UPDATE meal_plan
SET temp_meal_id      = NULL,
    temp_cook_user_id = NULL,
    meal_id           = COALESCE(repeating_meal_id, meal_id),
    cook_user_id      = COALESCE(repeating_cook_user_id, cook_user_id),
    updated_at        = now()
WHERE household_id = $1 OR user_id = $2;
