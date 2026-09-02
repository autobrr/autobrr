ALTER TABLE "action"
    ADD COLUMN reannounce_skip BOOLEAN DEFAULT false;

ALTER TABLE "action"
    ADD COLUMN reannounce_delete BOOLEAN DEFAULT false;

ALTER TABLE "action"
    ADD COLUMN reannounce_interval INTEGER DEFAULT 7;

ALTER TABLE "action"
    ADD COLUMN reannounce_max_attempts INTEGER DEFAULT 50;
