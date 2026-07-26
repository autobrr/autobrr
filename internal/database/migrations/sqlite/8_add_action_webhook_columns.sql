ALTER TABLE "action"
    ADD COLUMN webhook_host TEXT;

ALTER TABLE "action"
    ADD COLUMN webhook_data TEXT;

ALTER TABLE "action"
    ADD COLUMN webhook_method TEXT;

ALTER TABLE "action"
    ADD COLUMN webhook_type TEXT;

ALTER TABLE "action"
    ADD COLUMN webhook_headers TEXT [] DEFAULT '{}';
