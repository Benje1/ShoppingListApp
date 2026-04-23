-- Rollback migration 010
DROP INDEX IF EXISTS idx_meal_plan_day_week_household;
DROP INDEX IF EXISTS idx_meal_plan_day_week_user;

-- Restore the original partial unique indexes from migration 008.
CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_plan_day_household
    ON meal_plan (day_name, household_id)
    WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_plan_day_user
    ON meal_plan (day_name, user_id)
    WHERE user_id IS NOT NULL;

ALTER TABLE meal_plan DROP COLUMN IF EXISTS week_start;
