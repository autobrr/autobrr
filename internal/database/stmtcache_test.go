// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestPreparedDB(t *testing.T) *DB {
	t.Helper()

	handler, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)

	handler.SetMaxOpenConns(1)

	_, err = handler.ExecContext(t.Context(), `CREATE TABLE item (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL)`)
	require.NoError(t, err)

	db := &DB{log: zerolog.Nop(), Handler: handler, Driver: DriverSQLite}
	db.stmts = newStatementCache(handler)

	t.Cleanup(func() {
		_ = db.stmts.close()
		_ = handler.Close()
	})

	return db
}

func TestStatementCache_ReusesStatement(t *testing.T) {
	db := newTestPreparedDB(t)

	first, err := db.stmts.get(t.Context(), `SELECT id FROM item WHERE name = ?`)
	require.NoError(t, err)

	second, err := db.stmts.get(t.Context(), `SELECT id FROM item WHERE name = ?`)
	require.NoError(t, err)

	assert.Same(t, first, second)
}

func TestStatementCache_ConcurrentMissesShareOneStatement(t *testing.T) {
	db := newTestPreparedDB(t)

	const query = `SELECT name FROM item WHERE id = ?`

	var wg sync.WaitGroup
	stmts := make([]*sql.Stmt, 16)

	for i := range stmts {
		wg.Go(func() {
			stmt, err := db.stmts.get(t.Context(), query)
			assert.NoError(t, err)
			stmts[i] = stmt
		})
	}

	wg.Wait()

	for _, stmt := range stmts {
		assert.Same(t, stmts[0], stmt)
	}

	assert.Len(t, db.stmts.stmts, 1)
}

func TestStatementCache_FullFallsBackUnprepared(t *testing.T) {
	db := newTestPreparedDB(t)

	for i := range maxCachedStatements {
		_, err := db.stmts.get(t.Context(), fmt.Sprintf(`SELECT %d`, i))
		require.NoError(t, err)
	}

	_, err := db.stmts.get(t.Context(), `SELECT name FROM item`)
	require.ErrorIs(t, err, errStatementCacheFull)

	_, err = db.ExecPrepared(t.Context(), `INSERT INTO item (name) VALUES (?)`, "overflow")
	require.NoError(t, err)

	var name string
	require.NoError(t, db.QueryRowPrepared(t.Context(), `SELECT name FROM item WHERE name = ?`, []any{"overflow"}, &name))
	assert.Equal(t, "overflow", name)
}

func TestStatementCache_PrepareFailureIsRemembered(t *testing.T) {
	db := newTestPreparedDB(t)

	const query = `SELECT * FROM missing_table`

	_, err := db.stmts.get(t.Context(), query)
	require.Error(t, err)

	_, err = db.stmts.get(t.Context(), query)
	require.ErrorIs(t, err, errStatementPrepareFailed)

	_, err = db.QueryPrepared(t.Context(), query)
	require.Error(t, err, "the unprepared fallback should surface the real error")
}

func TestStatementCache_ClosedCacheRunsUnprepared(t *testing.T) {
	db := newTestPreparedDB(t)

	require.NoError(t, db.stmts.close())

	_, err := db.ExecPrepared(t.Context(), `INSERT INTO item (name) VALUES (?)`, "after-close")
	require.NoError(t, err)

	assert.Empty(t, db.stmts.stmts)
}

func TestDB_QueryRowPrepared(t *testing.T) {
	db := newTestPreparedDB(t)

	var id int64
	require.NoError(t, db.QueryRowPrepared(t.Context(), `INSERT INTO item (name) VALUES (?) RETURNING id`, []any{"first"}, &id))
	assert.Equal(t, int64(1), id)

	require.NoError(t, db.QueryRowPrepared(t.Context(), `INSERT INTO item (name) VALUES (?) RETURNING id`, []any{"second"}, &id))
	assert.Equal(t, int64(2), id)

	var name string
	err := db.QueryRowPrepared(t.Context(), `SELECT name FROM item WHERE id = ?`, []any{99}, &name)
	require.ErrorIs(t, err, sql.ErrNoRows)

	rows, err := db.QueryPrepared(t.Context(), `SELECT name FROM item ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, []string{"first", "second"}, names)
	assert.Len(t, db.stmts.stmts, 3)
}
