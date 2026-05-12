-- Migration 013: household-scoped meal cooks
--
-- Before this migration, meal_cooks only recorded (meal_id, user_id), meaning
-- a cook assignment was global across every household.
-- After this migration:
--   • meal_cooks also carries an optional household_id.
--     NULL means the assignment applies in every household the cook belongs to
--     (i.e. the original behaviour is preserved for existing rows).
--     A non-NULL value restricts the assignment to that one household.
--   • meals gains an optional household_id so that a meal can be marked as
--     belonging to a specific household (household-specific meals).
--     NULL means the meal is shared / global.

-- 1. Add household_id to meals (NULL = global/shared meal)
ALTER TABLE meals
    ADD COLUMN IF NOT EXISTS household_id INT REFERENCES households(household_id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_meals_household
    ON meals (household_id) WHERE household_id IS NOT NULL;

-- 2. Drop the old PK on meal_cooks (meal_id, user_id)
--    and re-create it to include household_id.
--    We use a nullable household_id: NULL means "any/all households".
ALTER TABLE meal_cooks
    DROP CONSTRAINT meal_cooks_pkey;

ALTER TABLE meal_cooks
    ADD COLUMN IF NOT EXISTS household_id INT REFERENCES households(household_id) ON DELETE CASCADE;

-- Unique: one cook entry per (meal, user, household).
-- We need two partial unique indexes because NULL != NULL in SQL.
CREATE UNIQUE INDEX idx_meal_cooks_meal_user_household
    ON meal_cooks (meal_id, user_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_meal_cooks_meal_user_no_household
    ON meal_cooks (meal_id, user_id) WHERE household_id IS NULL;

-- Restore a surrogate PK so foreign keys / ORMs stay happy
ALTER TABLE meal_cooks ADD COLUMN IF NOT EXISTS id SERIAL PRIMARY KEY;
