CREATE TABLE feed_dg_tmp
(
    id            INTEGER PRIMARY KEY,
    indexer       TEXT,
    name          TEXT,
    type          TEXT,
    enabled       BOOLEAN,
    url           TEXT,
    interval      INTEGER,
    capabilities  TEXT      DEFAULT '{}' NOT NULL,
    api_key       TEXT,
    settings      TEXT,
    indexer_id    INTEGER
        REFERENCES indexer
            ON DELETE CASCADE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    timeout       INTEGER   DEFAULT 60,
    max_age       INTEGER   DEFAULT 0,
    last_run      TIMESTAMP,
    last_run_data TEXT,
    cookie        TEXT
);

INSERT INTO feed_dg_tmp(id, indexer, name, type, enabled, url, interval, capabilities, api_key, settings, indexer_id,
                        created_at, updated_at, timeout, max_age, last_run, last_run_data, cookie)
SELECT id,
       indexer,
       name,
       type,
       enabled,
       url,
       interval,
       capabilities,
       api_key,
       settings,
       indexer_id,
       created_at,
       updated_at,
       timeout,
       max_age,
       last_run,
       last_run_data,
       cookie
FROM feed;

DROP TABLE feed;

ALTER TABLE feed_dg_tmp
    RENAME TO feed;
