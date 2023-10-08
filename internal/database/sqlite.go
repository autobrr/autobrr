// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/lib/pq"
	_ "modernc.org/sqlite"
)

func (db *DB) openSQLite() error {
	if db.DSN == "" {
		return errors.New("DSN required")
	}

	var err error

	// open database connection
	if db.handler, err = sql.Open("sqlite", db.DSN+"?_pragma=busy_timeout%3d1000"); err != nil {
		db.log.Fatal().Err(err).Msg("could not open db connection")
		return err
	}

	// Set busy timeout
	//if _, err = db.handler.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
	//	return errors.New("busy timeout pragma: %w", err)
	//}

	// Enable WAL. SQLite performs better with the WAL  because it allows
	// multiple readers to operate while data is being written.
	if _, err = db.handler.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return errors.Wrap(err, "enable wal")
	}

	// SQLite has a query planner that uses lifecycle stats to fund optimizations.
	// This restricts the SQLite query planner optimizer to only run if sufficient 
	// information has been gathered over the lifecycle of the connection.
	// The SQLite documentation is inconsistent in this regard,
	// suggestions of 400 and 1000 are both "recommended", so lets use the lower bound.
	if _, err = db.handler.Exec(`PRAGMA analysis_limit = 400;`); err != nil {
		return errors.Wrap(err, "analysis_limit")
	}

	// When Autobrr does not cleanly shutdown, the WAL will still be present and not committed.
	// This is a no-op if the WAL is empty, and a commit when the WAL is not to start fresh.
	// When commits hit 1000, PRAGMA wal_checkpoint(PASSIVE); is invoked which tries its best
	// to commit from the WAL (and can fail to commit all pending operations).
	// Forcing a PRAGMA wal_checkpoint(RESTART); in the future on a "quiet period" could be
	// considered.
	if _, err = db.handler.Exec(`PRAGMA wal_checkpoint(TRUNCATE);`); err != nil {
		return errors.Wrap(err, "commit wal")
	}

	// Enable foreign key checks. For historical reasons, SQLite does not check
	// foreign key constraints by default. There's some overhead on inserts to
	// verify foreign key integrity, but it's definitely worth it.
	//if _, err = db.handler.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
	//	return errors.New("foreign keys pragma: %w", err)
	//}

	// migrate db
	if err = db.migrateSQLite(); err != nil {
		db.log.Fatal().Err(err).Msg("could not migrate db")
		return err
	}

	return nil
}

func (db *DB) closingSQLite() error {
	if db.handler == nil {
		return nil
	}

	// SQLite has a query planner that uses lifecycle stats to fund optimizations.
	// Based on the limit defined at connection time, run optimize to
	// help tweak the performance of the database on the next run.
	if _, err = db.handler.Exec(`PRAGMA optimize;`); err != nil {
		return errors.Wrap(err, "query planner optimization")
	}

	return nil
}

func (db *DB) migrateSQLite() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	var version int
	if err := db.handler.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return errors.Wrap(err, "failed to query schema version")
	}

	if version == len(sqliteMigrations) {
		return nil
	} else if version > len(sqliteMigrations) {
		return errors.New("autobrr (version %d) older than schema (version: %d)", len(sqliteMigrations), version)
	}

	db.log.Info().Msgf("Beginning database schema upgrade from version %v to version: %v", version, len(sqliteMigrations))

	tx, err := db.handler.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if version == 0 {
		if _, err := tx.Exec(sqliteSchema); err != nil {
			return errors.Wrap(err, "failed to initialize schema")
		}
	} else {
		for i := version; i < len(sqliteMigrations); i++ {
			db.log.Info().Msgf("Upgrading Database schema to version: %v", i)
			if _, err := tx.Exec(sqliteMigrations[i]); err != nil {
				return errors.Wrap(err, "failed to execute migration #%v", i)
			}
		}
	}

	// temp custom data migration
	// get data from filter.sources, check if specific types, move to new table and clear
	// if migration 6
	// TODO 2022-01-30 remove this in future version
	if version == 5 && len(sqliteMigrations) == 6 {
		if err := customMigrateCopySourcesToMedia(tx); err != nil {
			return errors.Wrap(err, "could not run custom data migration")
		}
	}

	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", len(sqliteMigrations)))
	if err != nil {
		return errors.Wrap(err, "failed to bump schema version")
	}

	db.log.Info().Msgf("Database schema upgraded to version: %v", len(sqliteMigrations))

	return tx.Commit()
}

// customMigrateCopySourcesToMedia move music specific sources to media
func customMigrateCopySourcesToMedia(tx *sql.Tx) error {
	rows, err := tx.Query(`
		SELECT id, sources
		FROM filter
		WHERE sources LIKE '%"CD"%'
		   OR sources LIKE '%"WEB"%'
		   OR sources LIKE '%"DVD"%'
		   OR sources LIKE '%"Vinyl"%'
		   OR sources LIKE '%"Soundboard"%'
		   OR sources LIKE '%"DAT"%'
		   OR sources LIKE '%"Cassette"%'
		   OR sources LIKE '%"Blu-Ray"%'
		   OR sources LIKE '%"SACD"%'
		;`)
	if err != nil {
		return errors.Wrap(err, "could not run custom data migration")
	}

	defer rows.Close()

	type tmpDataStruct struct {
		id      int
		sources []string
	}

	var tmpData []tmpDataStruct

	// scan data
	for rows.Next() {
		var t tmpDataStruct

		if err := rows.Scan(&t.id, pq.Array(&t.sources)); err != nil {
			return err
		}

		tmpData = append(tmpData, t)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// manipulate data
	for _, d := range tmpData {
		// create new slice with only music source if they exist in d.sources
		mediaSources := []string{}
		for _, source := range d.sources {
			switch source {
			case "CD":
				mediaSources = append(mediaSources, source)
			case "DVD":
				mediaSources = append(mediaSources, source)
			case "Vinyl":
				mediaSources = append(mediaSources, source)
			case "Soundboard":
				mediaSources = append(mediaSources, source)
			case "DAT":
				mediaSources = append(mediaSources, source)
			case "Cassette":
				mediaSources = append(mediaSources, source)
			case "Blu-Ray":
				mediaSources = append(mediaSources, source)
			case "SACD":
				mediaSources = append(mediaSources, source)
			}
		}
		_, err = tx.Exec(`UPDATE filter SET media = ? WHERE id = ?`, pq.Array(mediaSources), d.id)
		if err != nil {
			return err
		}

		// remove all music specific sources
		cleanSources := []string{}
		for _, source := range d.sources {
			switch source {
			case "CD", "WEB", "DVD", "Vinyl", "Soundboard", "DAT", "Cassette", "Blu-Ray", "SACD":
				continue
			}
			cleanSources = append(cleanSources, source)
		}
		_, err := tx.Exec(`UPDATE filter SET sources = ? WHERE id = ?`, pq.Array(cleanSources), d.id)
		if err != nil {
			return err
		}

	}

	return nil
}
