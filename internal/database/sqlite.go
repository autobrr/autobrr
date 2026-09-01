// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/pkg/errors"

	_ "modernc.org/sqlite"
)

const badFormat = "2006-01-02T15:04:05"
const timeFormat = "2006-01-02.15-04-05"

func (db *DB) openSQLite() error {
	if db.DSN == "" {
		return errors.New("DSN required")
	}

	var err error

	pragmaSlice := []string{
		// Set busy timeout to 10 seconds. It forces the driver to keep retrying a locked operation before giving up.
		"?_pragma=busy_timeout(10000)",

		// Enable Write-Ahead Logging (WAL). SQLite performs better with the WAL because it allows
		// multiple readers to operate while data is being written.
		"_pragma=journal_mode(WAL)",

		// Default is FULL. Reducing this to NORMAL saves a significant amount of disk I/O (fsyncs).
		"_pragma=synchronous(NORMAL)",

		// When Autobrr does not cleanly shut down, the WAL will still be present and not committed.
		// This is a no-op if the WAL is empty, and a commit when the WAL is not to start fresh.
		// When commits hit 1000, PRAGMA wal_checkpoint(PASSIVE); is invoked which tries its best
		// to commit from the WAL (and can fail to commit all pending operations).
		// Forcing a PRAGMA wal_checkpoint(RESTART); in the future on a "quiet period" could be
		// considered.
		//"_pragma=wal_checkpoint(TRUNCATE)",

		// SQLite has a query planner that uses lifecycle stats to fund optimizations.
		// This restricts the SQLite query planner optimizer to only run if sufficient
		// information has been gathered over the lifecycle of the connection.
		// The SQLite documentation is inconsistent in this regard,
		// suggestions of 400 and 1000 are both "recommended", so lets use the lower bound.
		"_pragma=analysis_limit(400)",

		// Memory-mapping the first 256MB of the database
		// allows SQLite to read the file directly from memory, bypassing system call overhead.
		"_pragma=mmap_size(268435456)",

		// The default cache size is usually small (~2MB).
		// Bumping this to 64MB reduces disk reads significantly
		"_pragma=cache_size(-64000)",

		"_pragma=page_size(4096)",
	}

	if os.Getenv("IS_TEST_ENV") == "true" {
		// Enable foreign key checks. For historical reasons, SQLite does not check
		// foreign key constraints by default. There's some overhead on inserts to
		// verify foreign key integrity, but it's definitely worth it.

		// Enable it for testing for consistency with postgres.
		pragmaSlice = append(pragmaSlice, "_pragma=foreign_keys(ON)")
	}

	pragmas := strings.Join(pragmaSlice, "&")

	// open database connection
	if db.Handler, err = sql.Open("sqlite", db.DSN+pragmas); err != nil {
		return errors.Wrap(err, "could not open db connection")
	}

	db.Handler.SetMaxOpenConns(1)
	db.Handler.SetMaxIdleConns(5)
	db.Handler.SetConnMaxLifetime(0)

	// migrate db
	if db.cfg.DatabaseAutoMigrate {
		if err = db.migrateSQLite(); err != nil {
			return errors.Wrap(err, "could not migrate db")
		}
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
	migrate := migrations.SQLiteMigrations(db.Handler, db.log)
	migrate.PreMigrationHook = func() error {
		if db.cfg.DatabaseMaxBackups > 0 {
			if err := db.databaseConsistencyCheckSQLite(); err != nil {
				return errors.Wrap(err, "database image malformed")

			}

			if err := db.backupSQLiteDatabase(); err != nil {
				return errors.Wrap(err, "failed to create database backup")
			}
		}

		return nil
	}

	if err := migrate.Migrate(); err != nil {
		return errors.Wrap(err, "failed to migrate database")
	}

	if err := db.cleanupSQLiteBackups(); err != nil {
		return err
	}

	return nil
}

// startProgressLogger logs a heartbeat until the returned stop func is called,
// so long-running maintenance operations do not look like a hang at startup.
func (db *DB) startProgressLogger(op string) func() {
	start := time.Now()
	done := make(chan struct{})

	go func() {
		tick := time.Tick(30 * time.Second)

		for {
			select {
			case <-done:
				return
			case <-tick:
				db.log.Info().Msgf("%s still running, elapsed: %s", op, time.Since(start).Round(time.Second))
			}
		}
	}()

	return func() { close(done) }
}

func (db *DB) databaseConsistencyCheckSQLite() error {
	db.log.Info().Msg("Database integrity check..")

	results, err := db.sqliteIntegrityCheck()
	if err != nil {
		return err
	}

	if len(results) == 1 && results[0] == "ok" {
		db.log.Info().Msg("Database integrity check OK!")
		return nil
	}

	if err := db.sqlitePerformReIndexing(results); err != nil {
		return errors.Wrap(err, "failed to reindex database")
	}

	db.log.Info().Msg("Database integrity check post re-indexing..")

	results, err = db.sqliteIntegrityCheck()
	if err != nil {
		return err
	}

	status := strings.Join(results, "; ")

	db.log.Info().Msgf("database integrity check status: %s", status)

	if len(results) != 1 || results[0] != "ok" {
		return errors.New("backup integrity check failed: %q", status)
	}

	return nil
}

func (db *DB) sqliteIntegrityCheck() ([]string, error) {
	stop := db.startProgressLogger("database integrity check")
	defer stop()

	rows, err := db.Handler.Query("PRAGMA integrity_check;")
	if err != nil {
		return nil, errors.Wrap(err, "failed to query integrity check")
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return nil, errors.Wrap(err, "backup integrity unexpected state")
		}

		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "backup integrity unexpected state")
	}

	return results, nil
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
			db.log.Warn().Str("index", index).Msg("database integrity check failed on index")

			badIndexes = append(badIndexes, index)
		}
	}

	if len(badIndexes) == 0 {
		return errors.New("found no indexes to reindex")
	}

	stop := db.startProgressLogger("database re-indexing")
	defer stop()

	for _, index := range badIndexes {
		db.log.Info().Str("index", index).Msg("database attempt to re-index")

		_, err := db.Handler.Exec(fmt.Sprintf("REINDEX %s;", index))
		if err != nil {
			return errors.Wrap(err, "could not reindex %s", index)
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

	db.log.Info().Str("path", backupFile).Msg("creating database backup")

	stop := db.startProgressLogger("database backup")
	_, err := db.Handler.Exec("VACUUM INTO ?;", backupFile)
	stop()

	if err != nil {
		return errors.Wrap(err, "failed to backup database")
	}

	db.log.Info().Str("path", backupFile).Msg("database backup created")

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

	db.log.Info().Int("count", len(backups)).Msg("found SQLite backups")

	if len(backups) == 0 {
		return nil
	}

	// Sort backups by timestamp
	slices.SortFunc(backups, func(a, b string) int {
		ta, _ := time.Parse(timeFormat, strings.TrimSuffix(strings.Split(a, "_")[2], ".backup"))
		tb, _ := time.Parse(timeFormat, strings.TrimSuffix(strings.Split(b, "_")[2], ".backup"))
		return tb.Compare(ta)
	})

	for i := 0; len(broken) != 0 && len(backups) == db.cfg.DatabaseMaxBackups && i < len(broken); i++ {
		db.log.Info().Str("backup", broken[i]).Msg("remove old SQLite backup")

		if err := os.Remove(filepath.Join(backupDir, broken[i])); err != nil {
			return errors.Wrap(err, "failed to remove old backups")
		}

		db.log.Info().Str("backup", broken[i]).Msg("removed old SQLite backup")
	}

	for i := db.cfg.DatabaseMaxBackups; i < len(backups); i++ {
		db.log.Info().Str("backup", backups[i]).Msg("remove SQLite backup")

		if err := os.Remove(filepath.Join(backupDir, backups[i])); err != nil {
			return errors.Wrap(err, "failed to remove old backups")
		}

		db.log.Info().Str("backup", backups[i]).Msg("removed SQLite backup")
	}

	return nil
}
