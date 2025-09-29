// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"net"
	"net/url"

	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/pkg/errors"

	_ "github.com/lib/pq"
)

func (db *DB) openPostgres() error {
	var err error

	// open database connection
	if db.Handler, err = sql.Open("postgres", db.DSN); err != nil {
		return errors.Wrap(err, "could not open postgres connection")
	}

	err = db.Handler.Ping()
	if err != nil {
		return errors.Wrap(err, "could not ping postgres database")
	}

	// migrate db
	if db.cfg.DatabaseAutoMigrate {
		if err = db.migratePostgres(); err != nil {
			return errors.Wrap(err, "could not migrate postgres database")
		}
	}

	return nil
}

func (db *DB) migratePostgres() error {
	migrate := migrations.PostgresMigrations(db.Handler)

	err := migrate.Migrate()
	if err != nil {
		return errors.Wrap(err, "could not migrate postgres database")
	}

	return nil
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
