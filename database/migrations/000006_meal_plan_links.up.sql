-- meal_plan: replace free-text meal_name with a proper meal reference,
-- and add an optional cook assignment per day.
ALTER TABLE meal_plan
    ADD COLUMN IF NOT EXISTS meal_id      INT REFERENCES meals(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS cook_user_id INT REFERENCES users(id) ON DELETE SET NULL;

-- Keep meal_name for backward compat but make it nullable so old rows don't break
ALTER TABLE meal_plan
    ALTER COLUMN meal_name DROP NOT NULL,
    ALTER COLUMN meal_name SET DEFAULT NULL;

-- Which users can cook a given meal (many-to-many)
CREATE TABLE IF NOT EXISTS meal_cooks (
    meal_id INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (meal_id, user_id)
);
