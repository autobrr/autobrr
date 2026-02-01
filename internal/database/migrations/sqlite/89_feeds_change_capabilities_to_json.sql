ALTER TABLE feed RENAME TO feed_old;

CREATE TABLE feed
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
    categories    TEXT []   DEFAULT '{}' NOT NULL,
    capabilities  TEXT      DEFAULT '{}' NOT NULL,
    api_key       TEXT,
    cookie        TEXT,
    settings      TEXT,
    indexer_id    INTEGER,
    last_run      TIMESTAMP,
    last_run_data TEXT,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (indexer_id) REFERENCES indexer (id) ON DELETE SET NULL
);

INSERT INTO feed (
    id, indexer, name, type, enabled, url, interval, timeout, max_age,
    api_key, cookie, settings, indexer_id,
    last_run, last_run_data, created_at, updated_at
)
SELECT
    id, indexer, name, type, enabled, url, interval, timeout, max_age,
    api_key, cookie, settings, indexer_id,
    last_run, last_run_data, created_at, updated_at
FROM feed_old;

DROP TABLE feed_old;
