package database

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"
)

func (db *DB) openSQLite() error {
	if db.DSN == "" {
		return fmt.Errorf("DSN required")
	}

	var err error

	// open database connection
	if db.handler, err = sql.Open("sqlite", db.DSN+"?_pragma=busy_timeout%3d1000"); err != nil {
		log.Fatal().Err(err).Msg("could not open db connection")
		return err
	}

	// Set busy timeout
	//if _, err = db.handler.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
	//	return fmt.Errorf("busy timeout pragma: %w", err)
	//}

	// Enable WAL. SQLite performs better with the WAL  because it allows
	// multiple readers to operate while data is being written.
	if _, err = db.handler.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return fmt.Errorf("enable wal: %w", err)
	}

	// Enable foreign key checks. For historical reasons, SQLite does not check
	// foreign key constraints by default. There's some overhead on inserts to
	// verify foreign key integrity, but it's definitely worth it.
	if _, err = db.handler.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return fmt.Errorf("foreign keys pragma: %w", err)
	}

	// migrate db
	if err = db.migrateSQLite(); err != nil {
		log.Fatal().Err(err).Msg("could not migrate db")
		return err
	}

	return nil
}

func (db *DB) migrateSQLite() error {
	db.lock.Lock()
	defer db.lock.Unlock()

	var version int
	if err := db.handler.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("failed to query schema version: %v", err)
	}

	if version == len(sqliteMigrations) {
		return nil
	} else if version > len(sqliteMigrations) {
		return fmt.Errorf("autobrr (version %d) older than schema (version: %d)", len(sqliteMigrations), version)
	}

	tx, err := db.handler.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if version == 0 {
		if _, err := tx.Exec(sqliteSchema); err != nil {
			return fmt.Errorf("failed to initialize schema: %v", err)
		}
	} else {
		for i := version; i < len(sqliteMigrations); i++ {
			if _, err := tx.Exec(sqliteMigrations[i]); err != nil {
				return fmt.Errorf("failed to execute migration #%v: %v", i, err)
			}
		}
	}

	// temp custom data migration
	// get data from filter.sources, check if specific types, move to new table and clear
	// if migration 6
	// TODO 2022-01-30 remove this in future version
	if version == 5 && len(sqliteMigrations) == 6 {
		if err := customMigrateCopySourcesToMedia(tx); err != nil {
			return fmt.Errorf("could not run custom data migration: %v", err)
		}
	}

	_, err = tx.Exec(fmt.Sprintf("PRAGMA user_version = %d", len(sqliteMigrations)))
	if err != nil {
		return fmt.Errorf("failed to bump schema version: %v", err)
	}

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
		return fmt.Errorf("could not run custom data migration: %v", err)
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
