ALTER TABLE release
    RENAME COLUMN release_group TO "group";

ALTER TABLE release
    DROP COLUMN raw;

ALTER TABLE release
    DROP COLUMN audio;

ALTER TABLE release
    DROP COLUMN region;

ALTER TABLE release
    DROP COLUMN language;

ALTER TABLE release
    DROP COLUMN edition;

ALTER TABLE release
    DROP COLUMN unrated;

ALTER TABLE release
    DROP COLUMN hybrid;

ALTER TABLE release
    DROP COLUMN artists;

ALTER TABLE release
    DROP COLUMN format;

ALTER TABLE release
    DROP COLUMN quality;

ALTER TABLE release
    DROP COLUMN log_score;

ALTER TABLE release
    DROP COLUMN has_log;

ALTER TABLE release
    DROP COLUMN has_cue;

ALTER TABLE release
    DROP COLUMN is_scene;

ALTER TABLE release
    DROP COLUMN freeleech;

ALTER TABLE release
    DROP COLUMN freeleech_percent;

ALTER TABLE "filter"
    ADD COLUMN origins TEXT []   DEFAULT '{}';