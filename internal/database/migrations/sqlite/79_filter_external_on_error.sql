ALTER TABLE filter_external
    ADD COLUMN on_error TEXT DEFAULT 'REJECT';
