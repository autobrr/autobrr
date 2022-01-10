package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog/log"
)

type SqliteDB struct {
	lock    sync.RWMutex
	handler *sql.DB
	ctx     context.Context
	cancel  func()
}

func OpenSqliteDB(source string) (*SqliteDB, error) {

	// if configPath is set then put database inside that path, otherwise create wherever it's run
	var dataSource = DataSourceName(source, "autobrr.db")

	// open database connection
	conn, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		log.Fatal().Err(err).Msg("could not open db connection")
		return nil, err
	}
	//conn.SetMaxOpenConns(2)

	db := &SqliteDB{
		handler: conn,
	}

	db.ctx, db.cancel = context.WithCancel(context.Background())

	if _, err := db.handler.Exec(`PRAGMA busy_timeout = 5000;`); err != nil {
		return nil, fmt.Errorf("busy timeout pragma")
	}
	if _, err := db.handler.Exec(`PRAGMA journal_mode = wal;`); err != nil {
		return nil, fmt.Errorf("enable wal: %w", err)
	}
	if _, err := db.handler.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
		return nil, fmt.Errorf("foreign keys pragma: %w", err)
	}

	// migrate db
	if err = db.migrate(); err != nil {
		log.Fatal().Err(err).Msg("could not migrate db")
		return nil, err
	}

	return db, err
}

//func (db *SqliteDB) Open() error {
//	// if configPath is set then put database inside that path, otherwise create wherever it's run
//	var dataSource = DataSourceName(source, "autobrr.db")
//
//	// open database connection
//	conn, err := sql.Open("sqlite3", dataSource)
//	if err != nil {
//		log.Fatal().Err(err).Msg("could not open db connection")
//		return  err
//	}
//	//conn.SetMaxOpenConns(2)
//
//
//	// migrate db
//	if err = db.migrate(); err != nil {
//		log.Fatal().Err(err).Msg("could not migrate db")
//		return  err
//	}
//
//	if _, err := db.handler.Exec(`PRAGMA journal_mode = wal;`); err != nil {
//		return fmt.Errorf("enable wal: %w", err)
//	}
//	if _, err := db.handler.Exec(`PRAGMA foreign_keys = ON;`); err != nil {
//		return fmt.Errorf("foreign keys pragma: %w", err)
//	}
//	return nil
//}

func (db *SqliteDB) Close() error {
	db.lock.Lock()
	defer db.lock.Unlock()
	return db.handler.Close()
}
