DROP INDEX IF EXISTS idx_pantry_status;
DROP INDEX IF EXISTS idx_pantry_expires_on;
DROP INDEX IF EXISTS idx_pantry_item_user;
DROP INDEX IF EXISTS idx_pantry_item_household;
DROP TABLE IF EXISTS pantry;
DROP INDEX IF EXISTS idx_meal_components_sub;
DROP TABLE IF EXISTS meal_components;
ALTER TABLE shopping_items DROP COLUMN IF EXISTS shelf_life_days;
