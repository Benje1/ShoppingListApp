-- Tear down the full schema in reverse dependency order.

DROP TABLE IF EXISTS pantry;
DROP TABLE IF EXISTS meal_plan;
DROP TABLE IF EXISTS meal_cooks;
DROP TABLE IF EXISTS meal_components;
DROP TABLE IF EXISTS meal_option_group_entries;
DROP TABLE IF EXISTS meal_ingredients;
DROP TABLE IF EXISTS meals;
DROP TABLE IF EXISTS shopping_list_have_it;
DROP TABLE IF EXISTS shopping_list;
DROP TABLE IF EXISTS shopping_items;
DROP TABLE IF EXISTS household_invites;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS household_members;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS households;

DROP TYPE IF EXISTS season;
DROP TYPE IF EXISTS shopping_item_type;
