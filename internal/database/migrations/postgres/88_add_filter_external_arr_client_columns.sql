ALTER TABLE filter_external
    ADD COLUMN client_id INTEGER;

ALTER TABLE filter_external
    ADD COLUMN external_download_client_id INTEGER;

ALTER TABLE filter_external
    ADD COLUMN external_download_client TEXT;
