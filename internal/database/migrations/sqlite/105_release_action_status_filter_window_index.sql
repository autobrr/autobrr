-- timestamps are now UTC text that sorts by time, so the download window count can range over the index
DROP INDEX IF EXISTS release_action_status_filter_id_status_index;

CREATE INDEX release_action_status_filter_id_status_timestamp_index
    ON release_action_status (filter_id, status, timestamp);
