CREATE TABLE release_action_status
(
    id         INTEGER PRIMARY KEY,
    status     TEXT,
    action     TEXT                   NOT NULL,
    type       TEXT                   NOT NULL,
    rejections TEXT []   DEFAULT '{}' NOT NULL,
    timestamp  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    raw        TEXT,
    log        TEXT,
    release_id INTEGER                NOT NULL,
    FOREIGN KEY (release_id) REFERENCES "release" (id)
);

INSERT INTO "release_action_status" (status, action, type, timestamp, release_id)
SELECT push_status, 'DEFAULT', 'QBITTORRENT', timestamp, id
FROM "release";

ALTER TABLE "release"
    DROP COLUMN push_status;
