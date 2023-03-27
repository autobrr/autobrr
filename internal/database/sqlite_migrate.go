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

CREATE TABLE indexer
(
    id             INTEGER PRIMARY KEY,
    identifier     TEXT,
	implementation TEXT,
	base_url       TEXT,
    enabled        BOOLEAN,
    name           TEXT NOT NULL,
    settings       TEXT,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
    connected           BOOLEAN,
    connected_since     TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
    match_releases                 TEXT,
    except_releases                TEXT,
    use_regex                      BOOLEAN,
    match_release_groups           TEXT,
    except_release_groups          TEXT,
    match_release_tags             TEXT,
    except_release_tags            TEXT,
    use_regex_release_tags         BOOLEAN DEFAULT FALSE,
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
    tags                           TEXT,
    except_tags                    TEXT,
    origins                        TEXT []   DEFAULT '{}',
    except_origins                 TEXT []   DEFAULT '{}',
    external_script_enabled        BOOLEAN   DEFAULT FALSE,
    external_script_cmd            TEXT,
    external_script_args           TEXT,
    external_script_expect_status  INTEGER,
    external_webhook_enabled       BOOLEAN   DEFAULT FALSE,
    external_webhook_host          TEXT,
    external_webhook_data          TEXT,
    external_webhook_expect_status INTEGER,
    created_at                     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at                     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

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
    skip_hash_check         BOOLEAN DEFAULT false,
    content_layout          TEXT,
    limit_upload_speed      INT,
    limit_download_speed    INT,
    limit_ratio             REAL,
    limit_seed_time         INT,
    reannounce_skip         BOOLEAN DEFAULT false,
    reannounce_delete       BOOLEAN DEFAULT false,
    reannounce_interval     INTEGER DEFAULT 7,
    reannounce_max_attempts INTEGER DEFAULT 50,
    webhook_host            TEXT,
    webhook_method          TEXT,
    webhook_type            TEXT,
    webhook_data            TEXT,
    webhook_headers         TEXT[] DEFAULT '{}',
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
    group_id          TEXT,
    torrent_id        TEXT,
    torrent_name      TEXT,
    size              INTEGER,
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
	type          TEXT NOT NULL,
	client        TEXT,
	filter        TEXT,
	rejections    TEXT []   DEFAULT '{}' NOT NULL,
	timestamp     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	raw           TEXT,
	log           TEXT,
	release_id    INTEGER NOT NULL,
	FOREIGN KEY (release_id) REFERENCES "release"(id) ON DELETE CASCADE
);

CREATE INDEX release_action_status_release_id_index
    ON release_action_status (release_id);

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
	max_age       INTEGER DEFAULT 3600,
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
	bucket TEXT,
	key    TEXT,
	value  TEXT,
	ttl    TIMESTAMP
);

CREATE TABLE api_key
(
    name       TEXT,
    key        TEXT PRIMARY KEY,
    scopes     TEXT []   DEFAULT '{}' NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
	`ALTER TABLE notification
    ADD COLUMN user_key TEXT;
ALTER TABLE notification
    ADD COLUMN priority TEXT DEFAULT '0';
    `,
}
