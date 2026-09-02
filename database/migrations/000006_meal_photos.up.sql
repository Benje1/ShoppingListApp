-- =============================================================================
-- Meal photo storage
--   Uploaded meal photos are stored content-addressed (SHA-256 of the bytes) in
--   Postgres, so they survive redeploys on hosts with an ephemeral filesystem
--   (e.g. Render) without needing separate object storage or secrets. The API
--   serves them back as a hosted URL, which the web app stores in meals.photo_url.
--   Swapping to a CDN later only changes how upload/serve work, not the API.
-- =============================================================================

CREATE TABLE IF NOT EXISTS meal_photos (
    -- Lowercase hex SHA-256 of the (normalised) image bytes; also the public,
    -- unguessable handle used in the served URL.
    id           TEXT PRIMARY KEY,
    content_type TEXT NOT NULL,
    bytes        BYTEA NOT NULL,
    byte_size    INT   NOT NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT now()
);
