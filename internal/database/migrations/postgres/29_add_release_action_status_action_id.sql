ALTER TABLE release_action_status
    ADD action_id INTEGER;

ALTER TABLE release_action_status
    ADD CONSTRAINT release_action_status_action_id_fk
        FOREIGN KEY (action_id) REFERENCES action;
