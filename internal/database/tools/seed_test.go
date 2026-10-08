// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package tools

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSeederDB(t *testing.T) string {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "autobrr.db")

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	for _, table := range GetTables() {
		_, err := db.Exec(`CREATE TABLE "` + table + `" (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT)`)
		require.NoError(t, err)
	}

	_, err = db.Exec(`INSERT INTO filter (name) VALUES ('existing')`)
	require.NoError(t, err)

	return dbPath
}

func writeSeedFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "seed.sql")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	return path
}

func filterNames(t *testing.T, dbPath string) []string {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	require.NoError(t, err)
	defer db.Close()

	rows, err := db.Query(`SELECT name FROM filter ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		names = append(names, name)
	}
	require.NoError(t, rows.Err())

	return names
}

func TestSQLiteSeeder_ResetAndSeed(t *testing.T) {
	tests := []struct {
		name    string
		seed    string
		wantErr bool
		want    []string
	}{
		{
			name: "replaces rows with seed",
			seed: "INSERT INTO filter (name) VALUES ('seeded');\n",
			want: []string{"seeded"},
		},
		{
			name:    "failing seed keeps existing rows",
			seed:    "INSERT INTO filter (name) VALUES ('seeded'); INSERT INTO missing_table (name) VALUES ('x');",
			wantErr: true,
			want:    []string{"existing"},
		},
		{
			name:    "empty seed keeps existing rows",
			seed:    " ;\n ; ",
			wantErr: true,
			want:    []string{"existing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := setupSeederDB(t)

			err := NewSQLiteSeeder(dbPath, writeSeedFile(t, tt.seed)).ResetAndSeed()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, filterNames(t, dbPath))
		})
	}
}

func TestSQLiteSeeder_ResetAndSeed_MissingSeedFile(t *testing.T) {
	dbPath := setupSeederDB(t)

	err := NewSQLiteSeeder(dbPath, filepath.Join(t.TempDir(), "missing.sql")).ResetAndSeed()
	assert.Error(t, err)

	assert.Equal(t, []string{"existing"}, filterNames(t, dbPath))
}
