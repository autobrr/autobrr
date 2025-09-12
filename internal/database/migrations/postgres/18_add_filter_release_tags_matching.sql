ALTER TABLE filter
    ADD COLUMN match_release_tags TEXT;

ALTER TABLE filter
    ADD COLUMN except_release_tags TEXT;

ALTER TABLE filter
    ADD COLUMN use_regex_release_tags BOOLEAN DEFAULT FALSE;