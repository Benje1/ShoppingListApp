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

-- Future, add uniue id
CREATE TABLE shopping_items (
    name TEXT NOT NULL,
    item_type shopping_item_type  NOT NULL,
    PRIMARY KEY (name, item_type)
)