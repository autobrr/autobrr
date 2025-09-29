// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/lib/pq"
	_ "modernc.org/sqlite"
)

const badFormat = "2006-01-02T15:04:05"
const timeFormat = "2006-01-02.15-04-05"

func (db *DB) openSQLite() error {
	if db.DSN == "" {
		return errors.New("DSN required")
	}

	var err error

	// open database connection
	if db.Handler, err = sql.Open("sqlite", db.DSN+"?_pragma=busy_timeout%3d1000"); err != nil {
		db.log.Fatal().Err(err).Msg("could not open db connection")
		return err
	}

	// Set busy timeout
	if _, err = db.Handler.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return errors.Wrap(err, "busy timeout pragma")
	}

	// Enable WAL. SQLite performs better with the WAL  because it allows
	// multiple readers to operate while data is being written.
	if _, err = db.Handler.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return errors.Wrap(err, "enable wal")
	}

	// SQLite has a query planner that uses lifecycle stats to fund optimizations.
	// This restricts the SQLite query planner optimizer to only run if sufficient
	// information has been gathered over the lifecycle of the connection.
	// The SQLite documentation is inconsistent in this regard,
	// suggestions of 400 and 1000 are both "recommended", so lets use the lower bound.
	if _, err = db.Handler.Exec(`PRAGMA analysis_limit = 400;`); err != nil {
		return errors.Wrap(err, "analysis_limit")
	}

	// When Autobrr does not cleanly shutdown, the WAL will still be present and not committed.
	// This is a no-op if the WAL is empty, and a commit when the WAL is not to start fresh.
	// When commits hit 1000, PRAGMA wal_checkpoint(PASSIVE); is invoked which tries its best
	// to commit from the WAL (and can fail to commit all pending operations).
	// Forcing a PRAGMA wal_checkpoint(RESTART); in the future on a "quiet period" could be
	// considered.
	if _, err = db.Handler.Exec(`PRAGMA wal_checkpoint(TRUNCATE);`); err != nil {
		return errors.Wrap(err, "commit wal")
	}

	// Enable foreign key checks. For historical reasons, SQLite does not check
	// foreign key constraints by default. There's some overhead on inserts to
	// verify foreign key integrity, but it's definitely worth it.

	// Enable it for testing for consistency with postgres.
	if os.Getenv("IS_TEST_ENV") == "true" {
		if _, err = db.Handler.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
			return errors.New("foreign keys pragma")
		}
	}

	//if _, err = db.Handler.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
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
	if db.Handler == nil {
		return nil
	}

	// SQLite has a query planner that uses lifecycle stats to fund optimizations.
	// Based on the limit defined at connection time, run optimize to
	// help tweak the performance of the database on the next run.
	if _, err := db.Handler.Exec(`PRAGMA optimize;`); err != nil {
		return errors.Wrap(err, "query planner optimization")
	}

	return nil
}

func (db *DB) migrateSQLite() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	var version int
	if err := db.Handler.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return errors.Wrap(err, "failed to query schema version")
	}

	if version == len(sqliteMigrations) {
		return nil
	} else if version > len(sqliteMigrations) {
		return errors.New("autobrr (version %d) older than schema (version: %d)", len(sqliteMigrations), version)
	}

	db.log.Info().Msgf("Beginning database schema upgrade from version %d to version: %d", version, len(sqliteMigrations))

	tx, err := db.Handler.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if version == 0 {
		if _, err := tx.Exec(sqliteSchema); err != nil {
			return errors.Wrap(err, "failed to initialize schema")
		}
	} else {
		if db.cfg.DatabaseMaxBackups > 0 {
			if err := db.databaseConsistencyCheckSQLite(); err != nil {
				return errors.Wrap(err, "database image malformed")
			}

			if err := db.backupSQLiteDatabase(); err != nil {
				return errors.Wrap(err, "failed to create database backup")
			}
		}

		for i := version; i < len(sqliteMigrations); i++ {
			db.log.Info().Msgf("Upgrading Database schema to version: %v", i+1)

			if _, err := tx.Exec(sqliteMigrations[i]); err != nil {
				return errors.Wrap(err, "failed to execute migration #%v", i)
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
	}

	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", len(sqliteMigrations)))
	if err != nil {
		return errors.Wrap(err, "failed to bump schema version")
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit migration transaction")
	}

	db.log.Info().Msgf("Database schema upgraded to version: %d", len(sqliteMigrations))

	if err := db.cleanupSQLiteBackups(); err != nil {
		return err
	}

	return nil
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

func (db *DB) databaseConsistencyCheckSQLite() error {
	db.log.Info().Msg("Database integrity check..")

	rows, err := db.Handler.Query("PRAGMA integrity_check;")
	if err != nil {
		return errors.Wrap(err, "failed to query integrity check")
	}

	var results []string
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return errors.Wrap(err, "backup integrity unexpected state")
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return errors.Wrap(err, "backup integrity unexpected state")
	}

	if len(results) == 1 && results[0] == "ok" {
		db.log.Info().Msg("Database integrity check OK!")
		return nil
	}

	if err := db.sqlitePerformReIndexing(results); err != nil {
		return errors.Wrap(err, "failed to reindex database")
	}

	db.log.Info().Msg("Database integrity check post re-indexing..")

	row := db.Handler.QueryRow("PRAGMA integrity_check;")

	var status string
	if err := row.Scan(&status); err != nil {
		return errors.Wrap(err, "backup integrity unexpected state")
	}

	db.log.Info().Msgf("Database integrity check: %s", status)

	if status != "ok" {
		return errors.New("backup integrity check failed: %q", status)
	}

	return nil
}

// sqlitePerformReIndexing try to reindex bad indexes
func (db *DB) sqlitePerformReIndexing(results []string) error {
	db.log.Warn().Msg("Database integrity check failed!")

	db.log.Info().Msg("Backing up database before re-indexing..")

	if err := db.backupSQLiteDatabase(); err != nil {
		return errors.Wrap(err, "failed to create database backup")
	}

	db.log.Info().Msg("Database backup created!")

	var badIndexes []string

	for _, issue := range results {
		index, found := strings.CutPrefix(issue, "wrong # of entries in index ")
		if found {
			db.log.Warn().Msgf("Database integrity check failed on index: %s", index)

			badIndexes = append(badIndexes, index)
		}
	}

	if len(badIndexes) == 0 {
		return errors.New("found no indexes to reindex")
	}

	for _, index := range badIndexes {
		db.log.Info().Msgf("Database attempt to re-index: %s", index)

		_, err := db.Handler.Exec(fmt.Sprintf("REINDEX %s;", index))
		if err != nil {
			return errors.Wrap(err, "failed to backup database")
		}
	}

	db.log.Info().Msg("Database re-indexing OK!")

	return nil
}

func (db *DB) backupSQLiteDatabase() error {
	var version int
	if err := db.Handler.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return errors.Wrap(err, "failed to query schema version")
	}

	backupFile := db.DSN + fmt.Sprintf("_sv%v_%s.backup", version, time.Now().UTC().Format(timeFormat))

	db.log.Info().Msgf("Creating database backup: %s", backupFile)

	_, err := db.Handler.Exec("VACUUM INTO ?;", backupFile)
	if err != nil {
		return errors.Wrap(err, "failed to backup database")
	}

	db.log.Info().Msgf("Database backup created at: %s", backupFile)

	return nil
}

func (db *DB) cleanupSQLiteBackups() error {
	if db.cfg.DatabaseMaxBackups == 0 {
		return nil
	}

	backupDir := filepath.Dir(db.DSN)

	files, err := os.ReadDir(backupDir)
	if err != nil {
		return errors.Wrap(err, "failed to read backup directory: %s", backupDir)
	}

	var backups []string
	var broken []string
	// Parse the filenames to extract timestamps
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".backup") {
			// Extract timestamp from filename
			parts := strings.Split(file.Name(), "_")
			if len(parts) < 3 {
				continue
			}
			timestamp := strings.TrimSuffix(parts[2], ".backup")
			if _, err := time.Parse(timeFormat, timestamp); err == nil {
				backups = append(backups, file.Name())
			} else if _, err := time.Parse(badFormat, timestamp); err == nil {
				broken = append(broken, file.Name())
			}
		}
	}

	db.log.Info().Msgf("Found %d SQLite backups", len(backups))

	if len(backups) == 0 {
		return nil
	}

	// Sort backups by timestamp
	sort.Slice(backups, func(i, j int) bool {
		t1, _ := time.Parse(timeFormat, strings.TrimSuffix(strings.Split(backups[i], "_")[2], ".backup"))
		t2, _ := time.Parse(timeFormat, strings.TrimSuffix(strings.Split(backups[j], "_")[2], ".backup"))
		return t1.After(t2)
	})

	for i := 0; len(broken) != 0 && len(backups) == db.cfg.DatabaseMaxBackups && i < len(broken); i++ {
		db.log.Info().Msgf("Remove Old SQLite backup: %s", broken[i])

		if err := os.Remove(filepath.Join(backupDir, broken[i])); err != nil {
			return errors.Wrap(err, "failed to remove old backups")
		}

		db.log.Info().Msgf("Removed Old SQLite backup: %s", broken[i])
	}

	for i := db.cfg.DatabaseMaxBackups; i < len(backups); i++ {
		db.log.Info().Msgf("Remove SQLite backup: %s", backups[i])

		if err := os.Remove(filepath.Join(backupDir, backups[i])); err != nil {
			return errors.Wrap(err, "failed to remove old backups")
		}

		db.log.Info().Msgf("Removed SQLite backup: %s", backups[i])
	}

	return nil
}
