CREATE TABLE feed
(
    id           INTEGER PRIMARY KEY,
    indexer      TEXT,
    name         TEXT,
    type         TEXT,
    enabled      BOOLEAN,
    url          TEXT,
    interval     INTEGER,
    categories   TEXT []   DEFAULT '{}' NOT NULL,
    capabilities TEXT []   DEFAULT '{}' NOT NULL,
    api_key      TEXT,
    settings     TEXT,
    indexer_id   INTEGER,
    created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (indexer_id) REFERENCES indexer (id) ON DELETE SET NULL
);

CREATE TABLE feed_cache
(
    bucket TEXT,
    key    TEXT,
    value  TEXT,
    ttl    TIMESTAMP
);
