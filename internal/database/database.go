// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
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

	squirrel sq.StatementBuilderType
}

func NewDB(cfg *domain.Config, log logger.Logger) (*DB, error) {
	db := &DB{
		// set default placeholder for squirrel to support both sqlite and postgres
		squirrel: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
		log:      log.With().Str("module", "database").Str("type", cfg.DatabaseType).Logger(),
		cfg:      cfg,
	}
	db.ctx, db.cancel = context.WithCancel(context.Background())

	switch cfg.DatabaseType {
	case "sqlite":
		db.Driver = "sqlite"
		if os.Getenv("IS_TEST_ENV") == "true" {
			db.DSN = ":memory:"
		} else {
			db.DSN = dataSourceName(cfg.ConfigPath, "autobrr.db")
		}
	case "postgres":
		if cfg.PostgresHost == "" || cfg.PostgresPort == 0 || cfg.PostgresDatabase == "" {
			return nil, errors.New("postgres: bad variables")
		}
		db.DSN = fmt.Sprintf("postgres://%v:%v@%v:%d/%v?sslmode=%v", cfg.PostgresUser, cfg.PostgresPass, cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresDatabase, cfg.PostgresSSLMode)
		if cfg.PostgresExtraParams != "" {
			db.DSN = fmt.Sprintf("%s&%s", db.DSN, cfg.PostgresExtraParams)
		}
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
