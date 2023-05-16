// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"bufio"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/argon2id"
	"github.com/autobrr/autobrr/pkg/errors"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/term"
)

const usage = `usage: autobrrctl --config path <action>

  create-user		<username>	Create user
  change-password	<username>	Change password for user
  version				Can be run without --config
  help					Show this help message

`

var (
	version = "dev"
	commit  = ""
	date    = ""

	owner = "autobrr"
	repo  = "autobrr"
)

//const sqliteSchema = `...` // Your SQLite schema here

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
    connected           BOOLEAN,
    connected_since     TIMESTAMP,
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
    id                SERIAL PRIMARY KEY,
    filter_status     TEXT,
    rejections        TEXT []   DEFAULT '{}' NOT NULL,
    indexer           TEXT,
    filter            TEXT,
    protocol          TEXT,
    implementation    TEXT,
    timestamp         TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
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
	FOREIGN KEY (action_id) REFERENCES "action"(id),
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

func init() {
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), usage)
	}
}

func migrate(sqliteDBPath, postgresDBURL string) {
	// Connect to SQLite database
	sqliteDB, err := sql.Open("sqlite3", sqliteDBPath)
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}
	defer sqliteDB.Close()

	// Connect to PostgreSQL database
	postgresDB, err := sql.Open("postgres", postgresDBURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL database: %v", err)
	}
	defer postgresDB.Close()

	// Create PostgreSQL schema
	_, err = postgresDB.Exec(postgresSchema)
	if err != nil {
		log.Fatalf("Failed to create PostgreSQL schema: %v", err)
	}

	// List of table names to migrate
	tables := []string{
		"users", "indexer", "irc_network", "irc_channel", "filter", "filter_indexer", "client", "action", "release", "release_action_status", "notification", "feed", "feed_cache", "api_key",
	}

	for _, table := range tables {
		// Get all rows from the SQLite table
		rows, err := sqliteDB.Query(fmt.Sprintf("SELECT * FROM %s", table))
		if err != nil {
			log.Fatalf("Failed to query SQLite table '%s': %v", table, err)
		}

		// Get column names and types
		columns, err := rows.ColumnTypes()
		if err != nil {
			log.Fatalf("Failed to get column types for table '%s': %v", table, err)
		}

		// Prepare an INSERT statement for the PostgreSQL table
		colNames := ""
		colPlaceholders := ""
		for i, col := range columns {
			colNames += col.Name()
			colPlaceholders += fmt.Sprintf("$%d", i+1)
			if i < len(columns)-1 {
				colNames += ", "
				colPlaceholders += ", "
			}
		}
		insertStmt, err := postgresDB.Prepare(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, colNames, colPlaceholders))
		if err != nil {
			log.Fatalf("Failed to prepare INSERT statement for table '%s': %v", table, err)
		}

		// Iterate through SQLite rows and insert them into the PostgreSQL table
		for rows.Next() {
			values := make([]interface{}, len(columns))
			valuePtrs := make([]interface{}, len(columns))
			for i := range values {
				valuePtrs[i] = &values[i]
			}

			err = rows.Scan(valuePtrs...)
			if err != nil {
				log.Fatalf("Failed to scan row from SQLite table '%s': %v", table, err)
			}

			_, err = insertStmt.Exec(values...)
			if err != nil {
				log.Fatalf("Failed to insert row into PostgreSQL table '%s': %v", table, err)
			}
		}

		fmt.Printf("Migrated table '%s' from SQLite to PostgreSQL\n", table)
	}

	fmt.Println("Migration completed successfully!")
}

func resetDB(configPath string) {
	// Open the existing SQLite database
	dbPath := filepath.Join(filepath.Dir(configPath), "autobrr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Update the tables list with the provided table names
	tables := []string{
		"action",
		"api_key",
		"client",
		"feed",
		"feed_cache",
		"filter",
		"filter_indexer",
		"indexer",
		"irc_channel",
		"irc_network",
		"notification",
		"release",
		"release_action_status",
		"users",
	}

	// Execute SQL commands to remove all rows and reset primary key sequences
	for _, table := range tables {
		_, err = db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("failed to delete rows from table %s: %v", table, err)
		}

		// Attempt to update sqlite_sequence, ignore errors caused by missing sqlite_sequence entry
		_, err = db.Exec(fmt.Sprintf("UPDATE sqlite_sequence SET seq = 0 WHERE name = '%s'", table))
		if err != nil && !strings.Contains(err.Error(), "no such table") {
			log.Printf("failed to reset primary key sequence for table %s: %v", table, err)
		}
	}
}

func seedDB(seedDBPath string, configPath string) {
	// Read SQL file
	sqlFile, err := ioutil.ReadFile(seedDBPath)
	if err != nil {
		log.Fatalf("failed to read SQL file: %v", err)
	}

	// Create a new SQLite database
	dbPath := filepath.Join(filepath.Dir(configPath), "autobrr.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		log.Fatalf("failed to create database: %v", err)
	}
	defer db.Close()

	// Execute SQL commands from the file
	sqlCommands := strings.Split(string(sqlFile), ";")
	for _, cmd := range sqlCommands {
		_, err = db.Exec(cmd)
		if err != nil {
			log.Printf("failed to execute SQL command: %v", err)
		}
	}
}

func main() {
	var seedDBPath string
	flag.StringVar(&seedDBPath, "seed-db", "", "path to SQL seed file")

	var configPath string
	flag.StringVar(&configPath, "config", "", "path to configuration file")
	flag.Parse()

	switch cmd := flag.Arg(0); cmd {

	case "db:migrate":
		sqliteDBPath := flag.Arg(1)
		postgresDBURL := flag.Arg(2)

		if sqliteDBPath == "" || postgresDBURL == "" {
			flag.Usage()
			os.Exit(1)
		}

		migrate(sqliteDBPath, postgresDBURL)

	case "db:seed":
		seedDBPath := flag.Arg(1)
		if seedDBPath == "" {
			fmt.Println("Error: missing path to SQL seed file")
			flag.Usage()
			os.Exit(1)
		}
		seedDB(seedDBPath, configPath)
		fmt.Println("Database seeding completed successfully!")

	case "db:reset":
		if configPath == "" {
			log.Fatal("--config required")
		}
		seedDBPath := flag.Arg(1)
		if seedDBPath == "" {
			fmt.Println("Error: missing path to SQL seed file")
			flag.Usage()
			os.Exit(1)
		}
		resetDB(configPath)
		seedDB(seedDBPath, configPath)
		fmt.Println("Database reset completed successfully!")

	case "version":
		fmt.Printf("Version: %v\nCommit: %v\nBuild: %v\n", version, commit, date)

		// get the latest release tag from brr-api
		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Get(fmt.Sprintf("https://api.autobrr.com/repos/%s/%s/releases/latest", owner, repo))
		if err != nil {
			if errors.Is(err, http.ErrHandlerTimeout) {
				fmt.Println("Server timed out while fetching latest release from api")
			} else {
				fmt.Printf("Failed to fetch latest release from api: %v\n", err)
			}
			os.Exit(1)
		}
		defer resp.Body.Close()

		// brr-api returns 500 instead of 404 here
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusInternalServerError {
			fmt.Printf("No release found for %s/%s\n", owner, repo)
			os.Exit(1)
		}

		var rel struct {
			TagName string `json:"tag_name"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
			fmt.Printf("Failed to decode response from api: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Latest release: %v\n", rel.TagName)

	case "create-user":

		if configPath == "" {
			log.Fatal("--config required")
		}

		// read config
		cfg := config.New(configPath, version)

		// init new logger
		l := logger.New(cfg.Config)

		// open database connection
		db, _ := database.NewDB(cfg.Config, l)
		if err := db.Open(); err != nil {
			log.Fatal("could not open db connection")
		}

		userRepo := database.NewUserRepo(l, db)

		username := flag.Arg(1)
		if username == "" {
			flag.Usage()
			os.Exit(1)
		}

		password, err := readPassword()
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}
		hashed, err := argon2id.CreateHash(string(password), argon2id.DefaultParams)
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}

		user := domain.CreateUserRequest{
			Username: username,
			Password: hashed,
		}
		if err := userRepo.Store(context.Background(), user); err != nil {
			log.Fatalf("failed to create user: %v", err)
		}
	case "change-password":

		if configPath == "" {
			log.Fatal("--config required")
		}

		// read config
		cfg := config.New(configPath, version)

		// init new logger
		l := logger.New(cfg.Config)

		// open database connection
		db, _ := database.NewDB(cfg.Config, l)
		if err := db.Open(); err != nil {
			log.Fatal("could not open db connection")
		}

		userRepo := database.NewUserRepo(l, db)

		username := flag.Arg(1)
		if username == "" {
			flag.Usage()
			os.Exit(1)
		}

		user, err := userRepo.FindByUsername(context.Background(), username)
		if err != nil {
			log.Fatalf("failed to get user: %v", err)
		}

		if user == nil {
			log.Fatalf("failed to get user: %v", err)
		}

		password, err := readPassword()
		if err != nil {
			log.Fatalf("failed to read password: %v", err)
		}
		hashed, err := argon2id.CreateHash(string(password), argon2id.DefaultParams)
		if err != nil {
			log.Fatalf("failed to hash password: %v", err)
		}

		user.Password = hashed
		if err := userRepo.Update(context.Background(), *user); err != nil {
			log.Fatalf("failed to create user: %v", err)
		}
	default:
		flag.Usage()
		if cmd != "help" {
			os.Exit(1)
		}
	}
}

func readPassword() ([]byte, error) {
	var password []byte
	var err error
	fd := int(os.Stdin.Fd())

	if term.IsTerminal(fd) {
		fmt.Printf("Password: ")
		password, err = term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, err
		}
		fmt.Printf("\n")
	} else {
		//fmt.Fprintf(os.Stderr, "warning: Reading password from stdin.\n")
		scanner := bufio.NewScanner(os.Stdin)
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				log.Fatalf("failed to read password from stdin: %v", err)
			}
			log.Fatalf("failed to read password from stdin: stdin is empty %v", err)
		}
		password = scanner.Bytes()

		if len(password) == 0 {
			return nil, errors.New("zero length password")
		}
	}

	return password, nil
}
