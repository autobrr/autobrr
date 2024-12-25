// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

const sqliteSchema = `
CREATE TABLE users
(
    id         INTEGER PRIMARY KEY,
    username   TEXT NOT NULL,
    password   TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (username)
);

CREATE TABLE proxy
(
    id             INTEGER PRIMARY KEY,
    enabled        BOOLEAN,
    name           TEXT NOT NULL,
	type           TEXT NOT NULL,
    addr           TEXT NOT NULL,
	auth_user      TEXT,
	auth_pass      TEXT,
    timeout        INTEGER,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE indexer
(
    id                  INTEGER PRIMARY KEY,
    identifier          TEXT,
    identifier_external TEXT,
	implementation      TEXT,
	base_url            TEXT,
    enabled             BOOLEAN,
    name                TEXT NOT NULL,
    settings            TEXT,
    use_proxy           BOOLEAN DEFAULT FALSE,
    proxy_id            INTEGER,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (proxy_id) REFERENCES proxy(id) ON DELETE SET NULL,
    UNIQUE (identifier)
);

CREATE INDEX indexer_identifier_index
    ON indexer (identifier);

CREATE TABLE irc_network
(
    id                  INTEGER PRIMARY KEY,
    enabled             BOOLEAN,
    name                TEXT NOT NULL,
    server              TEXT NOT NULL,
    port                INTEGER NOT NULL,
    tls                 BOOLEAN,
    pass                TEXT,
    nick                TEXT,
    auth_mechanism      TEXT,
    auth_account        TEXT,
    auth_password       TEXT,
    invite_command      TEXT,
    use_bouncer         BOOLEAN,
    bouncer_addr        TEXT,
    bot_mode            BOOLEAN DEFAULT FALSE,
    connected           BOOLEAN,
    connected_since     TIMESTAMP,
    use_proxy           BOOLEAN DEFAULT FALSE,
    proxy_id            INTEGER,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (proxy_id) REFERENCES proxy(id) ON DELETE SET NULL,
    UNIQUE (server, port, nick)
);

CREATE TABLE irc_channel
(
    id          INTEGER PRIMARY KEY,
    enabled     BOOLEAN,
    name        TEXT NOT NULL,
    password    TEXT,
    detached    BOOLEAN,
    network_id  INTEGER NOT NULL,
    FOREIGN KEY (network_id) REFERENCES irc_network(id),
    UNIQUE (network_id, name)
);

CREATE TABLE filter
(
    id                             INTEGER PRIMARY KEY,
    enabled                        BOOLEAN,
    name                           TEXT NOT NULL,
    min_size                       TEXT,
    max_size                       TEXT,
    delay                          INTEGER,
    priority                       INTEGER   DEFAULT 0 NOT NULL,
    max_downloads                  INTEGER   DEFAULT 0,
    max_downloads_unit             TEXT,
	announce_types                 TEXT []   DEFAULT '{}',
    match_releases                 TEXT,
    except_releases                TEXT,
    use_regex                      BOOLEAN,
    match_release_groups           TEXT,
    except_release_groups          TEXT,
    match_release_tags             TEXT,
    except_release_tags            TEXT,
    use_regex_release_tags         BOOLEAN DEFAULT FALSE,
    match_description              TEXT,
    except_description             TEXT,
    use_regex_description          BOOLEAN DEFAULT FALSE,
    scene                          BOOLEAN,
    freeleech                      BOOLEAN,
    freeleech_percent              TEXT,
    smart_episode                  BOOLEAN DEFAULT FALSE,
    shows                          TEXT,
    seasons                        TEXT,
    episodes                       TEXT,
    resolutions                    TEXT []   DEFAULT '{}' NOT NULL,
    codecs                         TEXT []   DEFAULT '{}' NOT NULL,
    sources                        TEXT []   DEFAULT '{}' NOT NULL,
    containers                     TEXT []   DEFAULT '{}' NOT NULL,
    match_hdr                      TEXT []   DEFAULT '{}',
    except_hdr                     TEXT []   DEFAULT '{}',
    match_other                    TEXT []   DEFAULT '{}',
    except_other                   TEXT []   DEFAULT '{}',
    years                          TEXT,
    months                         TEXT,
    days                           TEXT,
    artists                        TEXT,
    albums                         TEXT,
    release_types_match            TEXT []   DEFAULT '{}',
    release_types_ignore           TEXT []   DEFAULT '{}',
    formats                        TEXT []   DEFAULT '{}',
    quality                        TEXT []   DEFAULT '{}',
    media                          TEXT []   DEFAULT '{}',
    log_score                      INTEGER,
    has_log                        BOOLEAN,
    has_cue                        BOOLEAN,
    perfect_flac                   BOOLEAN,
    match_categories               TEXT,
    except_categories              TEXT,
    match_uploaders                TEXT,
    except_uploaders               TEXT,
    match_record_labels            TEXT,
    except_record_labels           TEXT,
    match_language                 TEXT []   DEFAULT '{}',
    except_language                TEXT []   DEFAULT '{}',
    tags                           TEXT,
    except_tags                    TEXT,
    tags_match_logic               TEXT,
    except_tags_match_logic        TEXT,
    origins                        TEXT []   DEFAULT '{}',
    except_origins                 TEXT []   DEFAULT '{}',
    created_at                     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at                     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    min_seeders                    INTEGER DEFAULT 0,
    max_seeders                    INTEGER DEFAULT 0,
    min_leechers                   INTEGER DEFAULT 0,
    max_leechers                   INTEGER DEFAULT 0
);

CREATE INDEX filter_enabled_index
    ON filter (enabled);

CREATE INDEX filter_priority_index
    ON filter (priority);

CREATE TABLE filter_external
(
    id                                  INTEGER PRIMARY KEY,
    name                                TEXT     NOT NULL,
    idx                                 INTEGER,
    type                                TEXT,
    enabled                             BOOLEAN,
    exec_cmd                            TEXT,
    exec_args                           TEXT,
    exec_expect_status                  INTEGER,
    webhook_host                        TEXT,
    webhook_method                      TEXT,
    webhook_data                        TEXT,
    webhook_headers                     TEXT,
    webhook_expect_status               INTEGER,
    webhook_retry_status                TEXT,
    webhook_retry_attempts              INTEGER,
    webhook_retry_delay_seconds         INTEGER,
    filter_id                           INTEGER NOT NULL,
    FOREIGN KEY (filter_id)             REFERENCES filter(id) ON DELETE CASCADE
);

CREATE INDEX filter_external_filter_id_index
    ON filter_external(filter_id);

CREATE TABLE filter_indexer
(
    filter_id  INTEGER,
    indexer_id INTEGER,
    FOREIGN KEY (filter_id) REFERENCES filter(id),
    FOREIGN KEY (indexer_id) REFERENCES indexer(id) ON DELETE CASCADE,
    PRIMARY KEY (filter_id, indexer_id)
);

CREATE TABLE client
(
    id       		INTEGER PRIMARY KEY,
    name     		TEXT NOT NULL,
    enabled  		BOOLEAN,
    type     		TEXT,
    host     		TEXT NOT NULL,
    port     		INTEGER,
    tls      		BOOLEAN,
    tls_skip_verify BOOLEAN,
    username 		TEXT,
    password 		TEXT,
    settings 		JSON
);

CREATE TABLE action
(
    id                      INTEGER PRIMARY KEY,
    name                    TEXT,
    type                    TEXT,
    enabled                 BOOLEAN,
    exec_cmd                TEXT,
    exec_args               TEXT,
    watch_folder            TEXT,
    category                TEXT,
    tags                    TEXT,
    label                   TEXT,
    save_path               TEXT,
    paused                  BOOLEAN,
    ignore_rules            BOOLEAN,
    first_last_piece_prio   BOOLEAN DEFAULT false,
    skip_hash_check         BOOLEAN DEFAULT false,
    content_layout          TEXT,
    limit_upload_speed      INT,
    limit_download_speed    INT,
    limit_ratio             REAL,
    limit_seed_time         INT,
    priority                TEXT,
    reannounce_skip         BOOLEAN DEFAULT false,
    reannounce_delete       BOOLEAN DEFAULT false,
    reannounce_interval     INTEGER DEFAULT 7,
    reannounce_max_attempts INTEGER DEFAULT 50,
    webhook_host            TEXT,
    webhook_method          TEXT,
    webhook_type            TEXT,
    webhook_data            TEXT,
    webhook_headers         TEXT[] DEFAULT '{}',
    external_client_id      INTEGER,
    external_client         TEXT,
    client_id               INTEGER,
    filter_id               INTEGER,
    FOREIGN KEY (filter_id) REFERENCES filter (id),
    FOREIGN KEY (client_id) REFERENCES client (id) ON DELETE SET NULL
);

CREATE TABLE "release"
(
    id                INTEGER PRIMARY KEY,
    filter_status     TEXT,
    rejections        TEXT []   DEFAULT '{}' NOT NULL,
    indexer           TEXT,
    filter            TEXT,
    protocol          TEXT,
    implementation    TEXT,
    timestamp         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    announce_type     TEXT      DEFAULT 'NEW', 
    info_url          TEXT,
    download_url      TEXT,
    group_id          TEXT,
    torrent_id        TEXT,
    torrent_name      TEXT,
    size              INTEGER,
    title             TEXT,
    category          TEXT,
    season            INTEGER,
    episode           INTEGER,
    year              INTEGER,
    month             INTEGER,
    day               INTEGER,
    resolution        TEXT,
    source            TEXT,
    codec             TEXT,
    container         TEXT,
    hdr               TEXT,
    release_group     TEXT,
    proper            BOOLEAN,
    repack            BOOLEAN,
    website           TEXT,
    type              TEXT,
    origin            TEXT,
    tags              TEXT []   DEFAULT '{}' NOT NULL,
    uploader          TEXT,
    pre_time          TEXT,
    filter_id         INTEGER
        REFERENCES filter
            ON DELETE SET NULL
);

CREATE INDEX release_filter_id_index
    ON "release" (filter_id);

CREATE INDEX release_indexer_index
    ON "release" (indexer);

CREATE INDEX release_timestamp_index
    ON "release" (timestamp DESC);

CREATE INDEX release_torrent_name_index
    ON "release" (torrent_name);

CREATE TABLE release_action_status
(
	id            INTEGER PRIMARY KEY,
	status        TEXT,
	action        TEXT NOT NULL,
	action_id     INTEGER
        CONSTRAINT release_action_status_action_id_fk
            REFERENCES action,
	type          TEXT NOT NULL,
	client        TEXT,
	filter        TEXT,
    filter_id     INTEGER
        CONSTRAINT release_action_status_filter_id_fk
            REFERENCES filter,
	rejections    TEXT []   DEFAULT '{}' NOT NULL,
	timestamp     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	raw           TEXT,
	log           TEXT,
    release_id    INTEGER NOT NULL
        CONSTRAINT release_action_status_release_id_fkey
            REFERENCES "release"
            ON DELETE CASCADE
);

CREATE INDEX release_action_status_status_index
    ON release_action_status (status);

CREATE INDEX release_action_status_release_id_index
    ON release_action_status (release_id);

CREATE INDEX release_action_status_filter_id_index
    ON release_action_status (filter_id);

CREATE TABLE notification
(
	id         INTEGER PRIMARY KEY,
	name       TEXT,
	type       TEXT,
	enabled    BOOLEAN,
	events     TEXT []   DEFAULT '{}' NOT NULL,
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
	topic      TEXT,
	priority   INTEGER DEFAULT 0,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE feed
(
	id            INTEGER PRIMARY KEY,
	indexer       TEXT,
	name          TEXT,
	type          TEXT,
	enabled       BOOLEAN,
	url           TEXT,
	interval      INTEGER,
	timeout       INTEGER DEFAULT 60,
	max_age       INTEGER DEFAULT 0,
	categories    TEXT []   DEFAULT '{}' NOT NULL,
	capabilities  TEXT []   DEFAULT '{}' NOT NULL,
	api_key       TEXT,
	cookie        TEXT,
	settings      TEXT,
    indexer_id    INTEGER,
    last_run      TIMESTAMP,
    last_run_data TEXT,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (indexer_id) REFERENCES indexer(id) ON DELETE SET NULL
);

CREATE TABLE feed_cache
(
	feed_id INTEGER NOT NULL,
	key     TEXT,
	value   TEXT,
	ttl     TIMESTAMP,
	FOREIGN KEY (feed_id) REFERENCES feed (id) ON DELETE cascade
);

CREATE INDEX feed_cache_feed_id_key_index
    ON feed_cache (feed_id, key);

CREATE TABLE api_key
(
    name       TEXT,
    key        TEXT PRIMARY KEY,
    scopes     TEXT []   DEFAULT '{}' NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE list
(
    id                       INTEGER PRIMARY KEY,
    name                     TEXT                 NOT NULL,
    enabled                  BOOLEAN,
    type                     TEXT                 NOT NULL,
    client_id                INTEGER,
    url                      TEXT,
    headers                  TEXT [] DEFAULT '{}' NOT NULL,
    api_key                  TEXT,
    match_release            BOOLEAN,
    tags_included            TEXT [] DEFAULT '{}' NOT NULL,
    tags_excluded            TEXT [] DEFAULT '{}' NOT NULL,
    include_unmonitored      BOOLEAN,
    include_alternate_titles BOOLEAN,
    last_refresh_time        TIMESTAMP,
    last_refresh_status      TEXT,
    last_refresh_data        TEXT,
    created_at               TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at               TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES client (id) ON DELETE SET NULL
);

CREATE TABLE list_filter
(
    list_id   INTEGER,
    filter_id INTEGER,
    FOREIGN KEY (list_id) REFERENCES list(id) ON DELETE CASCADE,
    FOREIGN KEY (filter_id) REFERENCES filter(id) ON DELETE CASCADE,
    PRIMARY KEY (list_id, filter_id)
);
`

var sqliteMigrations = []string{
	"",
	`
	CREATE TABLE "release"
	(
		id                INTEGER PRIMARY KEY,
		filter_status     TEXT,
		push_status       TEXT,
		rejections        TEXT []   DEFAULT '{}' NOT NULL,
		indexer           TEXT,
		filter            TEXT,
		protocol          TEXT,
		implementation    TEXT,
		timestamp         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		group_id          TEXT,
		torrent_id        TEXT,
		torrent_name      TEXT,
		size              INTEGER,
		raw               TEXT,
		title             TEXT,
		category          TEXT,
		season            INTEGER,
		episode           INTEGER,
		year              INTEGER,
		resolution        TEXT,
		source            TEXT,
		codec             TEXT,
		container         TEXT,
		hdr               TEXT,
		audio             TEXT,
		release_group     TEXT,
		region            TEXT,
		language          TEXT,
		edition           TEXT,
		unrated           BOOLEAN,
		hybrid            BOOLEAN,
		proper            BOOLEAN,
		repack            BOOLEAN,
		website           TEXT,
		artists           TEXT []   DEFAULT '{}' NOT NULL,
		type              TEXT,
		format            TEXT,
		bitrate           TEXT,
		log_score         INTEGER,
		has_log           BOOLEAN,
		has_cue           BOOLEAN,
		is_scene          BOOLEAN,
		origin            TEXT,
		tags              TEXT []   DEFAULT '{}' NOT NULL,
		freeleech         BOOLEAN,
		freeleech_percent INTEGER,
		uploader          TEXT,
		pre_time          TEXT
	);
	`,
	`
	CREATE TABLE release_action_status
	(
		id            INTEGER PRIMARY KEY,
		status        TEXT,
		action        TEXT NOT NULL,
		type          TEXT NOT NULL,
		rejections    TEXT []   DEFAULT '{}' NOT NULL,
    	timestamp     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		raw           TEXT,
		log           TEXT,
		release_id    INTEGER NOT NULL,
		FOREIGN KEY (release_id) REFERENCES "release"(id)
	);

	INSERT INTO "release_action_status" (status, action, type, timestamp, release_id)
	SELECT push_status, 'DEFAULT', 'QBITTORRENT', timestamp, id FROM "release";

	ALTER TABLE "release"
	DROP COLUMN push_status;
	`,
	`
	ALTER TABLE "filter"
		ADD COLUMN match_hdr TEXT []   DEFAULT '{}';

	ALTER TABLE "filter"
		ADD COLUMN except_hdr TEXT []   DEFAULT '{}';
	`,
	`
	ALTER TABLE "release"
		RENAME COLUMN bitrate TO quality;

	ALTER TABLE "filter"
		ADD COLUMN artists TEXT;

	ALTER TABLE "filter"
		ADD COLUMN albums TEXT;

	ALTER TABLE "filter"
		ADD COLUMN release_types_match TEXT []   DEFAULT '{}';

	ALTER TABLE "filter"
		ADD COLUMN release_types_ignore TEXT []   DEFAULT '{}';

	ALTER TABLE "filter"
		ADD COLUMN formats TEXT []   DEFAULT '{}';

	ALTER TABLE "filter"
		ADD COLUMN quality TEXT []   DEFAULT '{}';

	ALTER TABLE "filter"
		ADD COLUMN log_score INTEGER;

	ALTER TABLE "filter"
		ADD COLUMN has_log BOOLEAN;

	ALTER TABLE "filter"
		ADD COLUMN has_cue BOOLEAN;

	ALTER TABLE "filter"
		ADD COLUMN perfect_flac BOOLEAN;
	`,
	`
	ALTER TABLE "filter"
		ADD COLUMN media TEXT []   DEFAULT '{}';
	`,
	`
	ALTER TABLE "filter"
		ADD COLUMN priority INTEGER DEFAULT 0 NOT NULL;
	`,
	`
	ALTER TABLE "client"
		ADD COLUMN tls_skip_verify BOOLEAN DEFAULT FALSE;

	ALTER TABLE "client"
		RENAME COLUMN ssl TO tls;
	`,
	`
	ALTER TABLE "action"
		ADD COLUMN webhook_host TEXT;

	ALTER TABLE "action"
		ADD COLUMN webhook_data TEXT;

	ALTER TABLE "action"
		ADD COLUMN webhook_method TEXT;

	ALTER TABLE "action"
		ADD COLUMN webhook_type TEXT;

	ALTER TABLE "action"
		ADD COLUMN webhook_headers TEXT []   DEFAULT '{}';
	`,
	`
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
	`,
	`
CREATE TABLE filter_indexer_dg_tmp
(
    filter_id  INTEGER
        CONSTRAINT filter_indexer_filter_id_fkey
            REFERENCES filter,
    indexer_id INTEGER
        CONSTRAINT filter_indexer_indexer_id_fkey
            REFERENCES indexer
            ON DELETE CASCADE,
    PRIMARY KEY (filter_id, indexer_id)
);

INSERT INTO filter_indexer_dg_tmp(filter_id, indexer_id)
SELECT filter_id, indexer_id
FROM filter_indexer;

DROP TABLE filter_indexer;

ALTER TABLE filter_indexer_dg_tmp
    RENAME TO filter_indexer;
	`,
	`
CREATE TABLE release_action_status_dg_tmp
(
    id         INTEGER PRIMARY KEY,
    status     TEXT,
    action     TEXT    not null,
    type       TEXT    not null,
    rejections TEXT []   default '{}' not null,
    timestamp  TIMESTAMP default CURRENT_TIMESTAMP,
    raw        TEXT,
    log        TEXT,
    release_id INTEGER not null
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
	`,
	`
	CREATE TABLE notification
	(
		id         INTEGER PRIMARY KEY,
		name       TEXT,
		type       TEXT,
		enabled    BOOLEAN,
		events     TEXT []   DEFAULT '{}' NOT NULL,
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
	`,
	`
	CREATE TABLE feed
	(
		id           INTEGER PRIMARY KEY,
		indexer      TEXT,
		name         TEXT,
		type         TEXT,
		enabled      BOOLEAN,
		url          TEXT,
		interval     INTEGER,
		categories   TEXT []   DEFAULT '{}' NOT NULL,
		capabilities TEXT []   DEFAULT '{}' NOT NULL,
		api_key      TEXT,
		settings     TEXT,
		indexer_id   INTEGER,
		created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (indexer_id) REFERENCES indexer(id) ON DELETE SET NULL
	);

	CREATE TABLE feed_cache
	(
		bucket TEXT,
		key    TEXT,
        value  TEXT,
		ttl    TIMESTAMP
	);
	`,
	`
	ALTER TABLE indexer
		ADD COLUMN implementation TEXT;
	`,
	`
	ALTER TABLE "release"
		RENAME COLUMN release_group TO "group";

	ALTER TABLE "release"
		DROP COLUMN raw;

	ALTER TABLE "release"
		DROP COLUMN audio;

	ALTER TABLE "release"
		DROP COLUMN region;

	ALTER TABLE "release"
		DROP COLUMN language;

	ALTER TABLE "release"
		DROP COLUMN edition;

	ALTER TABLE "release"
		DROP COLUMN unrated;

	ALTER TABLE "release"
		DROP COLUMN hybrid;

	ALTER TABLE "release"
		DROP COLUMN artists;

	ALTER TABLE "release"
		DROP COLUMN format;

	ALTER TABLE "release"
		DROP COLUMN quality;

	ALTER TABLE "release"
		DROP COLUMN log_score;

	ALTER TABLE "release"
		DROP COLUMN has_log;

	ALTER TABLE "release"
		DROP COLUMN has_cue;

	ALTER TABLE "release"
		DROP COLUMN is_scene;

	ALTER TABLE "release"
		DROP COLUMN freeleech;

	ALTER TABLE "release"
		DROP COLUMN freeleech_percent;

	ALTER TABLE "filter"
		ADD COLUMN origins TEXT []   DEFAULT '{}';
	`,
	`
	ALTER TABLE "filter"
		ADD COLUMN match_other TEXT []   DEFAULT '{}';

	ALTER TABLE "filter"
		ADD COLUMN except_other TEXT []   DEFAULT '{}';
	`,
	`
	ALTER TABLE "release"
		RENAME COLUMN "group" TO "release_group";
	`,
	`
	ALTER TABLE "action"
		ADD COLUMN reannounce_skip BOOLEAN DEFAULT false;

	ALTER TABLE "action"
		ADD COLUMN reannounce_delete BOOLEAN DEFAULT false;

	ALTER TABLE "action"
		ADD COLUMN reannounce_interval INTEGER DEFAULT 7;

	ALTER TABLE "action"
		ADD COLUMN reannounce_max_attempts INTEGER DEFAULT 50;
	`,
	`
	ALTER TABLE "action"
		ADD COLUMN limit_ratio REAL DEFAULT 0;

	ALTER TABLE "action"
		ADD COLUMN limit_seed_time INTEGER DEFAULT 0;
	`,
	`
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
	`,
	`
CREATE INDEX release_action_status_release_id_index
    ON release_action_status (release_id);

CREATE INDEX release_indexer_index
    ON "release" (indexer);

CREATE INDEX release_timestamp_index
    ON "release" (timestamp DESC);

CREATE INDEX release_torrent_name_index
    ON "release" (torrent_name);

CREATE INDEX indexer_identifier_index
    ON indexer (identifier);
	`,
	`
	ALTER TABLE release_action_status
		ADD COLUMN client TEXT;

	ALTER TABLE release_action_status
		ADD COLUMN filter TEXT;
	`,
	`
	ALTER TABLE filter
		ADD COLUMN external_script_enabled BOOLEAN DEFAULT FALSE;

	ALTER TABLE filter
		ADD COLUMN external_script_cmd TEXT;

	ALTER TABLE filter
		ADD COLUMN external_script_args TEXT;

	ALTER TABLE filter
		ADD COLUMN external_script_expect_status INTEGER;

	ALTER TABLE filter
		ADD COLUMN external_webhook_enabled BOOLEAN DEFAULT FALSE;

	ALTER TABLE filter
		ADD COLUMN external_webhook_host TEXT;

	ALTER TABLE filter
		ADD COLUMN external_webhook_data TEXT;

	ALTER TABLE filter
		ADD COLUMN external_webhook_expect_status INTEGER;
	`,
	`
	ALTER TABLE action
		ADD COLUMN skip_hash_check BOOLEAN DEFAULT FALSE;

	ALTER TABLE action
		ADD COLUMN content_layout TEXT;
	`,
	`
	ALTER TABLE filter
		ADD COLUMN except_origins TEXT []   DEFAULT '{}';
	`,
	`CREATE TABLE api_key
	(
		name       TEXT,
		key        TEXT PRIMARY KEY,
		scopes     TEXT []   DEFAULT '{}' NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`,
	`ALTER TABLE feed
     	ADD COLUMN timeout INTEGER DEFAULT 60;
    `,
	`ALTER TABLE feed
     	ADD COLUMN max_age INTEGER DEFAULT 3600;

	ALTER TABLE feed
     	ADD COLUMN last_run TIMESTAMP;

	ALTER TABLE feed
     	ADD COLUMN last_run_data TEXT;

	ALTER TABLE feed
     	ADD COLUMN cookie TEXT;
    `,
	`ALTER TABLE filter
		ADD COLUMN match_release_tags TEXT;

	ALTER TABLE filter
		ADD COLUMN except_release_tags TEXT;

	ALTER TABLE filter
		ADD COLUMN use_regex_release_tags BOOLEAN DEFAULT FALSE;
	`,
	`
CREATE TABLE irc_network_dg_tmp
(
    id              INTEGER
        primary key,
    enabled         BOOLEAN,
    name            TEXT    not null,
    server          TEXT    not null,
    port            INTEGER not null,
    tls             BOOLEAN,
    pass            TEXT,
    nick            TEXT,
    auth_mechanism  TEXT,
    auth_account    TEXT,
    auth_password   TEXT,
    invite_command  TEXT,
    connected       BOOLEAN,
    connected_since TIMESTAMP,
    created_at      TIMESTAMP default CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP default CURRENT_TIMESTAMP,
    unique (server, port, nick)
);

INSERT INTO irc_network_dg_tmp(id, enabled, name, server, port, tls, pass, nick, auth_mechanism, auth_account, auth_password, invite_command,
                               connected, connected_since, created_at, updated_at)
SELECT id,
       enabled,
       name,
       server,
       port,
       tls,
       pass,
       nickserv_account,
       'SASL_PLAIN',
       nickserv_account,
       nickserv_password,
       invite_command,
       connected,
       connected_since,
       created_at,
       updated_at
FROM irc_network;

DROP TABLE irc_network;

ALTER TABLE irc_network_dg_tmp
    RENAME TO irc_network;
	`,
	`ALTER TABLE indexer
     	ADD COLUMN base_url TEXT;
    `,
	`ALTER TABLE "filter"
	ADD COLUMN smart_episode BOOLEAN DEFAULT false;
	`,
	`ALTER TABLE "filter"
		ADD COLUMN match_language TEXT []   DEFAULT '{}';

	ALTER TABLE "filter"
		ADD COLUMN except_language TEXT []   DEFAULT '{}';
	`,
	`CREATE TABLE release_action_status_dg_tmp
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

INSERT INTO release_action_status_dg_tmp(id, status, action, type, rejections, timestamp, raw, log, release_id, client, filter)
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
FROM filter f WHERE f.name = release_action_status.filter);
	`,
	`ALTER TABLE "release"
ADD COLUMN info_url TEXT;

ALTER TABLE "release"
ADD COLUMN download_url TEXT;
	`,
	`ALTER TABLE filter
		ADD COLUMN tags_match_logic TEXT;

	ALTER TABLE filter
		ADD COLUMN except_tags_match_logic TEXT;

    UPDATE filter
    SET tags_match_logic = 'ANY'
    WHERE tags IS NOT NULL;

    UPDATE filter
    SET except_tags_match_logic = 'ANY'
    WHERE except_tags IS NOT NULL;
	`,
	`ALTER TABLE notification
ADD COLUMN priority INTEGER DEFAULT 0;`,
	`ALTER TABLE notification
ADD COLUMN topic text;`,
	`ALTER TABLE filter
		ADD COLUMN match_description TEXT;

	ALTER TABLE filter
		ADD COLUMN except_description TEXT;

	ALTER TABLE filter
		ADD COLUMN use_regex_description BOOLEAN DEFAULT FALSE;`,
	`create table release_action_status_dg_tmp
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
    on release_action_status (status);`,
	`ALTER TABLE irc_network
ADD COLUMN use_bouncer BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
ADD COLUMN bouncer_addr TEXT;`,
	`CREATE TABLE filter_external
(
    id                      INTEGER PRIMARY KEY,
    name                    TEXT     NOT NULL,
    idx                     INTEGER,
    type                    TEXT,
    enabled                 BOOLEAN,
    exec_cmd                TEXT,
    exec_args               TEXT,
    exec_expect_status      INTEGER,
    webhook_host            TEXT,
    webhook_method          TEXT,
    webhook_data            TEXT,
    webhook_headers         TEXT,
    webhook_expect_status   INTEGER,
    filter_id               INTEGER NOT NULL,
    FOREIGN KEY (filter_id) REFERENCES filter(id) ON DELETE CASCADE
);

INSERT INTO "filter_external" (name, type, enabled, exec_cmd, exec_args, exec_expect_status, filter_id)
SELECT 'exec', 'EXEC', external_script_enabled, external_script_cmd, external_script_args, external_script_expect_status, id FROM "filter" WHERE external_script_enabled = true;

INSERT INTO "filter_external" (name, type, enabled, webhook_host, webhook_data, webhook_method, webhook_expect_status, filter_id)
SELECT 'webhook', 'WEBHOOK', external_webhook_enabled, external_webhook_host, external_webhook_data, 'POST', external_webhook_expect_status, id FROM "filter" WHERE external_webhook_enabled = true;

create table filter_dg_tmp
(
    id                      INTEGER primary key,
    enabled                 BOOLEAN,
    name                    TEXT     not null,
    min_size                TEXT,
    max_size                TEXT,
    delay                   INTEGER,
    match_releases          TEXT,
    except_releases         TEXT,
    use_regex               BOOLEAN,
    match_release_groups    TEXT,
    except_release_groups   TEXT,
    scene                   BOOLEAN,
    freeleech               BOOLEAN,
    freeleech_percent       TEXT,
    shows                   TEXT,
    seasons                 TEXT,
    episodes                TEXT,
    resolutions             TEXT      default '{}' not null,
    codecs                  TEXT      default '{}' not null,
    sources                 TEXT      default '{}' not null,
    containers              TEXT      default '{}' not null,
    match_hdr               TEXT      default '{}',
    except_hdr              TEXT      default '{}',
    years                   TEXT,
    artists                 TEXT,
    albums                  TEXT,
    release_types_match     TEXT      default '{}',
    release_types_ignore    TEXT      default '{}',
    formats                 TEXT      default '{}',
    quality                 TEXT      default '{}',
    media                   TEXT      default '{}',
    log_score               INTEGER,
    has_log                 BOOLEAN,
    has_cue                 BOOLEAN,
    perfect_flac            BOOLEAN,
    match_categories        TEXT,
    except_categories       TEXT,
    match_uploaders         TEXT,
    except_uploaders        TEXT,
    tags                    TEXT,
    except_tags             TEXT,
    created_at              TIMESTAMP default CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP default CURRENT_TIMESTAMP,
    priority                INTEGER   default 0    not null,
    origins                 TEXT      default '{}',
    match_other             TEXT      default '{}',
    except_other            TEXT      default '{}',
    max_downloads           INTEGER   default 0,
    max_downloads_unit      TEXT,
    except_origins          TEXT      default '{}',
    match_release_tags      TEXT,
    except_release_tags     TEXT,
    use_regex_release_tags  BOOLEAN   default FALSE,
    smart_episode           BOOLEAN   default false,
    match_language          TEXT      default '{}',
    except_language         TEXT      default '{}',
    tags_match_logic        TEXT,
    except_tags_match_logic TEXT,
    match_description       TEXT,
    except_description      TEXT,
    use_regex_description   BOOLEAN   default FALSE
);

insert into filter_dg_tmp(id, enabled, name, min_size, max_size, delay, match_releases, except_releases, use_regex,
                          match_release_groups, except_release_groups, scene, freeleech, freeleech_percent, shows,
                          seasons, episodes, resolutions, codecs, sources, containers, match_hdr, except_hdr, years,
                          artists, albums, release_types_match, release_types_ignore, formats, quality, media,
                          log_score, has_log, has_cue, perfect_flac, match_categories, except_categories,
                          match_uploaders, except_uploaders, tags, except_tags, created_at, updated_at, priority,
                          origins, match_other, except_other, max_downloads, max_downloads_unit, except_origins,
                          match_release_tags, except_release_tags, use_regex_release_tags, smart_episode,
                          match_language, except_language, tags_match_logic, except_tags_match_logic, match_description,
                          except_description, use_regex_description)
select id,
       enabled,
       name,
       min_size,
       max_size,
       delay,
       match_releases,
       except_releases,
       use_regex,
       match_release_groups,
       except_release_groups,
       scene,
       freeleech,
       freeleech_percent,
       shows,
       seasons,
       episodes,
       resolutions,
       codecs,
       sources,
       containers,
       match_hdr,
       except_hdr,
       years,
       artists,
       albums,
       release_types_match,
       release_types_ignore,
       formats,
       quality,
       media,
       log_score,
       has_log,
       has_cue,
       perfect_flac,
       match_categories,
       except_categories,
       match_uploaders,
       except_uploaders,
       tags,
       except_tags,
       created_at,
       updated_at,
       priority,
       origins,
       match_other,
       except_other,
       max_downloads,
       max_downloads_unit,
       except_origins,
       match_release_tags,
       except_release_tags,
       use_regex_release_tags,
       smart_episode,
       match_language,
       except_language,
       tags_match_logic,
       except_tags_match_logic,
       match_description,
       except_description,
       use_regex_description
from filter;

drop table filter;

alter table filter_dg_tmp
    rename to filter;
`,
	`DROP TABLE IF EXISTS feed_cache;

CREATE TABLE feed_cache
(
	feed_id INTEGER NOT NULL,
	key     TEXT,
	value   TEXT,
	ttl     TIMESTAMP,
	FOREIGN KEY (feed_id) REFERENCES feed (id) ON DELETE cascade
);

CREATE INDEX feed_cache_feed_id_key_index
    ON feed_cache (feed_id, key);
`,
	`ALTER TABLE action
ADD COLUMN external_client_id INTEGER;
`,
	`ALTER TABLE filter_external
ADD COLUMN external_webhook_retry_status TEXT;

ALTER TABLE filter_external
	ADD COLUMN external_webhook_retry_attempts INTEGER;

ALTER TABLE filter_external
	ADD COLUMN external_webhook_retry_delay_seconds INTEGER;

ALTER TABLE filter_external
	ADD COLUMN external_webhook_retry_max_jitter_seconds INTEGER;
`,
	`
CREATE TABLE filter_external_dg_tmp
(
    id                               INTEGER PRIMARY KEY,
    name                             TEXT    NOT NULL,
    idx                              INTEGER,
    type                             TEXT,
    enabled                          BOOLEAN,
    exec_cmd                         TEXT,
    exec_args                        TEXT,
    exec_expect_status               INTEGER,
    webhook_host                     TEXT,
    webhook_method                   TEXT,
    webhook_data                     TEXT,
    webhook_headers                  TEXT,
    webhook_expect_status            INTEGER,
    webhook_retry_status             TEXT,
    webhook_retry_attempts           INTEGER,
    webhook_retry_delay_seconds      INTEGER,
    webhook_retry_max_jitter_seconds INTEGER,
    filter_id                        INTEGER NOT NULL
        REFERENCES filter
            ON DELETE CASCADE
);

INSERT INTO filter_external_dg_tmp(id, name, idx, type, enabled, exec_cmd, exec_args, exec_expect_status, webhook_host,
                                   webhook_method, webhook_data, webhook_headers, webhook_expect_status, filter_id,
                                   webhook_retry_status, webhook_retry_attempts, webhook_retry_delay_seconds,
                                   webhook_retry_max_jitter_seconds)
SELECT id,
       name,
       idx,
       type,
       enabled,
       exec_cmd,
       exec_args,
       exec_expect_status,
       webhook_host,
       webhook_method,
       webhook_data,
       webhook_headers,
       webhook_expect_status,
       filter_id,
       external_webhook_retry_status,
       external_webhook_retry_attempts,
       external_webhook_retry_delay_seconds,
       external_webhook_retry_max_jitter_seconds
FROM filter_external;

DROP TABLE filter_external;

ALTER TABLE filter_external_dg_tmp
    RENAME TO filter_external;
`,
	`ALTER TABLE filter_external
	DROP COLUMN webhook_retry_max_jitter_seconds;
`,
	`ALTER TABLE irc_network
	ADD COLUMN bot_mode BOOLEAN DEFAULT FALSE;
`,
	`CREATE TABLE feed_dg_tmp
(
    id            INTEGER PRIMARY KEY,
    indexer       TEXT,
    name          TEXT,
    type          TEXT,
    enabled       BOOLEAN,
    url           TEXT,
    interval      INTEGER,
    capabilities  TEXT      DEFAULT '{}' NOT NULL,
    api_key       TEXT,
    settings      TEXT,
    indexer_id    INTEGER
        REFERENCES indexer
            ON DELETE CASCADE,
    created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    timeout       INTEGER   DEFAULT 60,
    max_age       INTEGER   DEFAULT 0,
    last_run      TIMESTAMP,
    last_run_data TEXT,
    cookie        TEXT
);

INSERT INTO feed_dg_tmp(id, indexer, name, type, enabled, url, interval, capabilities, api_key, settings, indexer_id,
                        created_at, updated_at, timeout, max_age, last_run, last_run_data, cookie)
SELECT id,
       indexer,
       name,
       type,
       enabled,
       url,
       interval,
       capabilities,
       api_key,
       settings,
       indexer_id,
       created_at,
       updated_at,
       timeout,
       max_age,
       last_run,
       last_run_data,
       cookie
FROM feed;

DROP TABLE feed;

ALTER TABLE feed_dg_tmp
    RENAME TO feed;
`,
	`ALTER TABLE action
	ADD COLUMN priority TEXT;
`,
	`ALTER TABLE action
	ADD COLUMN external_client TEXT;
`, `
ALTER TABLE filter
    ADD COLUMN min_seeders INTEGER DEFAULT 0;

ALTER TABLE filter
    ADD COLUMN max_seeders INTEGER DEFAULT 0;

ALTER TABLE filter
    ADD COLUMN min_leechers INTEGER DEFAULT 0;

ALTER TABLE filter
    ADD COLUMN max_leechers INTEGER DEFAULT 0;
`,
	`UPDATE irc_network
    SET server = 'irc.nebulance.io'
    WHERE server = 'irc.nebulance.cc';
`,
	`UPDATE  irc_network
    SET server = 'irc.animefriends.moe',
        name = CASE  
			WHEN name = 'AnimeBytes-IRC' THEN 'AnimeBytes'
        	ELSE name
        END
	WHERE server = 'irc.animebytes.tv';
`,
	`ALTER TABLE action
    ADD COLUMN first_last_piece_prio BOOLEAN DEFAULT false;
`,
	`ALTER TABLE indexer
    ADD COLUMN identifier_external TEXT;

	UPDATE indexer
    SET identifier_external = name;
`,
	`ALTER TABLE "release"
ADD COLUMN month INTEGER;

ALTER TABLE "release"
ADD COLUMN day INTEGER;

ALTER TABLE filter
ADD COLUMN months TEXT;

ALTER TABLE filter
ADD COLUMN days TEXT;
`,
	`CREATE TABLE proxy
(
    id             INTEGER PRIMARY KEY,
    enabled        BOOLEAN,
    name           TEXT NOT NULL,
	type           TEXT NOT NULL,
    addr           TEXT NOT NULL,
	auth_user      TEXT,
	auth_pass      TEXT,
    timeout        INTEGER,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE indexer
    ADD proxy_id INTEGER
        CONSTRAINT indexer_proxy_id_fk
            REFERENCES proxy(id)
            ON DELETE SET NULL;

ALTER TABLE indexer
    ADD use_proxy BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
    ADD use_proxy BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
    ADD proxy_id INTEGER
        CONSTRAINT irc_network_proxy_id_fk
            REFERENCES proxy(id)
            ON DELETE SET NULL;
`,
	`UPDATE indexer
	SET base_url = 'https://fuzer.xyz/'
	WHERE base_url = 'https://fuzer.me/';
`,
	`CREATE INDEX filter_external_filter_id_index
    ON filter_external(filter_id);

CREATE INDEX filter_enabled_index
    ON filter (enabled);

CREATE INDEX filter_priority_index
    ON filter (priority);
`,
	`UPDATE irc_network
    SET server = 'irc.fuzer.xyz'
    WHERE server = 'irc.fuzer.me';
`,
	`UPDATE irc_network
	SET server = 'irc.scenehd.org'
	WHERE server = 'irc.scenehd.eu';
	
UPDATE irc_network
	SET server = 'irc.p2p-network.net', name = 'P2P-Network', nick = nick || '_0'
	WHERE server = 'irc.librairc.net';
	
UPDATE irc_network
	SET server = 'irc.atw-inter.net', name = 'ATW-Inter'
	WHERE server = 'irc.ircnet.com';
`,
	`UPDATE indexer
	SET base_url = 'https://redacted.sh/'
	WHERE base_url = 'https://redacted.ch/';
`,
	`UPDATE irc_network
    SET port = '6697', tls = true
    WHERE server = 'irc.seedpool.org';
`,
	`ALTER TABLE "release"
	ADD COLUMN announce_type TEXT DEFAULT 'NEW';

	ALTER TABLE filter
	ADD COLUMN announce_types TEXT []   DEFAULT '{}';
`,
	`CREATE TABLE list
(
    id                       INTEGER PRIMARY KEY,
    name                     TEXT                 NOT NULL,
    enabled                  BOOLEAN,
    type                     TEXT                 NOT NULL,
    client_id                INTEGER,
    url                      TEXT,
    headers                  TEXT [] DEFAULT '{}' NOT NULL,
    api_key                  TEXT,
    match_release            BOOLEAN,
    tags_included            TEXT [] DEFAULT '{}' NOT NULL,
    tags_excluded            TEXT [] DEFAULT '{}' NOT NULL,
    include_unmonitored      BOOLEAN,
    include_alternate_titles BOOLEAN,
    last_refresh_time        TIMESTAMP,
    last_refresh_status      TEXT,
    last_refresh_data        TEXT,
    created_at               TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at               TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (client_id) REFERENCES client (id) ON DELETE SET NULL
);

CREATE TABLE list_filter
(
    list_id   INTEGER,
    filter_id INTEGER,
    FOREIGN KEY (list_id) REFERENCES list(id) ON DELETE CASCADE,
    FOREIGN KEY (filter_id) REFERENCES filter(id) ON DELETE CASCADE,
    PRIMARY KEY (list_id, filter_id)
);
`,
	`ALTER TABLE filter
  ADD COLUMN match_record_labels TEXT DEFAULT '';

  ALTER TABLE filter
  ADD COLUMN except_record_labels TEXT DEFAULT '';
`,
	`UPDATE irc_channel 
    SET name = '#ptp-announce'
    WHERE name = '#ptp-announce-dev';
`,
	`UPDATE irc_network
  SET invite_command = REPLACE(invite_command, '#ptp-announce-dev', '#ptp-announce')
  WHERE invite_command LIKE '%#ptp-announce-dev%';
`,
}
