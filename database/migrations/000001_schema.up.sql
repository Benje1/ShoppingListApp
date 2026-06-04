-- =============================================================================
-- Full schema (collapsed from migrations 001–016)
-- =============================================================================

-- ── Types ─────────────────────────────────────────────────────────────────────

DO $$ BEGIN
    CREATE TYPE shopping_item_type AS ENUM (
        'fruit', 'vegetable', 'dairy', 'meat', 'meat_free', 'seafood',
        'bakery', 'pantry', 'snacks', 'frozen', 'drinks', 'cleaning',
        'toiletries', 'baby', 'health', 'household', 'spices', 'condiments'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE season AS ENUM ('spring', 'summer', 'autumn', 'winter');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- ── Core tables ───────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS households (
    household_id SERIAL PRIMARY KEY,
    num_people   INT  NOT NULL DEFAULT 1,
    name         TEXT
);

CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS household_members (
    household_id INT REFERENCES households(household_id) ON DELETE CASCADE,
    user_id      INT REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (household_id, user_id)
);

-- ── Sessions ──────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS sessions (
    id            TEXT        PRIMARY KEY,
    user_id       INT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username      TEXT        NOT NULL,
    household_ids INT[]       NOT NULL DEFAULT '{}',
    expires_at    TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);

-- ── Household invites ─────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS household_invites (
    id                   SERIAL PRIMARY KEY,
    household_id         INT  NOT NULL REFERENCES households(household_id) ON DELETE CASCADE,
    invite_code          TEXT NOT NULL UNIQUE,
    requested_by_user_id INT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status               TEXT NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending', 'approved', 'denied')),
    created_at           TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_household_invites_code      ON household_invites(invite_code);
CREATE INDEX IF NOT EXISTS idx_household_invites_household ON household_invites(household_id);

-- ── Shopping items ────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS shopping_items (
    id               SERIAL PRIMARY KEY,
    name             TEXT                NOT NULL,
    item_type        shopping_item_type  NOT NULL,
    text_id          TEXT                UNIQUE,
    portions_per_unit INT               NOT NULL DEFAULT 1,
    shelf_life_days  INT
);

-- ── Shopping list ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS shopping_list (
    id               SERIAL PRIMARY KEY,
    shopping_item_id INT  NOT NULL REFERENCES shopping_items(id),
    quantity         INT  NOT NULL DEFAULT 1,
    household_id     INT  REFERENCES households(household_id),
    user_id          INT  REFERENCES users(id),
    updated_at       TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_shopping_list_item_household
    ON shopping_list (shopping_item_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_shopping_list_item_user
    ON shopping_list (shopping_item_id, user_id) WHERE user_id IS NOT NULL;

-- ── Have-it tracking ──────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS shopping_list_have_it (
    id               SERIAL PRIMARY KEY,
    shopping_item_id INT  NOT NULL REFERENCES shopping_items(id) ON DELETE CASCADE,
    household_id     INT  REFERENCES households(household_id) ON DELETE CASCADE,
    user_id          INT  REFERENCES users(id) ON DELETE CASCADE,
    updated_at       TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_have_it_item_household
    ON shopping_list_have_it (shopping_item_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_have_it_item_user
    ON shopping_list_have_it (shopping_item_id, user_id) WHERE user_id IS NOT NULL;

-- ── Meals ─────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS meals (
    id               SERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    description      TEXT,
    default_portions INT  NOT NULL DEFAULT 2,
    season           season NULL,
    household_id     INT  REFERENCES households(household_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_meals_household
    ON meals (household_id) WHERE household_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS meal_ingredients (
    meal_id          INT  REFERENCES meals(id) ON DELETE CASCADE,
    shopping_item_id INT  REFERENCES shopping_items(id) ON DELETE RESTRICT,
    quantity         NUMERIC(10, 2) NOT NULL DEFAULT 1,
    unit             TEXT,
    PRIMARY KEY (meal_id, shopping_item_id)
);

-- ── Meal option groups ────────────────────────────────────────────────────────

-- Each row is one choice within an option group for a meal.
-- Exactly one of shopping_item_id or sub_meal_id must be set.
CREATE TABLE IF NOT EXISTS meal_option_group_entries (
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

CREATE INDEX IF NOT EXISTS idx_moge_meal     ON meal_option_group_entries(meal_id);
CREATE INDEX IF NOT EXISTS idx_moge_sub_meal ON meal_option_group_entries(sub_meal_id);
CREATE INDEX IF NOT EXISTS idx_moge_item     ON meal_option_group_entries(shopping_item_id);

-- ── Meal components (composite meals) ────────────────────────────────────────

CREATE TABLE IF NOT EXISTS meal_components (
    parent_meal_id INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    sub_meal_id    INT NOT NULL REFERENCES meals(id) ON DELETE RESTRICT,
    sort_order     INT NOT NULL DEFAULT 0,
    PRIMARY KEY (parent_meal_id, sub_meal_id),
    CHECK (parent_meal_id <> sub_meal_id)
);

CREATE INDEX IF NOT EXISTS idx_meal_components_sub ON meal_components(sub_meal_id);

-- ── Meal cooks ────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS meal_cooks (
    id           SERIAL PRIMARY KEY,
    meal_id      INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id      INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    household_id INT REFERENCES households(household_id) ON DELETE CASCADE
);

-- NULL household_id means the cook assignment applies in all households.
CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_cooks_meal_user_household
    ON meal_cooks (meal_id, user_id, household_id) WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_cooks_meal_user_no_household
    ON meal_cooks (meal_id, user_id) WHERE household_id IS NULL;

-- ── Meal plan ─────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS meal_plan (
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

-- Partial unique indexes correctly handle NULLs in the non-indexed column.
CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_plan_day_week_household
    ON meal_plan (day_name, week_start, household_id)
    WHERE household_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_meal_plan_day_week_user
    ON meal_plan (day_name, week_start, user_id)
    WHERE user_id IS NOT NULL;

-- ── Pantry ────────────────────────────────────────────────────────────────────

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
