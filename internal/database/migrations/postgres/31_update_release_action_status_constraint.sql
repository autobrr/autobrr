ALTER TABLE release_action_status
    DROP CONSTRAINT IF EXISTS release_action_status_action_id_fkey;

ALTER TABLE release_action_status
    DROP CONSTRAINT IF EXISTS release_action_status_action_id_fk;

ALTER TABLE release_action_status
    ADD FOREIGN KEY (action_id) REFERENCES action
        ON DELETE SET NULL;