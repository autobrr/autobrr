ALTER TABLE release_action_status
    ADD filter_id INTEGER;

CREATE INDEX release_action_status_filter_id_index
    ON release_action_status (filter_id);

ALTER TABLE release_action_status
    ADD CONSTRAINT release_action_status_filter_id_fk
        FOREIGN KEY (filter_id) REFERENCES filter;

UPDATE release_action_status
SET filter_id = (SELECT f.id
                 FROM filter f
                 WHERE f.name = release_action_status.filter);
