DROP INDEX IF EXISTS idx_meal_plan_day_user;
DROP INDEX IF EXISTS idx_meal_plan_day_household;
ALTER TABLE meal_plan ADD CONSTRAINT meal_plan_day_name_household_id_user_id_key
    UNIQUE (day_name, household_id, user_id);
