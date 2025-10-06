CREATE TABLE proxy
(
    id         INTEGER PRIMARY KEY,
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
    ADD proxy_id INTEGER
        CONSTRAINT indexer_proxy_id_fk
            REFERENCES proxy (id)
            ON DELETE SET NULL;

ALTER TABLE indexer
    ADD use_proxy BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
    ADD use_proxy BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
    ADD proxy_id INTEGER
        CONSTRAINT irc_network_proxy_id_fk
            REFERENCES proxy (id)
            ON DELETE SET NULL;
