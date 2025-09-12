ALTER TABLE filter_external
    ADD COLUMN external_webhook_retry_status TEXT;

ALTER TABLE filter_external
    ADD COLUMN external_webhook_retry_attempts INTEGER;

ALTER TABLE filter_external
    ADD COLUMN external_webhook_retry_delay_seconds INTEGER;

ALTER TABLE filter_external
    ADD COLUMN external_webhook_retry_max_jitter_seconds INTEGER;