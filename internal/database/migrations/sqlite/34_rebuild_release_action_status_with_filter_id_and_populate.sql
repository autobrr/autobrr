CREATE TABLE release_action_status_dg_tmp
(
    id         INTEGER
        PRIMARY KEY,
    status     TEXT,
    action     TEXT                   NOT NULL,
    type       TEXT                   NOT NULL,
    rejections TEXT      DEFAULT '{}' NOT NULL,
    timestamp  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    raw        TEXT,
    log        TEXT,
    release_id INTEGER                NOT NULL
        constraint release_action_status_release_id_fkey
            references "release"
            on delete cascade,
    client     TEXT,
    filter     TEXT,
    filter_id  INTEGER
        CONSTRAINT release_action_status_filter_id_fk
            REFERENCES filter
);

INSERT INTO release_action_status_dg_tmp(id, status, action, type, rejections, timestamp, raw, log, release_id, client,
                                         filter)
SELECT id,
       status,
       action,
       type,
       rejections,
       timestamp,
       raw,
       log,
       release_id,
       client,
       filter
FROM release_action_status;

DROP TABLE release_action_status;

ALTER TABLE release_action_status_dg_tmp
    RENAME TO release_action_status;

CREATE INDEX release_action_status_filter_id_index
    ON release_action_status (filter_id);

CREATE INDEX release_action_status_release_id_index
    ON release_action_status (release_id);

CREATE INDEX release_action_status_status_index
    ON release_action_status (status);

UPDATE release_action_status
SET filter_id = (SELECT f.id
                 FROM filter f
                 WHERE f.name = release_action_status.filter);
