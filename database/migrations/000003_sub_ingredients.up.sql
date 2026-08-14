-- =============================================================================
-- Sub-ingredients & ingredient allergens
--   * Each shopping item can carry structured allergen tags of its own.
--   * Each shopping item can be broken down into lightweight sub-ingredients
--     (e.g. a stock cube → salt, yeast extract, celery), each of which can
--     itself be flagged for allergens. Sub-ingredients are name + allergens
--     only: no quantity, unit, pantry tracking or purchasing. One level deep.
-- =============================================================================

-- ── Allergen tags on the shopping item itself ────────────────────────────────
-- Validated in application code against the same fixed EU-14 set as meals.
ALTER TABLE shopping_items ADD COLUMN IF NOT EXISTS allergens TEXT[] NOT NULL DEFAULT '{}';

-- ── Sub-ingredients ──────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS sub_ingredients (
    id               SERIAL PRIMARY KEY,
    shopping_item_id INT  NOT NULL REFERENCES shopping_items(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    allergens        TEXT[] NOT NULL DEFAULT '{}',
    sort_order       INT  NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_sub_ingredients_item
    ON sub_ingredients (shopping_item_id);
