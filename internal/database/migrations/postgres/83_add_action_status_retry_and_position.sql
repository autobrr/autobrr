ALTER TABLE action ADD COLUMN position INTEGER DEFAULT 0;
ALTER TABLE action ADD COLUMN exec_expect_status INTEGER;
ALTER TABLE action ADD COLUMN webhook_expect_status INTEGER;
ALTER TABLE action ADD COLUMN webhook_retry_status TEXT;
ALTER TABLE action ADD COLUMN webhook_retry_attempts INTEGER;
ALTER TABLE action ADD COLUMN webhook_retry_delay_seconds INTEGER;
ALTER TABLE action ADD COLUMN on_error TEXT DEFAULT 'CONTINUE';
