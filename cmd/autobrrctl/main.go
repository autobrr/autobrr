// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
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

const usage = `usage: autobrrctl <action> [options]

Actions:
  create-user          <username>                                                        Create a new user
  change-password      <username>                                                        Change the password
  db:seed              --db-path <path-to-database> --seed-db <path-to-sql-seed>         Seed the sqlite database
  db:reset             --db-path <path-to-database> --seed-db <path-to-sql-seed>         Reset the sqlite database
  db:migrate           --sqlite-db <path-to-sqlite-db> --postgres-url <postgres-db-url>  Migrate sqlite to postgres
  version                                                                                Display the version of autobrrctl
  help                                                                                   Show this help message

Examples:
  autobrrctl --config /path/to/config/dir create-user john
  autobrrctl --config /path/to/config/dir change-password john
  autobrrctl db:reset --db-path /path/to/autobrr.db --seed-db /path/to/seed
  autobrrctl db:seed --db-path /path/to/autobrr.db --seed-db /path/to/seed
  autobrrctl db:migrate --sqlite-db /path/to/autobrr.db --postgres-url postgres://username:password@127.0.0.1:5432/autobrr
  autobrrctl version
  autobrrctl help
`

var (
	version = "dev"
	commit  = ""
	date    = ""

	owner = "autobrr"
	repo  = "autobrr"
)

func init() {
	flag.Usage = func() {
		fmt.Fprint(flag.CommandLine.Output(), usage)
	}
}

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to configuration file")
	flag.Parse()

	switch cmd := flag.Arg(0); cmd {

	case "db:migrate":
		var sqliteDBPath, postgresDBURL string
		migrateFlagSet := flag.NewFlagSet("db:migrate", flag.ExitOnError)
		migrateFlagSet.StringVar(&sqliteDBPath, "sqlite-db", "", "path to SQLite database file")
		migrateFlagSet.StringVar(&postgresDBURL, "postgres-url", "", "URL for PostgreSQL database")

		if err := migrateFlagSet.Parse(flag.Args()[1:]); err != nil {
			fmt.Printf("Error parsing flags for db:migrate: %v\n", err)
			migrateFlagSet.Usage()
			os.Exit(1)
		}

		if sqliteDBPath == "" || postgresDBURL == "" {
			fmt.Println("Error: missing required flags for db:migrate")
			flag.Usage()
			os.Exit(1)
		}

		if err := database.Migrate(sqliteDBPath, postgresDBURL); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}

	case "db:seed", "db:reset":
		var dbPath, seedDBPath string
		seedResetFlagSet := flag.NewFlagSet("db:seed/db:reset", flag.ExitOnError)
		seedResetFlagSet.StringVar(&dbPath, "db-path", "", "path to the database file")
		seedResetFlagSet.StringVar(&seedDBPath, "seed-db", "", "path to SQL seed file")

		if err := seedResetFlagSet.Parse(flag.Args()[1:]); err != nil {
			fmt.Printf("Error parsing flags for db:seed or db:reset: %v\n", err)
			seedResetFlagSet.Usage()
			os.Exit(1)
		}

		if dbPath == "" || seedDBPath == "" {
			fmt.Println("Error: missing required flags for db:seed or db:reset")
			flag.Usage()
			os.Exit(1)
		}

		if cmd == "db:seed" {
			err := database.SeedDB(dbPath, seedDBPath)
			if err != nil {
				fmt.Println("Error seeding the database:", err)
				os.Exit(1)
			}
			fmt.Println("Database seeding completed successfully!")
		} else {
			err := database.ResetDB(dbPath)
			if err != nil {
				fmt.Println("Error resetting the database:", err)
				os.Exit(1)
			}
			err = database.SeedDB(dbPath, seedDBPath)
			if err != nil {
				fmt.Println("Error seeding the database:", err)
				os.Exit(1)
			}
			fmt.Println("Database reset and reseed completed successfully!")
		}

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
