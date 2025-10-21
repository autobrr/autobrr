create table release_action_status_dg_tmp
(
    id         INTEGER
        primary key,
    status     TEXT,
    action     TEXT                   not null,
    action_id  INTEGER
        constraint release_action_status_action_id_fk
            references action,
    type       TEXT                   not null,
    rejections TEXT      default '{}' not null,
    timestamp  TIMESTAMP default CURRENT_TIMESTAMP,
    raw        TEXT,
    log        TEXT,
    release_id INTEGER                not null
        constraint release_action_status_release_id_fkey
            references "release"
            on delete cascade,
    client     TEXT,
    filter     TEXT,
    filter_id  INTEGER
        constraint release_action_status_filter_id_fk
            references filter
);

insert into release_action_status_dg_tmp(id, status, action, type, rejections, timestamp, raw, log, release_id, client,
                                         filter, filter_id)
select id,
       status,
       action,
       type,
       rejections,
       timestamp,
       raw,
       log,
       release_id,
       client,
       filter,
       filter_id
from release_action_status;

drop table release_action_status;

alter table release_action_status_dg_tmp
    rename to release_action_status;

create index release_action_status_filter_id_index
    on release_action_status (filter_id);

create index release_action_status_release_id_index
    on release_action_status (release_id);

create index release_action_status_status_index
    on release_action_status (status);
