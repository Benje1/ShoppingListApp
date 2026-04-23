-- Migration 010: Add week_start date to meal_plan
--
-- The meal_plan table previously stored one row per day-of-week name per scope
-- (household or user), with no notion of which calendar week the plan belonged
-- to. This made it impossible to hold more than one week's worth of data.
--
-- This migration adds a `week_start DATE` column (always the Monday of the
-- relevant week) so that each (day_name, week_start, scope) triplet is unique
-- and multiple weeks can coexist in the table.
--
-- Existing rows are backfilled to the Monday of the current week so they
-- remain valid after the migration.

-- 1. Add the column as nullable first so we can backfill before enforcing NOT NULL.
ALTER TABLE meal_plan
    ADD COLUMN IF NOT EXISTS week_start DATE;

-- 2. Backfill existing rows to the Monday of the current ISO week.
UPDATE meal_plan
SET week_start = DATE_TRUNC('week', CURRENT_DATE)::DATE
WHERE week_start IS NULL;

-- 3. Now enforce NOT NULL with a sensible default for any future direct inserts.
ALTER TABLE meal_plan
    ALTER COLUMN week_start SET NOT NULL,
    ALTER COLUMN week_start SET DEFAULT (DATE_TRUNC('week', CURRENT_DATE)::DATE);

-- 4. Drop the old partial unique indexes (they no longer uniquely identify a row
--    now that week_start is part of the key).
DROP INDEX IF EXISTS idx_meal_plan_day_household;
DROP INDEX IF EXISTS idx_meal_plan_day_user;

-- 5. Create new partial unique indexes that include week_start.
CREATE UNIQUE INDEX idx_meal_plan_day_week_household
    ON meal_plan (day_name, week_start, household_id)
    WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_meal_plan_day_week_user
    ON meal_plan (day_name, week_start, user_id)
    WHERE user_id IS NOT NULL;
