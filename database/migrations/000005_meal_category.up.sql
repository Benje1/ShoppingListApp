-- =============================================================================
-- Meal category
--   A category is the "slot type" a meal fills when planning the week (e.g. a
--   "soup night" can be filled by any meal in the `soup` category). It is a
--   short lowercase string validated in application code against a fixed set;
--   '' means uncategorised.
-- =============================================================================

ALTER TABLE meals ADD COLUMN IF NOT EXISTS category TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_meals_category ON meals (category) WHERE category <> '';
