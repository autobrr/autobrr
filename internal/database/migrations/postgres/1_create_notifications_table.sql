CREATE TABLE notification
(
    id         SERIAL PRIMARY KEY,
    name       TEXT,
    type       TEXT,
    enabled    BOOLEAN,
    events     TEXT[]    DEFAULT '{}' NOT NULL,
    token      TEXT,
    api_key    TEXT,
    webhook    TEXT,
    title      TEXT,
    icon       TEXT,
    host       TEXT,
    username   TEXT,
    password   TEXT,
    channel    TEXT,
    rooms      TEXT,
    targets    TEXT,
    devices    TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);