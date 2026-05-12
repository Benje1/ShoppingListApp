-- Rollback migration 013

ALTER TABLE meal_cooks DROP COLUMN IF EXISTS id;
DROP INDEX IF EXISTS idx_meal_cooks_meal_user_household;
DROP INDEX IF EXISTS idx_meal_cooks_meal_user_no_household;
ALTER TABLE meal_cooks DROP COLUMN IF EXISTS household_id;
ALTER TABLE meal_cooks ADD PRIMARY KEY (meal_id, user_id);

DROP INDEX IF EXISTS idx_meals_household;
ALTER TABLE meals DROP COLUMN IF EXISTS household_id;
