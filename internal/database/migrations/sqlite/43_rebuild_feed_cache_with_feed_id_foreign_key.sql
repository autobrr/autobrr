DROP TABLE IF EXISTS feed_cache;

CREATE TABLE feed_cache
(
    feed_id INTEGER NOT NULL,
    key     TEXT,
    value   TEXT,
    ttl     TIMESTAMP,
    FOREIGN KEY (feed_id) REFERENCES feed (id) ON DELETE cascade
);

CREATE INDEX feed_cache_feed_id_key_index
    ON feed_cache (feed_id, key);
