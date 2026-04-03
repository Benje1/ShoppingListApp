-- Migration 009: Add repeating/temp cook and repeating/temp meal to meal_plan
--
-- Each day of the week can now carry:
--   repeating_cook_user_id  – the person who always handles this weekday
--   temp_cook_user_id       – a one-off override for the coming week
--   repeating_meal_id       – the meal that recurs every week on this day
--   temp_meal_id            – a one-off override meal for the coming week
--
-- Resolution order (highest priority first):
--   1. temp_cook / temp_meal   (used this week, then cleared)
--   2. repeating_cook / repeating_meal (fallback for every future week)
--
-- The existing cook_user_id and meal_id columns remain as the "effective"
-- resolved values that the shopping-list logic already reads, so no other
-- code needs to change for the current week's plan to work correctly.

ALTER TABLE meal_plan
    ADD COLUMN IF NOT EXISTS repeating_cook_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS temp_cook_user_id      INT REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS repeating_meal_id      INT REFERENCES meals(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS temp_meal_id           INT REFERENCES meals(id) ON DELETE SET NULL;

COMMENT ON COLUMN meal_plan.repeating_cook_user_id IS
    'Person who cooks on this weekday every week (standing assignment)';
COMMENT ON COLUMN meal_plan.temp_cook_user_id IS
    'One-off cook override for the next occurrence; cleared after the week rolls over';
COMMENT ON COLUMN meal_plan.repeating_meal_id IS
    'Meal served on this weekday every week (standing assignment)';
COMMENT ON COLUMN meal_plan.temp_meal_id IS
    'One-off meal override for the next occurrence; cleared after the week rolls over';
