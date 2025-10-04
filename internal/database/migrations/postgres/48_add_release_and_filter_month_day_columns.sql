ALTER TABLE "release"
    ADD COLUMN month INTEGER;

ALTER TABLE "release"
    ADD COLUMN day INTEGER;

ALTER TABLE filter
    ADD COLUMN months TEXT;

ALTER TABLE filter
    ADD COLUMN days TEXT;
