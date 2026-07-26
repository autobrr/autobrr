CREATE TABLE filter_external_dg_tmp
(
    id                               INTEGER PRIMARY KEY,
    name                             TEXT    NOT NULL,
    idx                              INTEGER,
    type                             TEXT,
    enabled                          BOOLEAN,
    exec_cmd                         TEXT,
    exec_args                        TEXT,
    exec_expect_status               INTEGER,
    webhook_host                     TEXT,
    webhook_method                   TEXT,
    webhook_data                     TEXT,
    webhook_headers                  TEXT,
    webhook_expect_status            INTEGER,
    webhook_retry_status             TEXT,
    webhook_retry_attempts           INTEGER,
    webhook_retry_delay_seconds      INTEGER,
    webhook_retry_max_jitter_seconds INTEGER,
    filter_id                        INTEGER NOT NULL
        REFERENCES filter
            ON DELETE CASCADE
);

INSERT INTO filter_external_dg_tmp(id, name, idx, type, enabled, exec_cmd, exec_args, exec_expect_status, webhook_host,
                                   webhook_method, webhook_data, webhook_headers, webhook_expect_status, filter_id,
                                   webhook_retry_status, webhook_retry_attempts, webhook_retry_delay_seconds,
                                   webhook_retry_max_jitter_seconds)
SELECT id,
       name,
       idx,
       type,
       enabled,
       exec_cmd,
       exec_args,
       exec_expect_status,
       webhook_host,
       webhook_method,
       webhook_data,
       webhook_headers,
       webhook_expect_status,
       filter_id,
       external_webhook_retry_status,
       external_webhook_retry_attempts,
       external_webhook_retry_delay_seconds,
       external_webhook_retry_max_jitter_seconds
FROM filter_external;

DROP TABLE filter_external;

ALTER TABLE filter_external_dg_tmp
    RENAME TO filter_external;
