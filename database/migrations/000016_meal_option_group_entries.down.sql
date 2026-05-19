DROP TABLE IF EXISTS meal_option_group_entries;

ALTER TABLE meal_ingredients
    ADD COLUMN IF NOT EXISTS option_group TEXT NULL,
    ADD COLUMN IF NOT EXISTS option_type  TEXT NULL
        CHECK (option_type IN ('one_of', 'many_of')),
    ADD CONSTRAINT ingredient_option_group_type_consistent
        CHECK (
            (option_group IS NULL AND option_type IS NULL) OR
            (option_group IS NOT NULL AND option_type IS NOT NULL)
        );
