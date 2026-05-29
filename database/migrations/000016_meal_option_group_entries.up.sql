-- Option groups are moved off meal_ingredients into their own table so that
-- entries can reference either a shopping item or a whole sub-meal.

ALTER TABLE meal_ingredients
    DROP CONSTRAINT IF EXISTS ingredient_option_group_type_consistent,
    DROP COLUMN IF EXISTS option_type,
    DROP COLUMN IF EXISTS option_group;

-- Each row is one choice within an option group for a meal.
-- Exactly one of shopping_item_id or sub_meal_id must be set.
CREATE TABLE IF NOT EXISTS meal_option_group_entries (
    id               SERIAL PRIMARY KEY,
    meal_id          INT  NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    option_group     TEXT NOT NULL, -- user-defined label, e.g. "potatoes", "meat"
    option_type      TEXT NOT NULL CHECK (option_type IN ('one_of', 'many_of')),
    sort_order       INT  NOT NULL DEFAULT 0,
    -- exactly one of these two columns is set
    shopping_item_id INT  REFERENCES shopping_items(id) ON DELETE CASCADE,
    sub_meal_id      INT  REFERENCES meals(id)          ON DELETE CASCADE,
    CHECK (
        (shopping_item_id IS NOT NULL AND sub_meal_id IS NULL) OR
        (shopping_item_id IS NULL     AND sub_meal_id IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_moge_meal        ON meal_option_group_entries(meal_id);
CREATE INDEX IF NOT EXISTS idx_moge_sub_meal    ON meal_option_group_entries(sub_meal_id);
CREATE INDEX IF NOT EXISTS idx_moge_item        ON meal_option_group_entries(shopping_item_id);
