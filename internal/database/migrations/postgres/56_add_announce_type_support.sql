ALTER TABLE "release"
    ADD COLUMN announce_type TEXT DEFAULT 'NEW';

ALTER TABLE filter
    ADD COLUMN announce_types TEXT[] DEFAULT '{}';