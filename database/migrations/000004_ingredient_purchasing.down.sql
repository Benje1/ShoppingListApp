-- Reverse 000004_ingredient_purchasing.up.sql
ALTER TABLE meal_ingredients DROP COLUMN IF EXISTS dietary_tags;
ALTER TABLE meal_ingredients DROP COLUMN IF EXISTS quantity_per_portion;
ALTER TABLE shopping_items   DROP COLUMN IF EXISTS sold_loose;
ALTER TABLE shopping_items   DROP COLUMN IF EXISTS pack_size;
ALTER TABLE shopping_items   DROP COLUMN IF EXISTS base_unit;
