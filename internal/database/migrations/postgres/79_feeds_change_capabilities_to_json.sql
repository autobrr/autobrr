ALTER TABLE feed
    DROP COLUMN IF EXISTS capabilities,
    ADD COLUMN capabilities json NOT NULL DEFAULT '{}'::json;

-- check if categories is needed. It as removed in some early version
ALTER TABLE feed
    ADD COLUMN IF NOT EXISTS categories TEXT [] DEFAULT '{}' NOT NULL;
