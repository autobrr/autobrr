package database

import (
	"fmt"
)

const schema = `
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
    id         INTEGER PRIMARY KEY,
    identifier TEXT,
    enabled    BOOLEAN,
    name       TEXT NOT NULL,
    settings   TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (identifier)
);

CREATE TABLE irc_network
(
    id                  INTEGER PRIMARY KEY,
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
    id                    INTEGER PRIMARY KEY,
    enabled               BOOLEAN,
    name                  TEXT NOT NULL,
    min_size              TEXT,
    max_size              TEXT,
    delay                 INTEGER,
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
    years                 TEXT,
    match_categories      TEXT,
    except_categories     TEXT,
    match_uploaders       TEXT,
    except_uploaders      TEXT,
    tags                  TEXT,
    except_tags           TEXT,
    created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE filter_indexer
(
    filter_id  INTEGER,
    indexer_id INTEGER,
    FOREIGN KEY (filter_id) REFERENCES filter(id),
    FOREIGN KEY (indexer_id) REFERENCES indexer(id),
    PRIMARY KEY (filter_id, indexer_id)
);

CREATE TABLE client
(
    id       INTEGER PRIMARY KEY,
    name     TEXT NOT NULL,
    enabled  BOOLEAN,
    type     TEXT,
    host     TEXT NOT NULL,
    port     INTEGER,
    ssl      BOOLEAN,
    username TEXT,
    password TEXT,
    settings JSON
);

CREATE TABLE action
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
    client_id            INTEGER,
    filter_id            INTEGER,
    FOREIGN KEY (client_id) REFERENCES client(id),
    FOREIGN KEY (filter_id) REFERENCES filter(id)
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
`

var migrations = []string{
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
}

func (db *SqliteDB) migrate() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	var version int
	if err := db.handler.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("failed to query schema version: %v", err)
	}

	if version == len(migrations) {
		return nil
	} else if version > len(migrations) {
		return fmt.Errorf("autobrr (version %d) older than schema (version: %d)", len(migrations), version)
	}

	tx, err := db.handler.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if version == 0 {
		if _, err := tx.Exec(schema); err != nil {
			return fmt.Errorf("failed to initialize schema: %v", err)
		}
	} else {
		for i := version; i < len(migrations); i++ {
			if _, err := tx.Exec(migrations[i]); err != nil {
				return fmt.Errorf("failed to execute migration #%v: %v", i, err)
			}
		}
	}

	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", len(migrations)))
	if err != nil {
		return fmt.Errorf("failed to bump schema version: %v", err)
	}

	return tx.Commit()
}
