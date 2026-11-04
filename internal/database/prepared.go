// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/lib/pq"
)

// QueryPrepared runs query through a cached prepared statement, falling back to
// a plain query when it can not be prepared. Only use it for a static SQL shape:
// every distinct query string stays prepared for the life of the process.
//
// Never call it while holding a transaction on the same goroutine, SQLite has a
// single connection.
func (db *DB) QueryPrepared(ctx context.Context, query string, args ...any) (rows *sql.Rows, err error) {
	start := time.Now()

	defer func() {
		db.logPrepared(query, args, time.Since(start), err)
	}()

	stmt, err := db.statement(ctx, query)
	if err != nil {
		return db.Handler.QueryContext(ctx, query, args...)
	}

	rows, err = stmt.QueryContext(ctx, args...)
	if err != nil && db.isStalePreparedStatement(err) {
		return db.Handler.QueryContext(ctx, query, args...)
	}

	return rows, err
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

	defer func() {
		db.logPrepared(query, args, time.Since(start), err)
	}()

	stmt, err := db.statement(ctx, query)
	if err != nil {
		return db.Handler.ExecContext(ctx, query, args...)
	}

	result, err = stmt.ExecContext(ctx, args...)
	if err != nil && db.isStalePreparedStatement(err) {
		return db.Handler.ExecContext(ctx, query, args...)
	}

	return result, err
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

// isStalePreparedStatement reports whether a named prepared statement vanished
// from the server under us. A pooler in transaction mode such as PgBouncer
// before 1.21 routes the execution to a different server connection than the
// prepare, which only fails here, at execution time. The cache is turned off
// for the rest of the process, and the failed call is safe to retry unprepared
// since the statement never ran.
func (db *DB) isStalePreparedStatement(err error) bool {
	pqErr, ok := errors.AsType[*pq.Error](err)
	if !ok {
		return false
	}

	if pqErr.Code != "26000" && pqErr.Code != "42P05" {
		return false
	}

	if db.stmts.disable() {
		db.log.Warn().Err(err).Msg("prepared statements do not survive this connection pooler, disabling the statement cache")
	}

	return true
}

func (db *DB) logPrepared(query string, args []any, duration time.Duration, err error) {
	logEvent := db.log.Trace()

	if err != nil {
		logEvent = db.log.Error().Err(err)
	} else if duration > slowQueryThreshold {
		logEvent = db.log.Warn().Bool("slow_query", true)
	}

	logEvent.Str("query", query).Interface("args", args).Dur("duration", duration).Bool("prepared", true).Msg("database query")
}
