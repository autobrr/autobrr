package database

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

type DB struct {
	log     zerolog.Logger
	handler *sql.DB
	lock    sync.RWMutex
	ctx     context.Context
	cancel  func()

	Driver string
	DSN    string

	squirrel sq.StatementBuilderType
}

func NewDB(cfg *domain.Config, log logger.Logger) (*DB, error) {
	db := &DB{
		// set default placeholder for squirrel to support both sqlite and postgres
		squirrel: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log:      log.With().Str("module", "database").Logger(),
	}
	db.ctx, db.cancel = context.WithCancel(context.Background())

	switch cfg.DatabaseType {
	case "sqlite":
		db.Driver = "sqlite"
		db.DSN = dataSourceName(cfg.ConfigPath, "autobrr.db")
	case "postgres":
		if cfg.PostgresHost == "" || cfg.PostgresPort == 0 || cfg.PostgresDatabase == "" {
			return nil, errors.New("postgres: bad variables")
		}
		db.DSN = fmt.Sprintf("postgres://%v:%v@%v:%d/%v?sslmode=disable", cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDatabase)
		db.Driver = "postgres"
	default:
		return nil, errors.New("unsupported database: %v", cfg.DatabaseType)
	}

	return db, nil
}

func (db *DB) Open() error {
	if db.DSN == "" {
		return errors.New("DSN required")
	}

	var err error

	switch db.Driver {
	case "sqlite":
		if err = db.openSQLite(); err != nil {
			db.log.Fatal().Err(err).Msg("could not open sqlite db connection")
			return err
		}
	case "postgres":
		if err = db.openPostgres(); err != nil {
			db.log.Fatal().Err(err).Msg("could not open postgres db connection")
			return err
		}
	}

	return nil
}

func (db *DB) Close() error {
	// cancel background context
	db.cancel()

	// close database
	if db.handler != nil {
		return db.handler.Close()
	}
	return nil
}

func (db *DB) Ping() error {
	return db.handler.Ping()
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
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
	handler *DB
}
