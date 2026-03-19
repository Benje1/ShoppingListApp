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

CREATE TABLE shopping_items (
    id                SERIAL PRIMARY KEY,
    name              TEXT NOT NULL,
    item_type         shopping_item_type NOT NULL,
    text_id           TEXT UNIQUE,
    portions_per_unit INT NOT NULL DEFAULT 1
);

CREATE TABLE shopping_list (
    id               SERIAL PRIMARY KEY,
    shopping_item_id INT REFERENCES shopping_items(id) NOT NULL,
    quantity         INT NOT NULL DEFAULT 1,
    household_id     INT REFERENCES households(household_id),
    user_id          INT REFERENCES users(id),
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
