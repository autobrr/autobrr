package database

import (
	"database/sql"

	"github.com/autobrr/autobrr/pkg/errors"
	sq "github.com/Masterminds/squirrel"

	_ "github.com/lib/pq"
)

func (db *DB) openPostgres() error {
	var err error

	// open database connection
	handler, err := sql.Open("postgres", db.DSN);
	if err != nil {
		db.log.Fatal().Err(err).Msg("could not open postgres connection")
		return errors.Wrap(err, "could not open postgres connection")
	}

	err = handler.Ping()
	if err != nil {
		db.log.Fatal().Err(err).Msg("could not ping postgres database")
		return errors.Wrap(err, "could not ping postgres database")
	}

	db.db = handler
	db.handler = sq.NewStmtCacheProxy(db.db)

	// migrate db
	if err = db.migratePostgres(); err != nil {
		db.log.Fatal().Err(err).Msg("could not migrate postgres database")
		return errors.Wrap(err, "could not migrate postgres database")
	}

	return nil
}

func (db *DB) migratePostgres() error {
	tx, err := db.handler.Begin()
	if err != nil {
		return errors.Wrap(err, "error starting transaction")
	}
	defer tx.Rollback()

	initialSchema := `CREATE TABLE IF NOT EXISTS schema_migrations (
	id INTEGER PRIMARY KEY,
	version INTEGER NOT NULL
);`

	if _, err := tx.Exec(initialSchema); err != nil {
		return errors.New("failed to create schema_migrations table")
	}

	var version int
	err = tx.QueryRow(`SELECT version FROM schema_migrations`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return errors.Wrap(err, "no rows")
	}

	if version == len(postgresMigrations) {
		return nil
	}
	if version > len(postgresMigrations) {
		return errors.New("old")
	}

	if version == 0 {
		if _, err := tx.Exec(postgresSchema); err != nil {
			return errors.Wrap(err, "failed to initialize schema")
		}
	} else {
		for i := version; i < len(postgresMigrations); i++ {
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
