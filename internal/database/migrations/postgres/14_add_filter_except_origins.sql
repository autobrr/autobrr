ALTER TABLE filter
    ADD COLUMN except_origins TEXT[] DEFAULT '{}';
