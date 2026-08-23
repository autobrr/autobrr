-- defaults preserve previous behavior: 1 unit window with calendar reset
ALTER TABLE filter
    ADD COLUMN max_downloads_period INTEGER NOT NULL DEFAULT 1;

ALTER TABLE filter
    ADD COLUMN max_downloads_window_type TEXT NOT NULL DEFAULT 'FIXED';

-- (filter_id, status) lets the download window count skip rejected/errored
-- rows; it replaces the plain filter_id index, which it covers as a prefix.
-- The timestamp column is left out on purpose: stored values mix formats and
-- offsets, so the window predicate must normalize through datetime() and can
-- never range over a timestamp index
DROP INDEX IF EXISTS release_action_status_filter_id_index;

CREATE INDEX release_action_status_filter_id_status_index
    ON release_action_status (filter_id, status);
