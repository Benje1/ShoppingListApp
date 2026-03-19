DO $$ BEGIN
    CREATE TYPE shopping_item_type AS ENUM (
        'fruit', 'vegetable', 'dairy', 'meat', 'meat_free', 'seafood', 
        'bakery', 'pantry', 'snacks', 'frozen', 'drinks', 'cleaning', 
        'toiletries', 'baby', 'health', 'household', 'spices', 'condiments'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS households (
    household_id SERIAL PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT now()
);

CREATE TABLE IF NOT EXISTS household_members (
    household_id INT REFERENCES households(household_id) ON DELETE CASCADE,
    user_id INT REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (household_id, user_id)
);

CREATE TABLE IF NOT EXISTS shopping_items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    item_type shopping_item_type NOT NULL,
    text_id TEXT UNIQUE
);

CREATE TABLE IF NOT EXISTS shopping_list (
    id SERIAL PRIMARY KEY,
    shopping_item_id INT REFERENCES shopping_items(id) NOT NULL,
    quantity INT NOT NULL DEFAULT 1,
    household_id INT REFERENCES households(household_id),
    user_id INT REFERENCES users(id),
    CHECK (
        (household_id IS NOT NULL AND user_id IS NULL)
        OR
        (household_id IS NULL AND user_id IS NOT NULL)
    )
);
