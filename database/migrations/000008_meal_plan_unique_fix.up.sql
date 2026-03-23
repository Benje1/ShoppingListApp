-- The original UNIQUE (day_name, household_id, user_id) constraint on meal_plan
-- does not work correctly when household_id or user_id is NULL, because PostgreSQL
-- treats NULL != NULL in standard unique constraints. This allows duplicate rows
-- like multiple "Monday" entries for the same household.
--
-- Fix: drop the broken constraint and replace with two partial unique indexes —
-- one for household-scoped rows, one for personal-scoped rows. Partial indexes
-- DO enforce uniqueness correctly even with NULLs in the non-indexed column.

-- 1. Remove all duplicate rows first, keeping only the most recent per day per scope.
--    We use a CTE to identify the survivors (highest id = most recent upsert).
DELETE FROM meal_plan
WHERE id NOT IN (
    SELECT DISTINCT ON (day_name, household_id, user_id) id
    FROM meal_plan
    ORDER BY day_name, household_id, user_id, id DESC
);

-- 2. Drop the broken table-level unique constraint.
ALTER TABLE meal_plan DROP CONSTRAINT IF EXISTS meal_plan_day_name_household_id_user_id_key;

-- 3. Add correct partial unique indexes.
CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_plan_day_household
    ON meal_plan (day_name, household_id)
    WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_plan_day_user
    ON meal_plan (day_name, user_id)
    WHERE user_id IS NOT NULL;
