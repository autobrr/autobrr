ALTER TABLE action
    ADD COLUMN skip_hash_check BOOLEAN DEFAULT FALSE;

ALTER TABLE action
    ADD COLUMN content_layout TEXT;