CREATE TABLE action_dg_tmp
(
    id                   INTEGER PRIMARY KEY,
    name                 TEXT,
    type                 TEXT,
    enabled              BOOLEAN,
    exec_cmd             TEXT,
    exec_args            TEXT,
    watch_folder         TEXT,
    category             TEXT,
    tags                 TEXT,
    label                TEXT,
    save_path            TEXT,
    paused               BOOLEAN,
    ignore_rules         BOOLEAN,
    limit_upload_speed   INT,
    limit_download_speed INT,
    client_id            INTEGER
        CONSTRAINT action_client_id_fkey
            REFERENCES client
            ON DELETE SET NULL,
    filter_id            INTEGER
        CONSTRAINT action_filter_id_fkey
            REFERENCES filter,
    webhook_host         TEXT,
    webhook_data         TEXT,
    webhook_method       TEXT,
    webhook_type         TEXT,
    webhook_headers      TEXT [] default '{}'
);

INSERT INTO action_dg_tmp(id, name, type, enabled, exec_cmd, exec_args, watch_folder, category, tags, label, save_path,
                          paused, ignore_rules, limit_upload_speed, limit_download_speed, client_id, filter_id,
                          webhook_host, webhook_data, webhook_method, webhook_type, webhook_headers)
SELECT id,
       name,
       type,
       enabled,
       exec_cmd,
       exec_args,
       watch_folder,
       category,
       tags,
       label,
       save_path,
       paused,
       ignore_rules,
       limit_upload_speed,
       limit_download_speed,
       client_id,
       filter_id,
       webhook_host,
       webhook_data,
       webhook_method,
       webhook_type,
       webhook_headers
FROM action;

DROP TABLE action;

ALTER TABLE action_dg_tmp
    RENAME TO action;
