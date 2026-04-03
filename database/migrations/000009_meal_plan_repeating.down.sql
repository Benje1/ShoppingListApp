-- Rollback migration 009
ALTER TABLE meal_plan
    DROP COLUMN IF EXISTS repeating_cook_user_id,
    DROP COLUMN IF EXISTS temp_cook_user_id,
    DROP COLUMN IF EXISTS repeating_meal_id,
    DROP COLUMN IF EXISTS temp_meal_id;
