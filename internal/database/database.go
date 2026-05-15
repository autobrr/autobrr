// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
)

const (
	DriverSQLite   = "sqlite"
	DriverPostgres = "postgres"

	slowQueryThreshold = 500 * time.Millisecond
)

type SQLDB interface {
	Open() error
	Migrate() error
	Close() error
	ExecContext(ctx context.Context, query string, args ...any) (result sql.Result, err error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Ping() error
	ILike(col string, val string) sq.Sqlizer
	//BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error)
}

type DB struct {
	log     zerolog.Logger
	Handler *sql.DB
	lock    sync.RWMutex
	ctx     context.Context
	cfg     *domain.Config

	cancel func()

	Driver string
	DSN    string

	squirrel sq.StatementBuilderType
}

func NewDB(cfg *domain.Config, log logger.Logger) (*DB, error) {
	db := &DB{
		// set a default placeholder for squirrel to support both sqlite and postgres
		squirrel: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log:      log.With().Str("module", "database").Str("type", cfg.DatabaseType).Logger(),
		cfg:      cfg,
	}
	db.ctx, db.cancel = context.WithCancel(context.Background())

	// Check for directly configured DSN in config
	if cfg.DatabaseDSN != "" {
		if strings.HasPrefix(cfg.DatabaseDSN, "postgres://") || strings.HasPrefix(cfg.DatabaseDSN, "postgresql://") {
			db.Driver = DriverPostgres
			db.DSN = cfg.DatabaseDSN
			return db, nil
		} else if strings.HasPrefix(cfg.DatabaseDSN, "file:") || cfg.DatabaseDSN == ":memory:" || strings.HasSuffix(cfg.DatabaseDSN, ".db") {
			db.Driver = DriverSQLite
			if strings.HasPrefix(cfg.DatabaseDSN, "file:") && strings.HasSuffix(cfg.DatabaseDSN, ".db") {
				db.DSN = strings.TrimPrefix(cfg.DatabaseDSN, "file:")
			} else {
				db.DSN = cfg.DatabaseDSN
			}
			return db, nil
		}

		return nil, errors.New("unsupported database DSN: %s", cfg.DatabaseDSN)
	}

	// If no direct DSN is provided, build it from individual settings
	switch cfg.DatabaseType {
	case DriverSQLite:
		db.Driver = DriverSQLite
		if os.Getenv("IS_TEST_ENV") == "true" {
			db.DSN = ":memory:"
		} else {
			db.DSN = dataSourceName(cfg.ConfigPath, "autobrr.db")
		}
	case DriverPostgres:
		db.Driver = DriverPostgres

		// If no database-specific settings are provided, return an error
		if cfg.PostgresDatabase == "" && cfg.DatabaseDSN == "" {
			return nil, errors.New("postgres: database name is required")
		}

		pgDsn, err := PostgresDSN(cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresDatabase, cfg.PostgresSocket, cfg.PostgresSSLMode, cfg.PostgresExtraParams)
		if err != nil {
			return nil, errors.Wrap(err, "postgres: failed to build DSN")
		}
		db.DSN = pgDsn

	default:
		return nil, errors.New("unsupported database: %v", cfg.DatabaseType)
	}

	return db, nil
}

func (db *DB) Open() error {
	if db.DSN == "" {
		return errors.New("DSN required")
	}

	switch db.Driver {
	case DriverSQLite:
		if err := db.openSQLite(); err != nil {
			return errors.Wrap(err, "could not open sqlite db connection")
		}

	case DriverPostgres:
		if err := db.openPostgres(); err != nil {
			return errors.Wrap(err, "could not open postgres db connection")
		}
	}

	return nil
}

func (db *DB) Migrate() error {
	switch db.Driver {
	case DriverSQLite:
		if err := db.migrateSQLite(); err != nil {
			return errors.Wrap(err, "could not migrate sqlite db")
		}

	case DriverPostgres:
		if err := db.migratePostgres(); err != nil {
			return errors.Wrap(err, "could not migrate postgres db")
		}
	}

	return nil
}

func (db *DB) Close() error {
	switch db.Driver {
	case DriverSQLite:
		if err := db.closingSQLite(); err != nil {
			return errors.Wrap(err, "could not run sqlite shutdown tasks")

		}
	case DriverPostgres:
	}

	// cancel background context
	db.cancel()

	// close database
	if db.Handler != nil {
		return db.Handler.Close()
	}
	return nil
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)

		logEvent := db.log.Trace()

		if err != nil {
			logEvent = db.log.Error().Err(err)
		} else if duration > slowQueryThreshold {
			logEvent = db.log.Warn().Bool("slow_query", true)
		}

		logEvent.Str("query", query).Interface("args", args).Dur("duration", duration).Msg("database query")
	}()

	rows, err = db.Handler.QueryContext(ctx, query, args...)
	return rows, err
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) (row *sql.Row) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)

		logEvent := db.log.Trace()

		if duration > slowQueryThreshold {
			logEvent = db.log.Warn().Bool("slow_query", true)
		}

		logEvent.Str("query", query).Interface("args", args).Dur("duration", duration).Msg("database query")
	}()

	row = db.Handler.QueryRowContext(ctx, query, args...)
	return row
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)

		logEvent := db.log.Trace()

		if err != nil {
			logEvent = db.log.Error()
		} else if duration > slowQueryThreshold {
			logEvent = db.log.Warn().Bool("slow_query", true)
		}

		logEvent.Str("query", query).Interface("args", args).Dur("duration", duration).Msg("database query")
	}()

	result, err = db.Handler.ExecContext(ctx, query, args...)
	return result, err
}

func (db *DB) BeginTX(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.Handler.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}

	return &Tx{
		Tx:      tx,
		handler: db,
	}, nil
}

func (db *DB) Ping() error {
	return db.Handler.Ping()
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.Handler.BeginTx(ctx, opts)
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

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)

		logEvent := tx.handler.log.Trace()

		if err != nil {
			logEvent = tx.handler.log.Error()
		} else if duration > slowQueryThreshold {
			logEvent = tx.handler.log.Warn().Bool("slow_query", true)
		}

		logEvent.Str("query", query).Interface("args", args).Dur("duration", duration).Bool("in_transaction", true).Msg("database query")
	}()

	rows, err = tx.Tx.QueryContext(ctx, query, args...)
	return rows, err
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...any) (row *sql.Row) {
	start := time.Now()

	defer func() {
		duration := time.Since(start)

		logEvent := tx.handler.log.Trace()

		if duration > slowQueryThreshold {
			logEvent = tx.handler.log.Warn().Bool("slow_query", true)
		}

		logEvent.Str("query", query).Interface("args", args).Dur("duration", duration).Bool("in_transaction", true).Msg("database query")
	}()

	row = tx.Tx.QueryRowContext(ctx, query, args...)
	return row
}

// ILike is a wrapper for sq.Like and sq.ILike
// SQLite does not support ILike but postgres does so this checks what database is being used
func (db *DB) ILike(col string, val string) sq.Sqlizer {
	//if databaseDriver == DriverSQLite {
	if db.Driver == DriverSQLite {
		return sq.Like{col: val}
	}

	return sq.ILike{col: val}
}
