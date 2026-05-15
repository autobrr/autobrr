ALTER TABLE filter
    ADD COLUMN external_script_enabled BOOLEAN DEFAULT FALSE;

ALTER TABLE filter
    ADD COLUMN external_script_cmd TEXT;

ALTER TABLE filter
    ADD COLUMN external_script_args TEXT;

ALTER TABLE filter
    ADD COLUMN external_script_expect_status INTEGER;

ALTER TABLE filter
    ADD COLUMN external_webhook_enabled BOOLEAN DEFAULT FALSE;

ALTER TABLE filter
    ADD COLUMN external_webhook_host TEXT;

ALTER TABLE filter
    ADD COLUMN external_webhook_data TEXT;

ALTER TABLE filter
    ADD COLUMN external_webhook_expect_status INTEGER;
