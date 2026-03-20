-- ── Shelf life on shopping items ─────────────────────────────────────────────
-- NULL means the item never expires (tinned goods, cleaning products, etc.)
ALTER TABLE shopping_items
    ADD COLUMN IF NOT EXISTS shelf_life_days INT;

-- ── Meal composition (meal-to-meal mapping) ───────────────────────────────────
-- A "composite" meal (e.g. "Curry Night") contains sub-meals
-- (e.g. Curry, Dahl, Rice) each of which is a full Meal in its own right.
CREATE TABLE IF NOT EXISTS meal_components (
    parent_meal_id INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    sub_meal_id    INT NOT NULL REFERENCES meals(id) ON DELETE RESTRICT,
    sort_order     INT NOT NULL DEFAULT 0,
    PRIMARY KEY (parent_meal_id, sub_meal_id),
    -- Prevent a meal from being a component of itself
    CHECK (parent_meal_id <> sub_meal_id)
);

CREATE INDEX IF NOT EXISTS idx_meal_components_sub ON meal_components(sub_meal_id);

-- ── Pantry — bought stock with portion and expiry tracking ────────────────────
-- Replaces the simple have-it boolean with real inventory.
-- One row per item per household/user.
-- portions_remaining: starts at shopping_items.portions_per_unit * quantity bought,
--                     decremented each time a meal using this ingredient is cooked.
-- expires_on:         now() + shelf_life_days when the item is added to pantry.
--                     NULL if the item has no shelf life.
-- status:             updated by the scheduled expiry job.
CREATE TABLE IF NOT EXISTS pantry (
    id                 SERIAL PRIMARY KEY,
    shopping_item_id   INT NOT NULL REFERENCES shopping_items(id) ON DELETE CASCADE,
    household_id       INT REFERENCES households(household_id) ON DELETE CASCADE,
    user_id            INT REFERENCES users(id) ON DELETE CASCADE,
    portions_remaining NUMERIC(10,2) NOT NULL DEFAULT 0,
    expires_on         DATE,
    status             TEXT NOT NULL DEFAULT 'fresh'
                           CHECK (status IN ('fresh', 'expiring_soon', 'expired')),
    bought_at          TIMESTAMP NOT NULL DEFAULT now(),
    updated_at         TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_pantry_item_household
    ON pantry (shopping_item_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_pantry_item_user
    ON pantry (shopping_item_id, user_id) WHERE user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_pantry_expires_on ON pantry(expires_on) WHERE expires_on IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_pantry_status      ON pantry(status);
