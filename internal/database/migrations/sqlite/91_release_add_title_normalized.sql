ALTER TABLE "release"
    ADD COLUMN title_normalized TEXT;

CREATE INDEX release_title_normalized_index
    ON "release" (title_normalized);