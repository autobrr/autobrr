CREATE TABLE release_action_status_dg_tmp
(
    id         INTEGER PRIMARY KEY,
    status     TEXT,
    action     TEXT                   not null,
    type       TEXT                   not null,
    rejections TEXT []   default '{}' not null,
    timestamp  TIMESTAMP default CURRENT_TIMESTAMP,
    raw        TEXT,
    log        TEXT,
    release_id INTEGER                not null
        CONSTRAINT release_action_status_release_id_fkey
            REFERENCES "release"
            ON DELETE CASCADE
);

INSERT INTO release_action_status_dg_tmp(id, status, action, type, rejections, timestamp, raw, log, release_id)
SELECT id,
       status,
       action,
       type,
       rejections,
       timestamp,
       raw,
       log,
       release_id
FROM release_action_status;

DROP TABLE release_action_status;

ALTER TABLE release_action_status_dg_tmp
    RENAME TO release_action_status;
