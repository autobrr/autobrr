-- Add per-filter notification support
CREATE TABLE filter_notification
(
    filter_id       INTEGER NOT NULL,
    notification_id INTEGER NOT NULL,
    events          TEXT[]  NOT NULL DEFAULT '{}',
    created_at      TIMESTAMP        DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (filter_id, notification_id),
    FOREIGN KEY (filter_id) REFERENCES filter (id) ON DELETE CASCADE,
    FOREIGN KEY (notification_id) REFERENCES notification (id) ON DELETE CASCADE
);

CREATE INDEX idx_filter_notification_filter_id ON filter_notification (filter_id);
