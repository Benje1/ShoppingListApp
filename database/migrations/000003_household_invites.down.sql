DROP INDEX IF EXISTS idx_household_invites_household;
DROP INDEX IF EXISTS idx_household_invites_code;
DROP TABLE IF EXISTS household_invites;
ALTER TABLE households DROP COLUMN IF EXISTS name;
