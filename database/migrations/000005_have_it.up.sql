-- Tracks which items a user or household already has / has bought.
-- Separate from the shopping list so ticking "have it" doesn't remove
-- the item from the list — it stays as a record.
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
