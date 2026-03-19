DROP TABLE IF EXISTS meal_ingredients;
DROP TABLE IF EXISTS meals;

ALTER TABLE shopping_items
    DROP COLUMN IF EXISTS portions_per_unit;

ALTER TABLE households
    DROP COLUMN IF EXISTS num_people;
