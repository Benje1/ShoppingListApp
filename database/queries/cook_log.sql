-- Cook-review queries: candidate past meals to confirm, and the answer log.
--
-- A meal_plan row's real date is week_start (a Monday) plus the weekday offset
-- of day_name. We compute that inline as cook_date.

-- name: ListCookReviewCandidates :many
-- Planned meals for a scope whose date falls in [since, until] and that the
-- user has not already answered. Pass household_id/user_id per the usual scope
-- convention (exactly one is non-NULL).
SELECT
    (mp.week_start + (CASE mp.day_name
        WHEN 'Monday'    THEN 0 WHEN 'Tuesday'  THEN 1 WHEN 'Wednesday' THEN 2
        WHEN 'Thursday'  THEN 3 WHEN 'Friday'   THEN 4 WHEN 'Saturday'  THEN 5
        WHEN 'Sunday'    THEN 6 ELSE 0 END))::date AS cook_date,
    mp.day_name,
    mp.meal_id,
    m.name             AS meal_name,
    m.default_portions,
    mp.household_id,
    mp.user_id
FROM meal_plan mp
JOIN meals m ON m.id = mp.meal_id
WHERE (mp.household_id = $1 OR mp.user_id = $2)
  AND mp.meal_id IS NOT NULL
  AND (mp.week_start + (CASE mp.day_name
        WHEN 'Monday'    THEN 0 WHEN 'Tuesday'  THEN 1 WHEN 'Wednesday' THEN 2
        WHEN 'Thursday'  THEN 3 WHEN 'Friday'   THEN 4 WHEN 'Saturday'  THEN 5
        WHEN 'Sunday'    THEN 6 ELSE 0 END))::date BETWEEN $3 AND $4
  AND NOT EXISTS (
      SELECT 1 FROM meal_cook_log l
      WHERE l.meal_id = mp.meal_id
        AND l.cook_date = (mp.week_start + (CASE mp.day_name
            WHEN 'Monday'    THEN 0 WHEN 'Tuesday'  THEN 1 WHEN 'Wednesday' THEN 2
            WHEN 'Thursday'  THEN 3 WHEN 'Friday'   THEN 4 WHEN 'Saturday'  THEN 5
            WHEN 'Sunday'    THEN 6 ELSE 0 END))::date
        AND (
            (l.household_id IS NOT NULL AND l.household_id = mp.household_id) OR
            (l.user_id      IS NOT NULL AND l.user_id      = mp.user_id)
        )
  )
ORDER BY cook_date;

-- name: UpsertMealCookLog :exec
-- Record the answer to "did you make this?" Idempotent per (scope, date, meal).
INSERT INTO meal_cook_log (meal_id, cook_date, made, household_id, user_id, answered_by)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT DO NOTHING;
