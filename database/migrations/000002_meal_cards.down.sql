-- Reverse the meal-card refactor.

DROP TABLE IF EXISTS meal_cook_log;

ALTER TABLE users DROP COLUMN IF EXISTS previous_seen_at;
ALTER TABLE users DROP COLUMN IF EXISTS last_seen_at;

ALTER TABLE meal_ingredients DROP COLUMN IF EXISTS optional;

ALTER TABLE meals DROP COLUMN IF EXISTS allergens;
ALTER TABLE meals DROP COLUMN IF EXISTS recipe;
ALTER TABLE meals DROP COLUMN IF EXISTS photo_url;
