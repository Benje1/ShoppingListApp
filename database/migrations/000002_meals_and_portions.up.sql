-- Add number of people to households
ALTER TABLE households
    ADD COLUMN IF NOT EXISTS num_people INT NOT NULL DEFAULT 1;

-- Add portions per unit to shopping items
ALTER TABLE shopping_items
    ADD COLUMN IF NOT EXISTS portions_per_unit INT NOT NULL DEFAULT 1;

-- Meals table
CREATE TABLE IF NOT EXISTS meals (
    id          SERIAL PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT,
    default_portions INT NOT NULL DEFAULT 2
);

-- Links meals to the shopping_items they require
-- quantity is how much of the item is needed (e.g. 2, 0.5)
-- unit is the measurement (e.g. 'cloves', 'grams', 'ml') — optional, raw count if NULL
CREATE TABLE IF NOT EXISTS meal_ingredients (
    meal_id         INT REFERENCES meals(id) ON DELETE CASCADE,
    shopping_item_id INT REFERENCES shopping_items(id) ON DELETE RESTRICT,
    quantity        NUMERIC(10, 2) NOT NULL DEFAULT 1,
    unit            TEXT,
    PRIMARY KEY (meal_id, shopping_item_id)
);
