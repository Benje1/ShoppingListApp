-- Add updated_at to shopping_list so the frontend can cache by timestamp
ALTER TABLE shopping_list
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT now();

-- Store the weekly meal plan per household or per user
-- Each row is one day's entry. updated_at lets the frontend cache by timestamp.
CREATE TABLE IF NOT EXISTS meal_plan (
    id           SERIAL PRIMARY KEY,
    day_name     TEXT NOT NULL,           -- e.g. 'Monday'
    meal_name    TEXT NOT NULL DEFAULT '',
    household_id INT REFERENCES households(household_id) ON DELETE CASCADE,
    user_id      INT REFERENCES users(id) ON DELETE CASCADE,
    updated_at   TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    ),
    UNIQUE (day_name, household_id, user_id)
);

-- Unique constraint needed for ON CONFLICT upsert in AddToShoppingList
-- A given item can only appear once per household or per user
CREATE UNIQUE INDEX IF NOT EXISTS idx_shopping_list_item_household
    ON shopping_list (shopping_item_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_shopping_list_item_user
    ON shopping_list (shopping_item_id, user_id) WHERE user_id IS NOT NULL;
