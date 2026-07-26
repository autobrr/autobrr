ALTER TABLE "release"
    RENAME COLUMN bitrate TO quality;

ALTER TABLE "filter"
    ADD COLUMN artists TEXT;

ALTER TABLE "filter"
    ADD COLUMN albums TEXT;

ALTER TABLE "filter"
    ADD COLUMN release_types_match TEXT [] DEFAULT '{}';

ALTER TABLE "filter"
    ADD COLUMN release_types_ignore TEXT [] DEFAULT '{}';

ALTER TABLE "filter"
    ADD COLUMN formats TEXT [] DEFAULT '{}';

ALTER TABLE "filter"
    ADD COLUMN quality TEXT [] DEFAULT '{}';

ALTER TABLE "filter"
    ADD COLUMN log_score INTEGER;

ALTER TABLE "filter"
    ADD COLUMN has_log BOOLEAN;

ALTER TABLE "filter"
    ADD COLUMN has_cue BOOLEAN;

ALTER TABLE "filter"
    ADD COLUMN perfect_flac BOOLEAN;
