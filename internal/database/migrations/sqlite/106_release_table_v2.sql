-- only matched releases are stored, so filter_status is always FILTER_APPROVED, and rejections is never read
DROP INDEX IF EXISTS release_filter_status_index;
ALTER TABLE "release" DROP COLUMN filter_status;
ALTER TABLE "release" DROP COLUMN rejections;

DROP INDEX IF EXISTS release_torrent_name_index;
DROP INDEX IF EXISTS release_hdr_index;
ALTER TABLE "release" RENAME COLUMN torrent_name TO release_name;
ALTER TABLE "release" RENAME COLUMN torrent_id TO external_id;
ALTER TABLE "release" RENAME COLUMN hdr TO dynamic_range;

ALTER TABLE "release" ADD COLUMN indexer_id INTEGER;
ALTER TABLE "release" ADD COLUMN name_key TEXT DEFAULT '' NOT NULL;
ALTER TABLE "release" ADD COLUMN title_key TEXT DEFAULT '' NOT NULL;
ALTER TABLE "release" ADD COLUMN sub_title_key TEXT DEFAULT '' NOT NULL;
ALTER TABLE "release" ADD COLUMN categories TEXT [] DEFAULT '{}' NOT NULL;
ALTER TABLE "release" ADD COLUMN indexer_flags TEXT [] DEFAULT '{}' NOT NULL;
ALTER TABLE "release" ADD COLUMN download_volume_factor REAL;
ALTER TABLE "release" ADD COLUMN upload_volume_factor REAL;
ALTER TABLE "release" ADD COLUMN publish_time TIMESTAMP;
ALTER TABLE "release" ADD COLUMN id_imdb TEXT;
ALTER TABLE "release" ADD COLUMN id_tmdb INTEGER;
ALTER TABLE "release" ADD COLUMN id_tvdb INTEGER;
ALTER TABLE "release" ADD COLUMN id_igdb INTEGER;
ALTER TABLE "release" ADD COLUMN info_hash TEXT;
ALTER TABLE "release" ADD COLUMN trace_id TEXT;
ALTER TABLE "release" ADD COLUMN metadata TEXT;

UPDATE "release"
SET indexer_id = (SELECT i.id FROM indexer i WHERE i.identifier = "release".indexer)
WHERE indexer IS NOT NULL;

CREATE INDEX release_release_name_index
    ON "release" (release_name);

CREATE INDEX release_name_key_index
    ON "release" (name_key);

CREATE INDEX release_title_key_year_index
    ON "release" (title_key, year);

DROP INDEX IF EXISTS release_action_status_release_id_index;

CREATE INDEX release_action_status_release_id_status_index
    ON release_action_status (release_id, status);
