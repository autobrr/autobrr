CREATE TABLE filter_external
(
    id                    SERIAL PRIMARY KEY,
    name                  TEXT    NOT NULL,
    idx                   INTEGER,
    type                  TEXT,
    enabled               BOOLEAN,
    exec_cmd              TEXT,
    exec_args             TEXT,
    exec_expect_status    INTEGER,
    webhook_host          TEXT,
    webhook_method        TEXT,
    webhook_data          TEXT,
    webhook_headers       TEXT,
    webhook_expect_status INTEGER,
    filter_id             INTEGER NOT NULL,
    FOREIGN KEY (filter_id) REFERENCES filter (id) ON DELETE CASCADE
);

INSERT INTO "filter_external" (name, type, enabled, exec_cmd, exec_args, exec_expect_status, filter_id)
SELECT 'exec',
       'EXEC',
       external_script_enabled,
       external_script_cmd,
       external_script_args,
       external_script_expect_status,
       id
FROM "filter"
WHERE external_script_enabled = true;

INSERT INTO "filter_external" (name, type, enabled, webhook_host, webhook_data, webhook_method, webhook_expect_status,
                               filter_id)
SELECT 'webhook',
       'WEBHOOK',
       external_webhook_enabled,
       external_webhook_host,
       external_webhook_data,
       'POST',
       external_webhook_expect_status,
       id
FROM "filter"
WHERE external_webhook_enabled = true;

ALTER TABLE filter
    DROP COLUMN IF EXISTS external_script_enabled;

ALTER TABLE filter
    DROP COLUMN IF EXISTS external_script_cmd;

ALTER TABLE filter
    DROP COLUMN IF EXISTS external_script_args;

ALTER TABLE filter
    DROP COLUMN IF EXISTS external_script_expect_status;

ALTER TABLE filter
    DROP COLUMN IF EXISTS external_webhook_enabled;

ALTER TABLE filter
    DROP COLUMN IF EXISTS external_webhook_host;

ALTER TABLE filter
    DROP COLUMN IF EXISTS external_webhook_data;

ALTER TABLE filter
    DROP COLUMN IF EXISTS external_webhook_expect_status;
