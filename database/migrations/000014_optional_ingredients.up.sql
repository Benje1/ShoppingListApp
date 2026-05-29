-- Allow ingredients on a meal to be marked as optional.
-- Optional ingredients are shown to the user but excluded from the shopping list.
ALTER TABLE meal_ingredients
    ADD COLUMN IF NOT EXISTS optional BOOLEAN NOT NULL DEFAULT false;
