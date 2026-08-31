ALTER TABLE indexer
    ADD COLUMN archived BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE indexer
    ADD COLUMN archived_at TIMESTAMP;

CREATE INDEX indexer_archived_index
    ON indexer (archived);

CREATE TABLE indexer_deprecation
(
    id            INTEGER PRIMARY KEY,
    identifier    TEXT NOT NULL UNIQUE,
    name          TEXT,
    reason        TEXT,
    issue_url     TEXT,
    alias_of      TEXT,
    deprecated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
