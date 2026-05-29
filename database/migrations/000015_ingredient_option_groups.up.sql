-- Replace the simple optional boolean with named option groups.
--
-- option_group: free-text label grouping variant/optional ingredients together
--               (e.g. "meat", "sides", "sauce"). NULL means the ingredient is
--               always required and always added to the shopping list.
--
-- option_type:  how the user should pick from the group:
--               "one_of"  — pick exactly one (e.g. chicken OR beef OR lamb)
--               "many_of" — pick any subset  (e.g. potatoes, carrots, parsnips)
--               Must be set whenever option_group is set, and NULL otherwise.

ALTER TABLE meal_ingredients
    DROP COLUMN IF EXISTS optional,
    ADD COLUMN IF NOT EXISTS option_group TEXT NULL,
    ADD COLUMN IF NOT EXISTS option_type  TEXT NULL
        CHECK (option_type IN ('one_of', 'many_of'));

-- Ensure option_type is always set when option_group is, and never set without it
ALTER TABLE meal_ingredients
    ADD CONSTRAINT ingredient_option_group_type_consistent
        CHECK (
            (option_group IS NULL AND option_type IS NULL) OR
            (option_group IS NOT NULL AND option_type IS NOT NULL)
        );
