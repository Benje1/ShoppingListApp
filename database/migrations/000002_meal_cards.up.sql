-- =============================================================================
-- Meal-card refactor
--   * Richer meals: photo, recipe, structured allergens
--   * Optional ingredients
--   * Login "did you make these?" review: last-seen tracking + a cook log
-- =============================================================================

-- ── Richer meals ────────────────────────────────────────────────────────────
ALTER TABLE meals ADD COLUMN IF NOT EXISTS photo_url TEXT;
ALTER TABLE meals ADD COLUMN IF NOT EXISTS recipe    TEXT;
-- Structured allergen tags, validated in application code against a fixed set.
ALTER TABLE meals ADD COLUMN IF NOT EXISTS allergens TEXT[] NOT NULL DEFAULT '{}';

-- ── Optional ingredients ────────────────────────────────────────────────────
ALTER TABLE meal_ingredients ADD COLUMN IF NOT EXISTS optional BOOLEAN NOT NULL DEFAULT false;

-- ── Last-seen tracking for the login cook-review window ──────────────────────
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_seen_at     TIMESTAMP;
-- previous_seen_at is snapshotted from last_seen_at on each login; it anchors
-- the start of the "days since your last visit" cook-review window.
ALTER TABLE users ADD COLUMN IF NOT EXISTS previous_seen_at TIMESTAMP;

-- ── Cook log ────────────────────────────────────────────────────────────────
-- Records the user's answer to "did you make <meal> on <date>?" so the same
-- prompt is never shown twice. One row per (scope, date, meal).
CREATE TABLE IF NOT EXISTS meal_cook_log (
    id           SERIAL PRIMARY KEY,
    meal_id      INT  NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    cook_date    DATE NOT NULL,
    made         BOOLEAN NOT NULL,
    household_id INT REFERENCES households(household_id) ON DELETE CASCADE,
    user_id      INT REFERENCES users(id) ON DELETE CASCADE,
    answered_by  INT REFERENCES users(id) ON DELETE SET NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_cook_log_household
    ON meal_cook_log (cook_date, meal_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_cook_log_user
    ON meal_cook_log (cook_date, meal_id, user_id) WHERE user_id IS NOT NULL;
