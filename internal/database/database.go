// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"os"
	"strings"
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
	cfg     *domain.Config

	cancel func()

	Driver string
	DSN    string

	squirrel  sq.StatementBuilderType
	Statement *StatementCache
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
			db.Driver = "postgres"
			db.DSN = cfg.DatabaseDSN
			return db, nil
		} else if strings.HasPrefix(cfg.DatabaseDSN, "file:") || cfg.DatabaseDSN == ":memory:" || strings.HasSuffix(cfg.DatabaseDSN, ".db") {
			db.Driver = "sqlite"
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
	case "sqlite":
		db.Driver = "sqlite"
		if os.Getenv("IS_TEST_ENV") == "true" {
			db.DSN = ":memory:"
		} else {
			db.DSN = dataSourceName(cfg.ConfigPath, "autobrr.db")
		}
	case "sqlite:memory":
		db.Driver = "sqlite"
		db.DSN = "file::memory:"
		//db.DSN = ":memory:"
	case "postgres":
		db.Driver = "postgres"

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

	db.Statement = NewStatementCache(db.handler)

	return nil
}

func (db *DB) Close() error {
	switch db.Driver {
	case "sqlite":
		if err := db.closingSQLite(); err != nil {
			db.log.Fatal().Err(err).Msg("could not run sqlite shutdown tasks")
		}
	case "postgres":
	}

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

// ILike is a wrapper for sq.Like and sq.ILike
// SQLite does not support ILike but postgres does so this checks what database is being used
func (db *DB) ILike(col string, val string) sq.Sqlizer {
	//if databaseDriver == "sqlite" {
	if db.Driver == "sqlite" {
		return sq.Like{col: val}
	}

	return sq.ILike{col: val}
}
