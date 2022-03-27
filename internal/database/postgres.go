package database

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

func (db *DB) openPostgres() error {
	var err error

	// open database connection
	if db.handler, err = sql.Open("postgres", db.DSN); err != nil {
		log.Fatal().Err(err).Msg("could not open postgres connection")
		return err
	}

	err = db.handler.Ping()
	if err != nil {
		log.Fatal().Err(err).Msg("could not ping postgres database")
		return err
	}

	// migrate db
	if err = db.migratePostgres(); err != nil {
		log.Fatal().Err(err).Msg("could not migrate postgres database")
		return err
	}

	return nil
}

func (db *DB) migratePostgres() error {
	tx, err := db.handler.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	initialSchema := `CREATE TABLE IF NOT EXISTS schema_migrations (
	id INTEGER PRIMARY KEY,
	version INTEGER NOT NULL
);`

	if _, err := tx.Exec(initialSchema); err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %s", err)
	}

	var version int
	err = tx.QueryRow(`SELECT version FROM schema_migrations`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if version == len(migrations) {
		return nil
	}
	if version > len(migrations) {
		return fmt.Errorf("old")
	}

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

	_, err = tx.Exec(`INSERT INTO schema_migrations (id, version) VALUES (1, $1) ON CONFLICT (id) DO UPDATE SET version = $1`, len(migrations))
	if err != nil {
		return fmt.Errorf("failed to bump schema version: %v", err)
	}

	return tx.Commit()
}
