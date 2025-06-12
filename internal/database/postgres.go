// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"

	"github.com/autobrr/autobrr/pkg/errors"

	_ "github.com/lib/pq"
)

func (db *DB) openPostgres() error {
	var err error

	// open database connection
	if db.Handler, err = sql.Open("postgres", db.DSN); err != nil {
		db.log.Fatal().Err(err).Msg("could not open postgres connection")
		return errors.Wrap(err, "could not open postgres connection")
	}

	err = db.Handler.Ping()
	if err != nil {
		db.log.Fatal().Err(err).Msg("could not ping postgres database")
		return errors.Wrap(err, "could not ping postgres database")
	}

	// migrate db
	if err = db.migratePostgres(); err != nil {
		db.log.Fatal().Err(err).Msg("could not migrate postgres database")
		return errors.Wrap(err, "could not migrate postgres database")
	}

	return nil
}

func (db *DB) migratePostgres() error {
	tx, err := db.Handler.Begin()
	if err != nil {
		return errors.Wrap(err, "error starting transaction")
	}
	defer tx.Rollback()

	initialSchema := `CREATE TABLE IF NOT EXISTS schema_migrations (
	id INTEGER PRIMARY KEY,
	version INTEGER NOT NULL
);`

	if _, err := tx.Exec(initialSchema); err != nil {
		return errors.New("failed to create schema_migrations table: %v", err.Error())
	}

	var version int
	err = tx.QueryRow(`SELECT version FROM schema_migrations`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "failed to query schema version")
	}

	if version == len(postgresMigrations) {
		return nil
	} else if version > len(postgresMigrations) {
		return errors.New("autobrr (version %d) older than schema (version: %d)", len(postgresMigrations), version)
	}

	db.log.Info().Msgf("Beginning database schema upgrade from version %v to version: %v", version, len(postgresMigrations))

	if version == 0 {
		if _, err := tx.Exec(postgresSchema); err != nil {
			return errors.Wrap(err, "failed to initialize schema")
		}
	} else {
		for i := version; i < len(postgresMigrations); i++ {
			db.log.Info().Msgf("Upgrading Database schema to version: %v", i+1)
			if _, err := tx.Exec(postgresMigrations[i]); err != nil {
				return errors.Wrap(err, "failed to execute migration #%v", i)
			}
		}
	}

	_, err = tx.Exec(`INSERT INTO schema_migrations (id, version) VALUES (1, $1) ON CONFLICT (id) DO UPDATE SET version = $1`, len(postgresMigrations))
	if err != nil {
		return errors.Wrap(err, "failed to bump schema version")
	}

	db.log.Info().Msgf("Database schema upgraded to version: %v", len(postgresMigrations))

	return tx.Commit()
}

// PostgresDSN build postgres dsn connect string
func PostgresDSN(host string, port int, user, pass, database, socket, sslMode, extraParams string) (string, error) {
	// If no database is provided, return an error
	if database == "" {
		return "", errors.New("postgres: database name is required")
	}

	pgDsn, err := url.Parse("postgres://")
	if err != nil {
		return "", errors.Wrap(err, "could not parse postgres DSN")
	}

	pgDsn.Path = database
	if user != "" {
		pgDsn.User = url.UserPassword(user, pass)
	}
	queryParams := pgDsn.Query()

	// Build DSN based on the connection type (TCP vs. Unix socket)
	if socket != "" {
		// Unix socket connection via the host param
		queryParams.Add("host", socket)
	} else {
		// TCP connection
		if host == "" && port == 0 {
			return "", errors.New("postgres: host and port are required for TCP connection")
		}
		if port > 0 {
			pgDsn.Host = net.JoinHostPort(host, fmt.Sprintf("%d", port))
		} else {
			pgDsn.Host = database
		}
	}

	// Add SSL mode if provided
	if sslMode != "" {
		queryParams.Add("sslmode", sslMode)
	}

	pgDsn.RawQuery = queryParams.Encode()

	// Add any extra parameters
	if extraParams != "" {
		values, err := url.ParseQuery(extraParams)
		if err != nil {
			return "", errors.Wrap(err, "could not parse extra params")
		}
		pgDsn.RawQuery = fmt.Sprintf("%s&%s", pgDsn.RawQuery, values.Encode())
	}

	return pgDsn.String(), nil
}
