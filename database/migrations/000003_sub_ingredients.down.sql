-- Reverse the sub-ingredients & ingredient-allergens change.

DROP TABLE IF EXISTS sub_ingredients;

ALTER TABLE shopping_items DROP COLUMN IF EXISTS allergens;
