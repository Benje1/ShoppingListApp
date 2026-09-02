-- =============================================================================
-- Ingredient purchasing & per-portion recipe control
--   * Shopping items describe how they are sold: a base unit (g/ml/tin/unit),
--     the amount in one pack, and whether the item can also be bought loose.
--   * Meal ingredients can specify a precise per-portion amount (e.g. 100 g of
--     pasta per person) and per-meal-card dietary variants (e.g. use GF pasta).
-- =============================================================================

-- ── How a shopping item is sold ──────────────────────────────────────────────
-- base_unit is the unit that pack_size and meal per-portion amounts are measured
-- in: 'g', 'ml', 'unit', 'tin', 'slice', etc. NULL means a plain countable unit.
ALTER TABLE shopping_items ADD COLUMN IF NOT EXISTS base_unit  TEXT;
-- pack_size is how many base units come in one standard pack (500 for a 500 g
-- bag of pasta, 4 for a 4-pack of baking potatoes, 4 for a 4-tin sweetcorn
-- multipack). Used to convert a recipe's required amount into packs to buy.
ALTER TABLE shopping_items ADD COLUMN IF NOT EXISTS pack_size  NUMERIC(10, 2) NOT NULL DEFAULT 1;
-- sold_loose marks items that can also be bought as single base units outside a
-- pack (loose potatoes, individual sweetcorn tins), so the shopping list can
-- offer buying the exact amount instead of rounding up to whole packs.
ALTER TABLE shopping_items ADD COLUMN IF NOT EXISTS sold_loose BOOLEAN NOT NULL DEFAULT false;

-- ── Per-portion recipe control on a meal card ────────────────────────────────
-- quantity_per_portion is the amount of the item's base_unit needed for ONE
-- portion (100 for 100 g of pasta per person, 1 for one baked potato per
-- person). NULL keeps the legacy behaviour where `quantity` is the whole-batch
-- amount for the meal's default_portions.
ALTER TABLE meal_ingredients ADD COLUMN IF NOT EXISTS quantity_per_portion NUMERIC(10, 2);
-- dietary_tags captures per-meal-card variant requirements for this ingredient
-- (e.g. this meal uses the gluten-free pasta). Validated in application code.
ALTER TABLE meal_ingredients ADD COLUMN IF NOT EXISTS dietary_tags TEXT[] NOT NULL DEFAULT '{}';
