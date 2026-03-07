-- Defaults to 1 to maintain previous behavior - ie 1 download per (1) day/week/month
ALTER TABLE filter
    ADD max_downloads_interval INTEGER DEFAULT 1;

-- Fixed (truncate to day/week/etc) vs rolling window types (e.g. last 24 hours, last 7 days, etc)
ALTER TABLE filter
    ADD max_downloads_window_type TEXT DEFAULT 'FIXED';
