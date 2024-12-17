// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

const postgresSchema = `
CREATE TABLE users
(
    id         SERIAL PRIMARY KEY,
    username   TEXT NOT NULL,
    password   TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (username)
);

CREATE TABLE proxy
(
    id             SERIAL PRIMARY KEY,
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
    id                  SERIAL PRIMARY KEY,
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
    id                  SERIAL PRIMARY KEY,
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
    id          SERIAL PRIMARY KEY,
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
    id                             SERIAL PRIMARY KEY,
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
    id                                  SERIAL PRIMARY KEY,
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
    id       		SERIAL PRIMARY KEY,
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
    id                      SERIAL PRIMARY KEY,
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
    priority			    TEXT,
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
    id                SERIAL PRIMARY KEY,
    filter_status     TEXT,
    rejections        TEXT []   DEFAULT '{}' NOT NULL,
    indexer           TEXT,
    filter            TEXT,
    protocol          TEXT,
    implementation    TEXT,
    timestamp         TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    announce_type     TEXT      DEFAULT 'NEW', 
    info_url          TEXT,
    download_url      TEXT,
    group_id          TEXT,
    torrent_id        TEXT,
    torrent_name      TEXT,
    size              BIGINT,
    raw               TEXT,
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
    quality           TEXT,
    log_score         INTEGER,
    has_log           BOOLEAN,
    has_cue           BOOLEAN,
    is_scene          BOOLEAN,
    origin            TEXT,
    tags              TEXT []   DEFAULT '{}' NOT NULL,
    freeleech         BOOLEAN,
    freeleech_percent INTEGER,
    uploader          TEXT,
	pre_time          TEXT,
    filter_id         INTEGER
        CONSTRAINT release_filter_id_fk
            REFERENCES filter
            ON DELETE SET NULL
);

CREATE INDEX release_filter_id_index
    ON release (filter_id);

CREATE INDEX release_indexer_index
    ON "release" (indexer);

CREATE INDEX release_timestamp_index
    ON "release" (timestamp DESC);

CREATE INDEX release_torrent_name_index
    ON "release" (torrent_name);

CREATE TABLE release_action_status
(
	id            SERIAL PRIMARY KEY,
	status        TEXT,
	action        TEXT NOT NULL,
	action_id     INTEGER,
	type          TEXT NOT NULL,
	client        TEXT,
	filter        TEXT,
	filter_id     INTEGER,
	rejections    TEXT []   DEFAULT '{}' NOT NULL,
	timestamp     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	raw           TEXT,
	log           TEXT,
	release_id    INTEGER NOT NULL,
	FOREIGN KEY (action_id) REFERENCES "action"(id) ON DELETE SET NULL,
	FOREIGN KEY (release_id) REFERENCES "release"(id) ON DELETE CASCADE,
	FOREIGN KEY (filter_id) REFERENCES "filter"(id) ON DELETE SET NULL
);

CREATE INDEX release_action_status_release_id_index
    ON release_action_status (release_id);

CREATE TABLE notification
(
	id         SERIAL PRIMARY KEY,
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
	id            SERIAL PRIMARY KEY,
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
`

var postgresMigrations = []string{
	"",
	`
	CREATE TABLE notification
	(
		id         SERIAL PRIMARY KEY,
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
		id           SERIAL PRIMARY KEY,
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
	ALTER TABLE release
		RENAME COLUMN release_group TO "group";

	ALTER TABLE release
		DROP COLUMN raw;

	ALTER TABLE release
		DROP COLUMN audio;

	ALTER TABLE release
		DROP COLUMN region;

	ALTER TABLE release
		DROP COLUMN language;

	ALTER TABLE release
		DROP COLUMN edition;

	ALTER TABLE release
		DROP COLUMN unrated;

	ALTER TABLE release
		DROP COLUMN hybrid;

	ALTER TABLE release
		DROP COLUMN artists;

	ALTER TABLE release
		DROP COLUMN format;

	ALTER TABLE release
		DROP COLUMN quality;

	ALTER TABLE release
		DROP COLUMN log_score;

	ALTER TABLE release
		DROP COLUMN has_log;

	ALTER TABLE release
		DROP COLUMN has_cue;

	ALTER TABLE release
		DROP COLUMN is_scene;

	ALTER TABLE release
		DROP COLUMN freeleech;

	ALTER TABLE release
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
	ALTER TABLE release
		RENAME COLUMN "group" TO "release_group";

	ALTER TABLE release
    	ALTER COLUMN size TYPE BIGINT USING size::BIGINT;
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
	ALTER TABLE filter
		ADD max_downloads INTEGER default 0;

	ALTER TABLE filter
		ADD max_downloads_unit TEXT;

	ALTER TABLE release
		add filter_id INTEGER;

	CREATE INDEX release_filter_id_index
		ON release (filter_id);

	ALTER TABLE release
		ADD CONSTRAINT release_filter_id_fk
			FOREIGN KEY (filter_id) REFERENCES FILTER
				ON DELETE SET NULL;
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
	`ALTER TABLE irc_network
    RENAME COLUMN nickserv_account TO auth_account;

	ALTER TABLE irc_network
		RENAME COLUMN nickserv_password TO auth_password;

	ALTER TABLE irc_network
		ADD nick TEXT;

	ALTER TABLE irc_network
		ADD auth_mechanism TEXT DEFAULT 'SASL_PLAIN';

	ALTER TABLE irc_network
		DROP CONSTRAINT irc_network_server_port_nickserv_account_key;

	ALTER TABLE irc_network
		ADD CONSTRAINT irc_network_server_port_nick_key
			UNIQUE (server, port, nick);

	UPDATE irc_network
		SET nick = irc_network.auth_account;

	UPDATE irc_network
		SET auth_mechanism = 'SASL_PLAIN';`,
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
	`ALTER TABLE release_action_status
    ADD filter_id INTEGER;

CREATE INDEX release_action_status_filter_id_index
    ON release_action_status (filter_id);

ALTER TABLE release_action_status
    ADD CONSTRAINT release_action_status_filter_id_fk
        FOREIGN KEY (filter_id) REFERENCES filter;

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
	`ALTER TABLE release_action_status
    ADD action_id INTEGER;

ALTER TABLE release_action_status
    ADD CONSTRAINT release_action_status_action_id_fk
        FOREIGN KEY (action_id) REFERENCES action;`,
	`ALTER TABLE irc_network
ADD COLUMN use_bouncer BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
ADD COLUMN bouncer_addr TEXT;`,
	`ALTER TABLE release_action_status
    DROP CONSTRAINT IF EXISTS release_action_status_action_id_fkey;

ALTER TABLE release_action_status
    DROP CONSTRAINT IF EXISTS release_action_status_action_id_fk;

ALTER TABLE release_action_status
    ADD FOREIGN KEY (action_id) REFERENCES action
        ON DELETE SET NULL;
	`,
	`CREATE TABLE filter_external
	(
	id                      SERIAL PRIMARY KEY,
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

	ALTER TABLE filter
		DROP COLUMN IF EXISTS external_script_enabled;

	ALTER TABLE filter
		DROP COLUMN IF EXISTS external_script_cmd;

	ALTER TABLE filter
		DROP COLUMN IF EXISTS external_script_args;

	ALTER TABLE filter
		DROP COLUMN IF EXISTS external_script_expect_status;

	ALTER TABLE filter
		DROP COLUMN IF EXISTS external_webhook_enabled;

	ALTER TABLE filter
		DROP COLUMN IF EXISTS external_webhook_host;

	ALTER TABLE filter
		DROP COLUMN IF EXISTS external_webhook_data;

	ALTER TABLE filter
		DROP COLUMN IF EXISTS external_webhook_expect_status;
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
	`ALTER TABLE filter_external
    RENAME COLUMN external_webhook_retry_status TO webhook_retry_status;

ALTER TABLE filter_external
    RENAME COLUMN external_webhook_retry_attempts TO webhook_retry_attempts;

ALTER TABLE filter_external
    RENAME COLUMN external_webhook_retry_delay_seconds TO webhook_retry_delay_seconds;

ALTER TABLE filter_external
    RENAME COLUMN external_webhook_retry_max_jitter_seconds TO webhook_retry_max_jitter_seconds;
`,
	`ALTER TABLE filter_external
	DROP COLUMN IF EXISTS webhook_retry_max_jitter_seconds;
`,
	`ALTER TABLE irc_network
		ADD COLUMN bot_mode BOOLEAN DEFAULT FALSE;
`,
	`ALTER TABLE feed
	ALTER COLUMN max_age SET DEFAULT 0;
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
	`ALTER TABLE "release"
	ALTER COLUMN timestamp TYPE timestamptz USING timestamp AT TIME ZONE 'UTC';
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
    id             SERIAL PRIMARY KEY,
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
    ADD COLUMN proxy_id INTEGER;

ALTER TABLE indexer
    ADD COLUMN use_proxy BOOLEAN DEFAULT FALSE;

ALTER TABLE indexer
    ADD FOREIGN KEY (proxy_id) REFERENCES proxy
        ON DELETE SET NULL;

ALTER TABLE irc_network
    ADD COLUMN proxy_id INTEGER;

ALTER TABLE irc_network
    ADD COLUMN use_proxy BOOLEAN DEFAULT FALSE;

ALTER TABLE irc_network
    ADD FOREIGN KEY (proxy_id) REFERENCES proxy
        ON DELETE SET NULL;
`,
	`UPDATE indexer
	SET base_url = 'https://fuzer.xyz/'
	WHERE base_url = 'https://fuzer.me/';
`,
	`CREATE INDEX filter_enabled_index
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
}
