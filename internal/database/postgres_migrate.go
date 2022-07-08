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

CREATE TABLE indexer
(
    id             SERIAL PRIMARY KEY,
    identifier     TEXT,
	implementation TEXT,
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
    id                  SERIAL PRIMARY KEY,
    enabled             BOOLEAN,
    name                TEXT NOT NULL,
    server              TEXT NOT NULL,
    port                INTEGER NOT NULL,
    tls                 BOOLEAN,
    pass                TEXT,
    invite_command      TEXT,
    nickserv_account    TEXT,
    nickserv_password   TEXT,
    connected           BOOLEAN,
    connected_since     TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (server, port, nickserv_account)
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
    id                    SERIAL PRIMARY KEY,
    enabled               BOOLEAN,
    name                  TEXT NOT NULL,
    min_size              TEXT,
    max_size              TEXT,
    delay                 INTEGER,
    priority              INTEGER DEFAULT 0 NOT NULL,
    max_downloads         INTEGER DEFAULT 0,
    max_downloads_unit    TEXT,
    match_releases        TEXT,
    except_releases       TEXT,
    use_regex             BOOLEAN,
    match_release_groups  TEXT,
    except_release_groups TEXT,
    scene                 BOOLEAN,
    freeleech             BOOLEAN,
    freeleech_percent     TEXT,
    shows                 TEXT,
    seasons               TEXT,
    episodes              TEXT,
    resolutions           TEXT []   DEFAULT '{}' NOT NULL,
    codecs                TEXT []   DEFAULT '{}' NOT NULL,
    sources               TEXT []   DEFAULT '{}' NOT NULL,
    containers            TEXT []   DEFAULT '{}' NOT NULL,
    match_hdr             TEXT []   DEFAULT '{}',
    except_hdr            TEXT []   DEFAULT '{}',
    match_other           TEXT []   DEFAULT '{}',
    except_other          TEXT []   DEFAULT '{}',
    years                 TEXT,
    artists               TEXT,
    albums                TEXT,
    release_types_match   TEXT []   DEFAULT '{}',
    release_types_ignore  TEXT []   DEFAULT '{}',
    formats               TEXT []   DEFAULT '{}',
    quality               TEXT []   DEFAULT '{}',
	media 				  TEXT []   DEFAULT '{}',
    log_score             INTEGER,
    has_log               BOOLEAN,
    has_cue               BOOLEAN,
    perfect_flac          BOOLEAN,
    match_categories      TEXT,
    except_categories     TEXT,
    match_uploaders       TEXT,
    except_uploaders      TEXT,
    tags                  TEXT,
    except_tags           TEXT,
    origins               TEXT []   DEFAULT '{}',
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    id                SERIAL PRIMARY KEY,
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
    size              BIGINT,
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
}
