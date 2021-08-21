package database

import (
	"database/sql"
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
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE irc_network
(
    id                  INTEGER PRIMARY KEY,
    enabled             BOOLEAN,
    name                TEXT NOT NULL,
    addr                TEXT NOT NULL,
    nick                TEXT NOT NULL,
    tls                 BOOLEAN,
    pass                TEXT,
    connect_commands    TEXT,
    sasl_mechanism      TEXT,
    sasl_plain_username TEXT,
    sasl_plain_password TEXT,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    unique (addr, nick)
);

CREATE TABLE irc_channel
(
    id       INTEGER PRIMARY KEY,
    enabled  BOOLEAN,
    name     TEXT NOT NULL,
    password TEXT,
    detached BOOLEAN,
    network_id  INTEGER NOT NULL,
    FOREIGN KEY (network_id) REFERENCES irc_network(id),
    unique (network_id, name)
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
`

var migrations = []string{
	"",
}

func Migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("failed to query schema version: %v", err)
	}

	if version == len(migrations) {
		return nil
	} else if version > len(migrations) {
		return fmt.Errorf("autobrr (version %d) older than schema (version: %d)", len(migrations), version)
	}

	tx, err := db.Begin()
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
