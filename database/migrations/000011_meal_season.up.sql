CREATE TYPE season AS ENUM ('spring', 'summer', 'autumn', 'winter');

ALTER TABLE meals
    ADD COLUMN IF NOT EXISTS season season NULL;
