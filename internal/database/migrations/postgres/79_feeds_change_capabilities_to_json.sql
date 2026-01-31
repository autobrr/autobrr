ALTER TABLE feed
    ALTER COLUMN capabilities type json using capabilities::json;

ALTER TABLE feed
    ALTER COLUMN capabilities DROP NOT NULL;

ALTER TABLE feed
    ALTER COLUMN capabilities DROP DEFAULT;
