CREATE TABLE proxy
(
    id         SERIAL PRIMARY KEY,
    enabled    BOOLEAN,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL,
    addr       TEXT NOT NULL,
    auth_user  TEXT,
    auth_pass  TEXT,
    timeout    INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE indexer
    ADD COLUMN proxy_id INTEGER;

ALTER TABLE indexer
    ADD COLUMN use_proxy BOOLEAN DEFAULT FALSE;

ALTER TABLE indexer
    ADD FOREIGN KEY (proxy_id) REFERENCES proxy
        ON DELETE SET NULL;

ALTER TABLE irc_network
    ADD COLUMN proxy_id INTEGER;

ALTER TABLE irc_network
    ADD COLUMN use_proxy BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
    ADD FOREIGN KEY (proxy_id) REFERENCES proxy
        ON DELETE SET NULL;