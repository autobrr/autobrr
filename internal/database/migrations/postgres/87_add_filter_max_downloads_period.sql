-- defaults preserve previous behavior: 1 unit window with calendar reset
ALTER TABLE filter
    ADD COLUMN max_downloads_period INTEGER NOT NULL DEFAULT 1;

ALTER TABLE filter
    ADD COLUMN max_downloads_window_type TEXT NOT NULL DEFAULT 'FIXED';

-- fresh installs never got this index; upgraded installs created it in
-- 23_release_action_status_add_filter_id.sql, hence the guard
CREATE INDEX IF NOT EXISTS release_action_status_filter_id_index
    ON release_action_status (filter_id);
