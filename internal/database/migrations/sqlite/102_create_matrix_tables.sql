CREATE TABLE matrix_network
(
    id                   INTEGER PRIMARY KEY,
    enabled              BOOLEAN   DEFAULT FALSE,
    name                 TEXT NOT NULL,
    homeserver           TEXT NOT NULL,
    auth_method          TEXT,
    user_id              TEXT,
    device_id            TEXT,
    access_token         TEXT,
    refresh_token        TEXT,
    token_expires_at     TIMESTAMP,
    oauth_client_id      TEXT,
    oauth_token_endpoint TEXT,
    created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE matrix_room
(
    id         INTEGER PRIMARY KEY,
    network_id INTEGER NOT NULL,
    enabled    BOOLEAN   DEFAULT TRUE,
    name       TEXT NOT NULL,
    room_id    TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (network_id) REFERENCES matrix_network (id) ON DELETE CASCADE,
    UNIQUE (network_id, name)
);
