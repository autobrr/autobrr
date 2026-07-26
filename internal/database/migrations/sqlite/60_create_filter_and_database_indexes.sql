CREATE INDEX filter_external_filter_id_index
    ON filter_external (filter_id);

CREATE INDEX filter_enabled_index
    ON filter (enabled);

CREATE INDEX filter_priority_index
    ON filter (priority);
