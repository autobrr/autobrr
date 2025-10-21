alter table filter
    add max_downloads INTEGER default 0;

alter table filter
    add max_downloads_unit TEXT;

create table release_dg_tmp
(
    id             INTEGER
        primary key,
    filter_status  TEXT,
    rejections     TEXT []   default '{}' not null,
    indexer        TEXT,
    filter         TEXT,
    protocol       TEXT,
    implementation TEXT,
    timestamp      TIMESTAMP default CURRENT_TIMESTAMP,
    group_id       TEXT,
    torrent_id     TEXT,
    torrent_name   TEXT,
    size           INTEGER,
    title          TEXT,
    category       TEXT,
    season         INTEGER,
    episode        INTEGER,
    year           INTEGER,
    resolution     TEXT,
    source         TEXT,
    codec          TEXT,
    container      TEXT,
    hdr            TEXT,
    release_group  TEXT,
    proper         BOOLEAN,
    repack         BOOLEAN,
    website        TEXT,
    type           TEXT,
    origin         TEXT,
    tags           TEXT []   default '{}' not null,
    uploader       TEXT,
    pre_time       TEXT,
    filter_id      INTEGER
        CONSTRAINT release_filter_id_fk
            REFERENCES filter
            ON DELETE SET NULL
);

INSERT INTO release_dg_tmp(id, filter_status, rejections, indexer, filter, protocol, implementation, timestamp,
                           group_id, torrent_id, torrent_name, size, title, category, season, episode, year, resolution,
                           source, codec, container, hdr, release_group, proper, repack, website, type, origin, tags,
                           uploader, pre_time)
SELECT id,
       filter_status,
       rejections,
       indexer,
       filter,
       protocol,
       implementation,
       timestamp,
       group_id,
       torrent_id,
       torrent_name,
       size,
       title,
       category,
       season,
       episode,
       year,
       resolution,
       source,
       codec,
       container,
       hdr,
       release_group,
       proper,
       repack,
       website,
       type,
       origin,
       tags,
       uploader,
       pre_time
FROM "release";

DROP TABLE "release";

ALTER TABLE release_dg_tmp
    RENAME TO "release";

CREATE INDEX release_filter_id_index
    ON "release" (filter_id);
