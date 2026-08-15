// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package database

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	sq "github.com/Masterminds/squirrel"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newDeprecationTestDB(t *testing.T) *DB {
	t.Helper()

	handler, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	handler.SetMaxOpenConns(1)
	t.Cleanup(func() { require.NoError(t, handler.Close()) })

	_, err = handler.Exec(`
		CREATE TABLE indexer (
			id INTEGER PRIMARY KEY,
			identifier TEXT NOT NULL UNIQUE,
			archived BOOLEAN NOT NULL DEFAULT FALSE,
			archived_at TIMESTAMP,
			updated_at TIMESTAMP
		);
		CREATE TABLE indexer_deprecation (
			identifier TEXT NOT NULL UNIQUE,
			name TEXT CHECK (name <> 'invalid'),
			reason TEXT,
			issue_url TEXT,
			alias_of TEXT,
			deprecated_at TIMESTAMP
		);
		CREATE TABLE filter_indexer (
			filter_id INTEGER NOT NULL,
			indexer_id INTEGER NOT NULL
		);
	`)
	require.NoError(t, err)

	return &DB{
		Handler:  handler,
		Driver:   DriverSQLite,
		squirrel: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func TestIndexerRepoReconcileDeprecations(t *testing.T) {
	db := newDeprecationTestDB(t)
	repo := NewIndexerRepo(zerolog.Nop(), db)
	ctx := context.Background()

	_, err := db.Handler.ExecContext(ctx, `
		INSERT INTO indexer (identifier, archived, archived_at)
		VALUES ('retired', FALSE, NULL), ('restored', TRUE, CURRENT_TIMESTAMP)
	`)
	require.NoError(t, err)

	deprecation := domain.IndexerDeprecation{
		Identifier:   "retired",
		Name:         "Retired Tracker",
		Reason:       "Tracker shut down",
		IssueURL:     "https://example.com/retired",
		DeprecatedAt: time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
	}
	require.NoError(t, repo.ReconcileDeprecations(ctx, []domain.IndexerDeprecation{deprecation}, map[string]struct{}{"restored": {}}))

	var retiredArchived, restoredArchived bool
	require.NoError(t, db.Handler.QueryRowContext(ctx, "SELECT archived FROM indexer WHERE identifier = 'retired'").Scan(&retiredArchived))
	require.NoError(t, db.Handler.QueryRowContext(ctx, "SELECT archived FROM indexer WHERE identifier = 'restored'").Scan(&restoredArchived))
	assert.True(t, retiredArchived)
	assert.False(t, restoredArchived)

	var restoredAt sql.NullTime
	require.NoError(t, db.Handler.QueryRowContext(ctx, "SELECT archived_at FROM indexer WHERE identifier = 'restored'").Scan(&restoredAt))
	assert.False(t, restoredAt.Valid)

	var storedName string
	require.NoError(t, db.Handler.QueryRowContext(ctx, "SELECT name FROM indexer_deprecation WHERE identifier = 'retired'").Scan(&storedName))
	assert.Equal(t, deprecation.Name, storedName)
}

func TestIndexerRepoReconcileDeprecationsRollsBack(t *testing.T) {
	db := newDeprecationTestDB(t)
	repo := NewIndexerRepo(zerolog.Nop(), db)
	ctx := context.Background()

	_, err := db.Handler.ExecContext(ctx, "INSERT INTO indexer (identifier) VALUES ('first')")
	require.NoError(t, err)

	err = repo.ReconcileDeprecations(ctx, []domain.IndexerDeprecation{
		{Identifier: "first", Name: "First"},
		{Identifier: "second", Name: "invalid"},
	}, nil)
	require.Error(t, err)

	var archived bool
	require.NoError(t, db.Handler.QueryRowContext(ctx, "SELECT archived FROM indexer WHERE identifier = 'first'").Scan(&archived))
	assert.False(t, archived)

	var count int
	require.NoError(t, db.Handler.QueryRowContext(ctx, "SELECT COUNT(*) FROM indexer_deprecation").Scan(&count))
	assert.Zero(t, count)
}

func TestDeleteArchivedIndexerConnectionsScope(t *testing.T) {
	db := newDeprecationTestDB(t)
	repo := NewFilterRepo(zerolog.Nop(), db)
	ctx := context.Background()

	_, err := db.Handler.ExecContext(ctx, `
		INSERT INTO indexer (id, identifier, archived)
		VALUES (1, 'first', TRUE), (2, 'second', TRUE), (3, 'active', FALSE);
		INSERT INTO filter_indexer (filter_id, indexer_id)
		VALUES (1, 1), (1, 2), (1, 3);
	`)
	require.NoError(t, err)

	removed, err := repo.DeleteArchivedIndexerConnections(ctx, []string{"first"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), removed)

	var remaining int
	require.NoError(t, db.Handler.QueryRowContext(ctx, "SELECT COUNT(*) FROM filter_indexer WHERE indexer_id = 2").Scan(&remaining))
	assert.Equal(t, 1, remaining)

	removed, err = repo.DeleteArchivedIndexerConnections(ctx, nil)
	require.NoError(t, err)
	assert.Equal(t, int64(1), removed)
	require.NoError(t, db.Handler.QueryRowContext(ctx, "SELECT COUNT(*) FROM filter_indexer").Scan(&remaining))
	assert.Equal(t, 1, remaining, "the active indexer connection must be preserved")
}
