-- Consolidated schema (migrations 001–011)
-- Drop and recreate cleanly for a fresh database.

CREATE TYPE shopping_item_type AS ENUM (
    'fruit', 'vegetable', 'dairy', 'meat', 'meat_free', 'seafood',
    'bakery', 'pantry', 'snacks', 'frozen', 'drinks', 'cleaning',
    'toiletries', 'baby', 'health', 'household', 'spices', 'condiments'
);

CREATE TYPE season AS ENUM ('spring', 'summer', 'autumn', 'winter');

CREATE TABLE households (
    household_id SERIAL PRIMARY KEY,
    num_people   INT  NOT NULL DEFAULT 1,
    name         TEXT
);

CREATE TABLE users (
    id               SERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    username         TEXT NOT NULL UNIQUE,
    password_hash    TEXT NOT NULL,
    created_at       TIMESTAMP DEFAULT now(),
    last_seen_at     TIMESTAMP,
    previous_seen_at TIMESTAMP
);

CREATE TABLE household_members (
    household_id INT REFERENCES households(household_id) ON DELETE CASCADE,
    user_id      INT REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (household_id, user_id)
);

CREATE TABLE shopping_items (
    id                SERIAL PRIMARY KEY,
    name              TEXT NOT NULL,
    item_type         shopping_item_type NOT NULL,
    text_id           TEXT UNIQUE,
    portions_per_unit INT NOT NULL DEFAULT 1,
    shelf_life_days   INT,
    -- Structured allergen tags, validated in application code against a fixed set.
    allergens         TEXT[] NOT NULL DEFAULT '{}',
    -- How the item is sold. base_unit is the measurement unit ('g','ml','tin',
    -- 'unit', ...); pack_size is how many base units are in one pack; sold_loose
    -- marks items that can also be bought as single base units.
    base_unit         TEXT,
    pack_size         NUMERIC(10, 2) NOT NULL DEFAULT 1,
    sold_loose        BOOLEAN NOT NULL DEFAULT false
);

-- Lightweight breakdown of a shopping item into its constituent parts
-- (e.g. a stock cube → salt, yeast extract, celery). Name + allergens only;
-- no quantity, unit, pantry tracking or purchasing. One level deep.
CREATE TABLE sub_ingredients (
    id               SERIAL PRIMARY KEY,
    shopping_item_id INT  NOT NULL REFERENCES shopping_items(id) ON DELETE CASCADE,
    name             TEXT NOT NULL,
    allergens        TEXT[] NOT NULL DEFAULT '{}',
    sort_order       INT  NOT NULL DEFAULT 0
);

CREATE INDEX idx_sub_ingredients_item ON sub_ingredients (shopping_item_id);

CREATE TABLE shopping_list (
    id               SERIAL PRIMARY KEY,
    shopping_item_id INT NOT NULL REFERENCES shopping_items(id),
    quantity         INT NOT NULL DEFAULT 1,
    household_id     INT REFERENCES households(household_id),
    user_id          INT REFERENCES users(id),
    updated_at       TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_shopping_list_item_household
    ON shopping_list (shopping_item_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_shopping_list_item_user
    ON shopping_list (shopping_item_id, user_id) WHERE user_id IS NOT NULL;

CREATE TABLE shopping_list_have_it (
    id               SERIAL PRIMARY KEY,
    shopping_item_id INT NOT NULL REFERENCES shopping_items(id) ON DELETE CASCADE,
    household_id     INT REFERENCES households(household_id) ON DELETE CASCADE,
    user_id          INT REFERENCES users(id) ON DELETE CASCADE,
    updated_at       TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_have_it_item_household
    ON shopping_list_have_it (shopping_item_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_have_it_item_user
    ON shopping_list_have_it (shopping_item_id, user_id) WHERE user_id IS NOT NULL;

CREATE TABLE household_invites (
    id                   SERIAL PRIMARY KEY,
    household_id         INT  NOT NULL REFERENCES households(household_id) ON DELETE CASCADE,
    invite_code          TEXT NOT NULL UNIQUE,
    requested_by_user_id INT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status               TEXT NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending', 'approved', 'denied')),
    created_at           TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX idx_household_invites_code      ON household_invites(invite_code);
CREATE INDEX idx_household_invites_household ON household_invites(household_id);

CREATE TABLE meals (
    id               SERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    description      TEXT,
    default_portions INT NOT NULL DEFAULT 2,
    season           season NULL,
    photo_url        TEXT,
    recipe           TEXT,
    -- Structured allergen tags, validated in application code against a fixed set.
    allergens        TEXT[] NOT NULL DEFAULT '{}',
    -- Planning "slot type" (soup, salad, ...): meals in the same category are
    -- interchangeable for a day. Validated in application code; '' = uncategorised.
    category         TEXT NOT NULL DEFAULT '',
    -- NULL = global/shared meal; non-NULL = only visible within this household
    household_id     INT REFERENCES households(household_id) ON DELETE CASCADE
);

CREATE INDEX idx_meals_household ON meals (household_id) WHERE household_id IS NOT NULL;
CREATE INDEX idx_meals_category  ON meals (category) WHERE category <> '';

CREATE TABLE meal_ingredients (
    meal_id          INT REFERENCES meals(id) ON DELETE CASCADE,
    shopping_item_id INT REFERENCES shopping_items(id) ON DELETE RESTRICT,
    quantity         NUMERIC(10, 2) NOT NULL DEFAULT 1,
    unit             TEXT,
    optional         BOOLEAN NOT NULL DEFAULT false,
    -- Precise per-portion amount in the item's base_unit (e.g. 100 g pasta per
    -- person). NULL falls back to `quantity` as a whole-batch amount.
    quantity_per_portion NUMERIC(10, 2),
    -- Per-meal-card dietary variant requirements for this ingredient
    -- (e.g. use the gluten-free pasta). Validated in application code.
    dietary_tags     TEXT[] NOT NULL DEFAULT '{}',
    PRIMARY KEY (meal_id, shopping_item_id)
);

CREATE TABLE meal_cooks (
    id           SERIAL PRIMARY KEY,
    meal_id      INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id      INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- NULL = assignment applies across all households the cook belongs to
    -- non-NULL = assignment scoped to this specific household only
    household_id INT REFERENCES households(household_id) ON DELETE CASCADE
);

-- Two partial unique indexes because NULL != NULL in standard SQL
CREATE UNIQUE INDEX idx_meal_cooks_meal_user_household
    ON meal_cooks (meal_id, user_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_meal_cooks_meal_user_no_household
    ON meal_cooks (meal_id, user_id) WHERE household_id IS NULL;

CREATE TABLE meal_components (
    parent_meal_id INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    sub_meal_id    INT NOT NULL REFERENCES meals(id) ON DELETE RESTRICT,
    sort_order     INT NOT NULL DEFAULT 0,
    PRIMARY KEY (parent_meal_id, sub_meal_id),
    CHECK (parent_meal_id <> sub_meal_id)
);

CREATE INDEX idx_meal_components_sub ON meal_components(sub_meal_id);

CREATE TABLE meal_option_group_entries (
    id               SERIAL PRIMARY KEY,
    meal_id          INT  NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    option_group     TEXT NOT NULL,
    option_type      TEXT NOT NULL CHECK (option_type IN ('one_of', 'many_of')),
    sort_order       INT  NOT NULL DEFAULT 0,
    shopping_item_id INT  REFERENCES shopping_items(id) ON DELETE CASCADE,
    sub_meal_id      INT  REFERENCES meals(id)          ON DELETE CASCADE,
    CHECK (
        (shopping_item_id IS NOT NULL AND sub_meal_id IS NULL) OR
        (shopping_item_id IS NULL     AND sub_meal_id IS NOT NULL)
    )
);

CREATE INDEX idx_moge_meal     ON meal_option_group_entries(meal_id);
CREATE INDEX idx_moge_sub_meal ON meal_option_group_entries(sub_meal_id);
CREATE INDEX idx_moge_item     ON meal_option_group_entries(shopping_item_id);

-- meal_plan uses two partial unique indexes per scope (household / user),
-- each including week_start so multiple weeks can coexist (migration 010).
CREATE TABLE meal_plan (
    id                     SERIAL PRIMARY KEY,
    day_name               TEXT NOT NULL,
    week_start             DATE NOT NULL DEFAULT (DATE_TRUNC('week', CURRENT_DATE)::DATE),
    meal_name              TEXT,
    household_id           INT REFERENCES households(household_id) ON DELETE CASCADE,
    user_id                INT REFERENCES users(id) ON DELETE CASCADE,
    updated_at             TIMESTAMP NOT NULL DEFAULT now(),
    meal_id                INT REFERENCES meals(id) ON DELETE SET NULL,
    cook_user_id           INT REFERENCES users(id) ON DELETE SET NULL,
    repeating_cook_user_id INT REFERENCES users(id) ON DELETE SET NULL,
    temp_cook_user_id      INT REFERENCES users(id) ON DELETE SET NULL,
    repeating_meal_id      INT REFERENCES meals(id) ON DELETE SET NULL,
    temp_meal_id           INT REFERENCES meals(id) ON DELETE SET NULL,
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX idx_meal_plan_day_week_household
    ON meal_plan (day_name, week_start, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_meal_plan_day_week_user
    ON meal_plan (day_name, week_start, user_id) WHERE user_id IS NOT NULL;

COMMENT ON COLUMN meal_plan.week_start IS
    'Monday of the ISO week this plan row belongs to';
COMMENT ON COLUMN meal_plan.repeating_cook_user_id IS
    'Person who cooks on this weekday every week (standing assignment)';
COMMENT ON COLUMN meal_plan.temp_cook_user_id IS
    'One-off cook override for the next occurrence; cleared after the week rolls over';
COMMENT ON COLUMN meal_plan.repeating_meal_id IS
    'Meal served on this weekday every week (standing assignment)';
COMMENT ON COLUMN meal_plan.temp_meal_id IS
    'One-off meal override for the next occurrence; cleared after the week rolls over';

CREATE TABLE pantry (
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

CREATE UNIQUE INDEX idx_pantry_item_household
    ON pantry (shopping_item_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_pantry_item_user
    ON pantry (shopping_item_id, user_id) WHERE user_id IS NOT NULL;

CREATE INDEX idx_pantry_expires_on ON pantry(expires_on) WHERE expires_on IS NOT NULL;
CREATE INDEX idx_pantry_status     ON pantry(status);

-- meal_cook_log records the answer to "did you make <meal> on <date>?" so the
-- login cook-review never re-prompts. One row per (scope, date, meal).
CREATE TABLE meal_cook_log (
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

CREATE UNIQUE INDEX idx_meal_cook_log_household
    ON meal_cook_log (cook_date, meal_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX idx_meal_cook_log_user
    ON meal_cook_log (cook_date, meal_id, user_id) WHERE user_id IS NOT NULL;

-- meal_photos stores uploaded meal images content-addressed by SHA-256, so they
-- persist across redeploys on hosts with an ephemeral filesystem. The API serves
-- them back as a hosted URL stored in meals.photo_url.
CREATE TABLE meal_photos (
    id           TEXT PRIMARY KEY,
    content_type TEXT NOT NULL,
    bytes        BYTEA NOT NULL,
    byte_size    INT   NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT now()
);