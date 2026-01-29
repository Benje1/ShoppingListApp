CREATE TYPE shopping_item_type AS ENUM (
    'fruit',
    'vegetable',
    'dairy',
    'meat',
    'meat_free',
    'seafood',
    'bakery',
    'pantry',
    'snacks',
    'frozen',
    'drinks',
    'cleaning',
    'toiletries',
    'baby',
    'health',
    'household',
    'spices',
    'condiments'
);

-- Future, add uniue text ids
CREATE TABLE shopping_items (
    id INT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    item_type shopping_item_type  NOT NULL
);

CREATE TABLE shopping_list (
    shopping_item_id int REFERENCES shopping_items(id),
    quantity INT
);

CREATE TABLE households (
    household_id INt GENERATED ALWAYS AS IDENTITY PRIMARY KEY
);

CREATE TABLE users (
    id GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT,
    household INT REFERENCES households(household_id),
    username TEXT,
    password TEXT
);