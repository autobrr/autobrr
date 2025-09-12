ALTER TABLE release
    RENAME COLUMN "group" TO "release_group";

ALTER TABLE release
    ALTER COLUMN size TYPE BIGINT USING size::BIGINT;