CREATE TYPE shopping_item_type AS ENUM (
    'fruit', 'vegetable', 'dairy', 'meat', 'meat_free', 'seafood',
    'bakery', 'pantry', 'snacks', 'frozen', 'drinks', 'cleaning',
    'toiletries', 'baby', 'health', 'household', 'spices', 'condiments'
);

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

-- shopping_items: portions_per_unit added in migration 002,
--                shelf_life_days added in migration 007
CREATE TABLE shopping_items (
    id                SERIAL PRIMARY KEY,
    name              TEXT NOT NULL,
    item_type         shopping_item_type NOT NULL,
    text_id           TEXT UNIQUE,
    portions_per_unit INT NOT NULL DEFAULT 1,
    shelf_life_days   INT
);

-- shopping_list: updated_at added in migration 004
CREATE TABLE shopping_list (
    id               SERIAL PRIMARY KEY,
    shopping_item_id INT REFERENCES shopping_items(id) NOT NULL,
    quantity         INT NOT NULL DEFAULT 1,
    household_id     INT REFERENCES households(household_id),
    user_id          INT REFERENCES users(id),
    updated_at       TIMESTAMP NOT NULL DEFAULT now(),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL) OR
        (household_id IS NULL     AND user_id IS NOT NULL)
    )
);

CREATE TABLE meals (
    id               SERIAL PRIMARY KEY,
    name             TEXT NOT NULL,
    description      TEXT,
    default_portions INT NOT NULL DEFAULT 2
);

CREATE TABLE meal_ingredients (
    meal_id          INT REFERENCES meals(id) ON DELETE CASCADE,
    shopping_item_id INT REFERENCES shopping_items(id) ON DELETE RESTRICT,
    quantity         NUMERIC(10,2) NOT NULL DEFAULT 1,
    unit             TEXT,
    PRIMARY KEY (meal_id, shopping_item_id)
);

CREATE TABLE household_invites (
    id                   SERIAL PRIMARY KEY,
    household_id         INT  NOT NULL REFERENCES households(household_id) ON DELETE CASCADE,
    invite_code          TEXT NOT NULL UNIQUE,
    requested_by_user_id INT  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status               TEXT NOT NULL DEFAULT 'pending'
                             CHECK (status IN ('pending', 'approved', 'denied')),
    created_at           TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE meal_cooks (
    meal_id INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (meal_id, user_id)
);

-- meal_plan: added in migration 004,
--            meal_id/cook_user_id added in migration 006,
--            repeating_*/temp_* columns added in migration 009
CREATE TABLE meal_plan (
    id                     SERIAL PRIMARY KEY,
    day_name               TEXT NOT NULL,
    meal_name              TEXT DEFAULT NULL,
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

-- shopping_list_have_it: added in migration 005
CREATE TABLE shopping_list_have_it (
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

-- meal_components: added in migration 007
CREATE TABLE meal_components (
    parent_meal_id INT NOT NULL REFERENCES meals(id) ON DELETE CASCADE,
    sub_meal_id    INT NOT NULL REFERENCES meals(id) ON DELETE RESTRICT,
    sort_order     INT NOT NULL DEFAULT 0,
    PRIMARY KEY (parent_meal_id, sub_meal_id),
    CHECK (parent_meal_id <> sub_meal_id)
);

-- pantry: added in migration 007
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
