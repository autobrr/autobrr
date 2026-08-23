-- defaults preserve previous behavior: 1 unit window with calendar reset
ALTER TABLE filter
    ADD COLUMN max_downloads_period INTEGER NOT NULL DEFAULT 1;

ALTER TABLE filter
    ADD COLUMN max_downloads_window_type TEXT NOT NULL DEFAULT 'FIXED';

-- TIMESTAMPTZ makes the download window comparisons against CURRENT_TIMESTAMP
-- instant-correct when the app and database run in different timezones (#2089).
-- Existing naive values are reinterpreted in the session timezone: exact where
-- app and database share a zone, off by the offset otherwise until history ages
-- out of the enforcement window. Metadata-only on UTC servers; non-UTC servers
-- rewrite the table once
ALTER TABLE release_action_status
    ALTER COLUMN timestamp TYPE TIMESTAMPTZ;

-- the composite index lets the download window count run as an index range
-- scan; it replaces the plain filter_id index from migration 23, which fresh
-- installs never got, hence the guard
DROP INDEX IF EXISTS release_action_status_filter_id_index;

CREATE INDEX release_action_status_filter_id_status_timestamp_index
    ON release_action_status (filter_id, status, timestamp);
