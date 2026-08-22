-- defaults preserve previous behavior: 1 unit window with calendar reset
ALTER TABLE filter
    ADD COLUMN max_downloads_period INTEGER NOT NULL DEFAULT 1;

ALTER TABLE filter
    ADD COLUMN max_downloads_window_type TEXT NOT NULL DEFAULT 'FIXED';
