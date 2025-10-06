ALTER TABLE filter_external
    RENAME COLUMN external_webhook_retry_status TO webhook_retry_status;

ALTER TABLE filter_external
    RENAME COLUMN external_webhook_retry_attempts TO webhook_retry_attempts;

ALTER TABLE filter_external
    RENAME COLUMN external_webhook_retry_delay_seconds TO webhook_retry_delay_seconds;

ALTER TABLE filter_external
    RENAME COLUMN external_webhook_retry_max_jitter_seconds TO webhook_retry_max_jitter_seconds;
