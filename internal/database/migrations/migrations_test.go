// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package migrations_test

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/pkg/migrator"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Pending migrations are picked by position, so a migration registered out of
// numeric order runs twice on some databases and never on others.
func TestMigrationsRegisteredInOrder(t *testing.T) {
	tests := []struct {
		name    string
		migrate *migrator.Migrator
		dir     string
	}{
		{name: "sqlite", migrate: migrations.SQLiteMigrations(nil, zerolog.Nop()), dir: "sqlite"},
		{name: "postgres", migrate: migrations.PostgresMigrations(nil, zerolog.Nop()), dir: "postgres"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registered := map[string]struct{}{}
			previous := -1

			for _, m := range tt.migrate.Migrations() {
				prefix, _, _ := strings.Cut(m.Name, "_")
				n, err := strconv.Atoi(prefix)
				require.NoError(t, err, "migration %s has no numeric prefix", m.Name)
				assert.Greater(t, n, previous, "migration %s is out of order", m.Name)

				previous = n
				registered[m.Name] = struct{}{}
			}

			entries, err := os.ReadDir(tt.dir)
			require.NoError(t, err)

			for _, e := range entries {
				if strings.HasSuffix(e.Name(), "_test.go") || strings.HasPrefix(e.Name(), "current_schema") {
					continue
				}

				name := strings.TrimSuffix(strings.TrimSuffix(e.Name(), ".sql"), ".go")

				assert.Contains(t, registered, name, "migration file %s is not registered", e.Name())
			}
		})
	}
}
