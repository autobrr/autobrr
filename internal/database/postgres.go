// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"cmp"
	"database/sql"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"

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
	migrate := migrations.PostgresMigrations(db.Handler, db.log)

	err := migrate.Migrate()
	if err != nil {
		return errors.Wrap(err, "could not migrate postgres database")
	}

	return nil
}

// PostgresDSN builds a libpq keyword/value connection string for lib/pq.
func PostgresDSN(host string, port int, user, pass, database, socket, sslMode, extraParams string) (string, error) {
	if database == "" {
		return "", errors.New("postgres: database name is required")
	}

	if socket == "" && host == "" && port == 0 {
		return "", errors.New("postgres: host and port are required for TCP connection")
	}

	var kvs []string
	emitted := map[string]bool{}
	add := func(key, value string) {
		if value == "" {
			return
		}
		emitted[key] = true
		kvs = append(kvs, key+"="+quoteDSNValue(value))
	}

	if socket != "" {
		add("host", socket)
	} else {
		add("host", host)
	}
	if port > 0 {
		add("port", strconv.Itoa(port))
	}
	add("dbname", database)
	add("user", user)
	add("password", pass)
	add("sslmode", sslMode)

	if extraParams != "" {
		values, err := url.ParseQuery(extraParams)
		if err != nil {
			return "", errors.Wrap(err, "could not parse extra params")
		}
		for _, key := range slices.Sorted(maps.Keys(values)) {
			if key == "" || strings.ContainsRune(key, '=') || strings.ContainsFunc(key, unicode.IsSpace) {
				return "", errors.New("postgres: invalid extra param name %q", key)
			}
			if emitted[cmp.Or(dsnKeyAliases[key], key)] {
				return "", errors.New("postgres: extra param %q is already set by a dedicated setting", key)
			}
			add(key, values.Get(key))
		}
	}

	return strings.Join(kvs, " "), nil
}

var dsnKeyAliases = map[string]string{"database": "dbname"}

var dsnValueEscaper = strings.NewReplacer(`\`, `\\`, `'`, `\'`)

func quoteDSNValue(value string) string {
	return "'" + dsnValueEscaper.Replace(value) + "'"
}
