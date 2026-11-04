// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"sync"
	"sync/atomic"

	"github.com/autobrr/autobrr/pkg/errors"
)

// maxCachedStatements bounds the cache as a safety valve. The opted-in queries
// are a handful of static strings, so it is never reached in practice.
const maxCachedStatements = 64

var (
	errStatementCacheDisabled = errors.Sentinel("statement cache disabled")
	errStatementCacheFull     = errors.Sentinel("statement cache full")
	errStatementPrepareFailed = errors.Sentinel("statement prepare failed before")
)

// statementCache keeps one prepared statement per query string for the life of
// the process. modernc.org/sqlite keeps the compiled statement per *sql.Stmt,
// so reusing one skips parsing and planning the query on every execution.
//
// There is no eviction on purpose: callers opt in with static SQL shapes only,
// and evicting a statement another goroutine is about to use would turn into
// "sql: statement is closed" on the announce path.
type statementCache struct {
	handler *sql.DB

	m      sync.RWMutex
	stmts  map[string]*sql.Stmt
	failed map[string]struct{}

	disabled atomic.Bool
}

func newStatementCache(handler *sql.DB) *statementCache {
	return &statementCache{
		handler: handler,
		stmts:   make(map[string]*sql.Stmt),
		failed:  make(map[string]struct{}),
	}
}

// get returns the cached statement for query, preparing it on first use. An
// error means the caller should run the query unprepared.
func (c *statementCache) get(ctx context.Context, query string) (*sql.Stmt, error) {
	if c.disabled.Load() {
		return nil, errStatementCacheDisabled
	}

	c.m.RLock()
	stmt, ok := c.stmts[query]
	_, failed := c.failed[query]
	c.m.RUnlock()

	if ok {
		return stmt, nil
	}

	if failed {
		return nil, errStatementPrepareFailed
	}

	// prepared outside the lock, with a single SQLite connection a prepare waits
	// for whoever holds it and must not block cache hits meanwhile
	stmt, err := c.handler.PrepareContext(ctx, query)
	if err != nil {
		if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
			c.m.Lock()
			c.failed[query] = struct{}{}
			c.m.Unlock()
		}

		return nil, err
	}

	c.m.Lock()
	defer c.m.Unlock()

	if existing, ok := c.stmts[query]; ok {
		_ = stmt.Close()
		return existing, nil
	}

	if len(c.stmts) >= maxCachedStatements {
		_ = stmt.Close()
		return nil, errStatementCacheFull
	}

	c.stmts[query] = stmt

	return stmt, nil
}

// disable stops handing out statements for the rest of the process. Statements
// already handed out stay open, a goroutine may still be running one.
func (c *statementCache) disable() bool {
	return c.disabled.CompareAndSwap(false, true)
}

func (c *statementCache) close() error {
	c.disabled.Store(true)

	c.m.Lock()
	defer c.m.Unlock()

	var closeErr error
	for query, stmt := range c.stmts {
		if err := stmt.Close(); err != nil && closeErr == nil {
			closeErr = errors.Wrap(err, "could not close prepared statement")
		}

		delete(c.stmts, query)
	}

	return closeErr
}
