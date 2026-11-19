// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"
)

// QueryPrepared runs query through a cached prepared statement, falling back to
// a plain query when it can not be prepared or the database has no statement
// cache (Postgres). Only use it for a static SQL shape: every distinct query
// string stays prepared for the life of the process.
//
// Never call it while holding a transaction on the same goroutine, SQLite has a
// single connection.
func (db *DB) QueryPrepared(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	start := time.Now()

	prepared := false

	defer func() {
		db.logPrepared(query, args, time.Since(start), prepared, err)
	}()

	stmt, err := db.statement(ctx, query)
	if err != nil {
		return db.Handler.QueryContext(ctx, query, args...)
	}

	prepared = true

	return stmt.QueryContext(ctx, args...)
}

// QueryRowPrepared is QueryPrepared for a query returning a single row, scanned
// into dest. It returns sql.ErrNoRows when there is no row.
func (db *DB) QueryRowPrepared(ctx context.Context, query string, args []any, dest ...any) error {
	rows, err := db.QueryPrepared(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}

		return sql.ErrNoRows
	}

	if err := rows.Scan(dest...); err != nil {
		return err
	}

	return rows.Close()
}

// ExecPrepared is QueryPrepared for statements that return no rows.
func (db *DB) ExecPrepared(ctx context.Context, query string, args ...any) (result sql.Result, err error) {
	start := time.Now()

	prepared := false

	defer func() {
		db.logPrepared(query, args, time.Since(start), prepared, err)
	}()

	stmt, err := db.statement(ctx, query)
	if err != nil {
		return db.Handler.ExecContext(ctx, query, args...)
	}

	prepared = true

	return stmt.ExecContext(ctx, args...)
}

func (db *DB) statement(ctx context.Context, query string) (*sql.Stmt, error) {
	if db.stmts == nil {
		return nil, errStatementCacheDisabled
	}

	stmt, err := db.stmts.get(ctx, query)
	if err != nil && !errors.Is(err, errStatementCacheDisabled) && !errors.Is(err, errStatementPrepareFailed) {
		db.log.Warn().Err(err).Str("query", query).Msg("could not prepare statement, running it unprepared")
	}

	return stmt, err
}

// logPrepared keeps args at trace level only, release inserts bind download
// urls that can carry a passkey.
func (db *DB) logPrepared(query string, args []any, duration time.Duration, prepared bool, err error) {
	logEvent := db.log.Trace().Interface("args", args)

	if err != nil {
		logEvent = db.log.Error().Err(err)
	} else if duration > slowQueryThreshold {
		logEvent = db.log.Warn().Bool("slow_query", true)
	}

	logEvent.Str("query", query).Dur("duration", duration).Bool("prepared", prepared).Msg("database query")
}
