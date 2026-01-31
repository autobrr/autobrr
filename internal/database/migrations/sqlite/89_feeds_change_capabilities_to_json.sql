CREATE TABLE feed_dg_tmp
(
    id            INTEGER PRIMARY KEY,
    indexer       TEXT,
    name          TEXT,
    type          TEXT,
    enabled       BOOLEAN,
    url           TEXT,
    interval      INTEGER,
    timeout       INTEGER   DEFAULT 60,
    max_age       INTEGER   DEFAULT 0,
    categories    TEXT      DEFAULT '{}' NOT NULL,
    capabilities  TEXT,
    api_key       TEXT,
    cookie        TEXT,
    settings      TEXT,
    indexer_id    INTEGER REFERENCES indexer ON DELETE SET NULL,
    last_run      TIMESTAMP,
    last_run_data TEXT,
    created_at    TIMESTAMP default CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP default CURRENT_TIMESTAMP
);

INSERT INTO feed_dg_tmp(id, indexer, name, type, enabled, url, interval, timeout, max_age, categories, capabilities,
                        api_key, cookie, settings, indexer_id, last_run, last_run_data, created_at, updated_at)
SELECT id,
       indexer,
       name,
       type,
       enabled,
       url,
       interval,
       timeout,
       max_age,
       categories,
       capabilities,
       api_key,
       cookie,
       settings,
       indexer_id,
       last_run,
       last_run_data,
       created_at,
       updated_at
FROM feed;

DROP TABLE feed;

ALTER TABLE feed_dg_tmp
    RENAME TO feed;
