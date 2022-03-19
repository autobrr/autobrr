package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"
)

type SqliteDB struct {
	lock    sync.RWMutex
	handler *sql.DB
	ctx     context.Context
	cancel  func()

	DSN string
}

func NewSqliteDB(source string) *SqliteDB {
	db := &SqliteDB{
		DSN: dataSourceName(source, "autobrr.db"),
	}

	db.ctx, db.cancel = context.WithCancel(context.Background())

	return db
}

func (db *SqliteDB) Open() error {
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
	if err = db.migrate(); err != nil {
		log.Fatal().Err(err).Msg("could not migrate db")
		return err
	}

	return nil
}

func (db *SqliteDB) Close() error {
	// cancel background context
	db.cancel()

	// close database
	if db.handler != nil {
		return db.handler.Close()
	}
	return nil
}

func (db *SqliteDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.handler.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Tx{
		Tx:      tx,
		handler: db,
	}, nil
}

type Tx struct {
	*sql.Tx
	handler *SqliteDB
}
