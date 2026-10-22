CREATE TABLE notification_inbox
(
    id            SERIAL PRIMARY KEY,
    event         TEXT NOT NULL,
    title         TEXT NOT NULL,
    message       TEXT NOT NULL,
    release_name  TEXT,
    indexer       TEXT,
    filter_name   TEXT,
    filter_id     INTEGER,
    action        TEXT,
    action_client TEXT,
    rejections    TEXT[]    DEFAULT '{}' NOT NULL,
    url           TEXT,
    read_at       TIMESTAMPTZ,
    created_at    TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX notification_inbox_created_at_index
    ON notification_inbox (created_at);

-- every install gets one built-in notification; the partial unique index keeps it a singleton
INSERT INTO notification (name, type, enabled, events, created_at, updated_at)
SELECT 'Built-in', 'BUILTIN', TRUE, '{PUSH_ERROR,IRC_DISCONNECTED,APP_UPDATE_AVAILABLE}'::TEXT[], CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM notification WHERE type = 'BUILTIN');

CREATE UNIQUE INDEX notification_builtin_unique
    ON notification (type)
    WHERE type = 'BUILTIN';
