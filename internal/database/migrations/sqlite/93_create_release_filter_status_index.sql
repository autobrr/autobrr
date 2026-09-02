CREATE INDEX release_filter_status_index
    ON "release" (filter_status);

CREATE INDEX release_action_status_timestamp_status_index
    ON release_action_status (timestamp, status);
