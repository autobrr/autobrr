CREATE INDEX release_action_status_release_id_index
    ON release_action_status (release_id);

CREATE INDEX release_indexer_index
    ON "release" (indexer);

CREATE INDEX release_timestamp_index
    ON "release" (timestamp DESC);

CREATE INDEX release_torrent_name_index
    ON "release" (torrent_name);

CREATE INDEX indexer_identifier_index
    ON indexer (identifier);