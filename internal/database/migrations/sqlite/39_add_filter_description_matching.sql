ALTER TABLE filter
    ADD COLUMN match_description TEXT;

ALTER TABLE filter
    ADD COLUMN except_description TEXT;

ALTER TABLE filter
    ADD COLUMN use_regex_description BOOLEAN DEFAULT FALSE;
