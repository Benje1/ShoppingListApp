ALTER TABLE meal_ingredients
    DROP CONSTRAINT IF EXISTS ingredient_option_group_type_consistent,
    DROP COLUMN IF EXISTS option_type,
    DROP COLUMN IF EXISTS option_group,
    ADD COLUMN IF NOT EXISTS optional BOOLEAN NOT NULL DEFAULT false;
