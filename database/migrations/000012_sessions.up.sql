CREATE TABLE sessions (
    id           TEXT        PRIMARY KEY,
    user_id      INT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    username     TEXT        NOT NULL,
    household_ids INT[]       NOT NULL DEFAULT '{}',
    expires_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
