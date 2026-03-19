-- Add optional display name to households
ALTER TABLE households
    ADD COLUMN IF NOT EXISTS name TEXT;

-- Invite requests: a user submits a code to request joining a household.
-- A household member can then approve or deny the request.
CREATE TABLE IF NOT EXISTS household_invites (
    id                  SERIAL PRIMARY KEY,
    household_id        INT  NOT NULL REFERENCES households(household_id) ON DELETE CASCADE,
    invite_code         TEXT NOT NULL UNIQUE,
    requested_by_user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'pending'
                            CHECK (status IN ('pending', 'approved', 'denied')),
    created_at          TIMESTAMP NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_household_invites_code       ON household_invites(invite_code);
CREATE INDEX IF NOT EXISTS idx_household_invites_household  ON household_invites(household_id);
