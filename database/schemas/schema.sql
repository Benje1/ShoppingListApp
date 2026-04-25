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
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    username      TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMP DEFAULT now()
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
    shelf_life_days   INT
);

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
    season           season NULL
);

CREATE TABLE meal_ingredients (
    meal_id          INT REFERENCES meals(id) ON DELETE CASCADE,
    shopping_item_id INT REFERENCES shopping_items(id) ON DELETE RESTRICT,
    quantity         NUMERIC(10, 2) NOT NULL DEFAULT 1,
    unit             TEXT,
    PRIMARY KEY (meal_id, shopping_item_id)
);

CREATE TABLE meal_cooks (
    meal_id INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (meal_id, user_id)
);

CREATE TABLE meal_components (
    parent_meal_id INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    sub_meal_id    INT NOT NULL REFERENCES meals(id) ON DELETE RESTRICT,
    sort_order     INT NOT NULL DEFAULT 0,
    PRIMARY KEY (parent_meal_id, sub_meal_id),
    CHECK (parent_meal_id <> sub_meal_id)
);

CREATE INDEX idx_meal_components_sub ON meal_components(sub_meal_id);

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