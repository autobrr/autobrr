ALTER TABLE filter
    ADD max_downloads INTEGER default 0;

ALTER TABLE filter
    ADD max_downloads_unit TEXT;

ALTER TABLE release
    add filter_id INTEGER;

CREATE INDEX release_filter_id_index
    ON release (filter_id);

ALTER TABLE release
    ADD CONSTRAINT release_filter_id_fk
        FOREIGN KEY (filter_id) REFERENCES FILTER
            ON DELETE SET NULL;