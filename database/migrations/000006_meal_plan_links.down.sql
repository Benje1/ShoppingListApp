DROP TABLE IF EXISTS meal_cooks;
ALTER TABLE meal_plan
    DROP COLUMN IF EXISTS cook_user_id,
    DROP COLUMN IF EXISTS meal_id,
    ALTER COLUMN meal_name SET NOT NULL,
    ALTER COLUMN meal_name SET DEFAULT '';
