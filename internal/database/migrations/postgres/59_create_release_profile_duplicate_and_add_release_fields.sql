CREATE TABLE release_profile_duplicate
(
    id            SERIAL PRIMARY KEY,
    name          TEXT NOT NULL,
    protocol      BOOLEAN DEFAULT FALSE,
    release_name  BOOLEAN DEFAULT FALSE,
    hash          BOOLEAN DEFAULT FALSE,
    title         BOOLEAN DEFAULT FALSE,
    sub_title     BOOLEAN DEFAULT FALSE,
    year          BOOLEAN DEFAULT FALSE,
    month         BOOLEAN DEFAULT FALSE,
    day           BOOLEAN DEFAULT FALSE,
    source        BOOLEAN DEFAULT FALSE,
    resolution    BOOLEAN DEFAULT FALSE,
    codec         BOOLEAN DEFAULT FALSE,
    container     BOOLEAN DEFAULT FALSE,
    dynamic_range BOOLEAN DEFAULT FALSE,
    audio         BOOLEAN DEFAULT FALSE,
    release_group BOOLEAN DEFAULT FALSE,
    season        BOOLEAN DEFAULT FALSE,
    episode       BOOLEAN DEFAULT FALSE,
    website       BOOLEAN DEFAULT FALSE,
    proper        BOOLEAN DEFAULT FALSE,
    repack        BOOLEAN DEFAULT FALSE,
    edition       BOOLEAN DEFAULT FALSE,
    language      BOOLEAN DEFAULT FALSE
);

INSERT INTO release_profile_duplicate (id, name, protocol, release_name, hash, title, sub_title, year, month, day,
                                       source, resolution, codec, container, dynamic_range, audio, release_group,
                                       season, episode, website, proper, repack, edition, language)
VALUES (1, 'Exact release', 'f', 't', 't', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f',
        'f', 'f', 'f', 'f'),
       (2, 'Movie', 'f', 'f', 'f', 't', 'f', 't', 'f', 'f', 'f', 't', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 'f',
        'f', 'f'),
       (3, 'TV', 'f', 'f', 'f', 't', 'f', 't', 't', 't', 'f', 'f', 'f', 'f', 'f', 'f', 'f', 't', 't', 'f', 'f', 'f',
        'f', 'f');

ALTER TABLE filter
    ADD release_profile_duplicate_id INTEGER;

ALTER TABLE filter
    ADD CONSTRAINT filter_release_profile_duplicate_id_fk
        FOREIGN KEY (release_profile_duplicate_id) REFERENCES release_profile_duplicate (id)
            ON DELETE SET NULL;

ALTER TABLE "release"
    ADD normalized_hash TEXT;

ALTER TABLE "release"
    ADD sub_title TEXT;

ALTER TABLE "release"
    ADD COLUMN IF NOT EXISTS audio TEXT;

ALTER TABLE "release"
    ADD audio_channels TEXT;

ALTER TABLE "release"
    ADD IF NOT EXISTS language TEXT;

ALTER TABLE "release"
    ADD media_processing TEXT;

ALTER TABLE "release"
    ADD IF NOT EXISTS edition TEXT;

ALTER TABLE "release"
    ADD IF NOT EXISTS cut TEXT;

ALTER TABLE "release"
    ADD IF NOT EXISTS hybrid BOOLEAN DEFAULT FALSE;

ALTER TABLE "release"
    ADD IF NOT EXISTS region TEXT;

ALTER TABLE "release"
    ADD IF NOT EXISTS other TEXT[] DEFAULT '{}' NOT NULL;

CREATE INDEX release_normalized_hash_index
    ON "release" (normalized_hash);

CREATE INDEX release_title_index
    ON "release" (title);

CREATE INDEX release_sub_title_index
    ON "release" (sub_title);

CREATE INDEX release_season_index
    ON "release" (season);

CREATE INDEX release_episode_index
    ON "release" (episode);

CREATE INDEX release_year_index
    ON "release" (year);

CREATE INDEX release_month_index
    ON "release" (month);

CREATE INDEX release_day_index
    ON "release" (day);

CREATE INDEX release_resolution_index
    ON "release" (resolution);

CREATE INDEX release_source_index
    ON "release" (source);

CREATE INDEX release_codec_index
    ON "release" (codec);

CREATE INDEX release_container_index
    ON "release" (container);

CREATE INDEX release_hdr_index
    ON "release" (hdr);

CREATE INDEX release_audio_index
    ON "release" (audio);

CREATE INDEX release_audio_channels_index
    ON "release" (audio_channels);

CREATE INDEX release_release_group_index
    ON "release" (release_group);

CREATE INDEX release_proper_index
    ON "release" (proper);

CREATE INDEX release_repack_index
    ON "release" (repack);

CREATE INDEX release_website_index
    ON "release" (website);

CREATE INDEX release_media_processing_index
    ON "release" (media_processing);

CREATE INDEX release_language_index
    ON "release" (language);

CREATE INDEX release_region_index
    ON "release" (region);

CREATE INDEX release_edition_index
    ON "release" (edition);

CREATE INDEX release_cut_index
    ON "release" (cut);

CREATE INDEX release_hybrid_index
    ON "release" (hybrid);
