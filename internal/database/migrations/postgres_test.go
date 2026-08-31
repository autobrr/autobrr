// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package migrations_test

import (
	"bytes"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/migrator"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// embeddedPGPaths returns the directory the Postgres binaries are extracted to along with a
// runtime directory for a single instance. embedded-postgres wipes its runtime path on every
// Start, so instances must not share one; the binaries are extracted once and reused. The
// binaries path is package specific so a cold start can never have two test binaries
// extracting into the same directory.
func embeddedPGPaths(t *testing.T) (binariesPath, runtimePath string) {
	t.Helper()

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	return filepath.Join(cacheDir, "autobrr", "embedded-postgres", "v17-migrations"), t.TempDir()
}

func setupPGTestDB(t *testing.T) (*database.DB, func(), error) {
	t.Helper()

	var (
		dbUsername = "postgres"
		dbPassword = "postgres"
		dbName     = "autobrr"
		dbPort     = 9876
	)

	binariesPath, runtimePath := embeddedPGPaths(t)

	pgLogger := &bytes.Buffer{}
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username(dbUsername).
		Password(dbPassword).
		Database(dbName).
		Port(uint32(dbPort)).
		Version(embeddedpostgres.V17).
		BinariesPath(binariesPath).
		RuntimePath(runtimePath).
		StartTimeout(45 * time.Second).
		StartParameters(map[string]string{"max_connections": "200"}).
		Logger(pgLogger))

	err := postgres.Start()
	if err != nil {
		//t.Error(err)
		return nil, nil, err
	}

	//dsn := "postgres://postgres:postgres@localhost:9876/autobrr?sslmode=disable"
	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable", dbUsername, dbPassword, dbPort, dbName)

	cfg := &domain.Config{
		DatabaseType:        "postgres",
		DatabaseDSN:         dsn,
		DatabaseAutoMigrate: false,
	}

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""}, nil)
	db, err := database.NewDB(cfg, log)
	if err != nil {
		return nil, nil, err
	}

	err = db.Open()
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = db.Close()
		_ = postgres.Stop()
	}

	return db, cleanup, nil
}

// Test full migration sequence
func TestFullMigrationSequencePostgres(t *testing.T) {
	db, cleanup, err := setupPGTestDB(t)
	defer cleanup()
	require.NoError(t, err)

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""}, nil)

	// This will run all migrations
	migrate := migrations.PostgresMigrations(db.Handler, log.With().Logger())

	err = migrate.Migrate()
	require.NoError(t, err)

	//// Verify current schema version
	//var version int
	//err = db.Handler.QueryRow("PRAGMA user_version").Scan(&version)
	//require.NoError(t, err)
	//
	//expectedVersion := len(database.sqliteMigrations)
	//assert.Equal(t, expectedVersion, version, "Should be at latest migration version")
}

//func TestMain(m *testing.M) {
//	pgDb, cleanup, err := setupPGTestDB()
//	//defer cleanup()
//	if err != nil {
//		fmt.Println(err)
//		os.Exit(1)
//	}
//
//	code := m.Run()
//
//	cleanup()
//
//	os.Exit(code)
//}

// startEmbeddedPGOnPort boots an embedded Postgres on the given port and returns
// an open *database.DB plus a cleanup that stops the server. Used so the per-migration
// test in this file can boot Postgres once and reuse it across sub-tests.
func startEmbeddedPGOnPort(t *testing.T, port int) (*database.DB, func()) {
	t.Helper()

	var (
		dbUsername = "postgres"
		dbPassword = "postgres"
		dbName     = "autobrr"
	)

	binariesPath, runtimePath := embeddedPGPaths(t)

	pgLogger := &bytes.Buffer{}
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username(dbUsername).
		Password(dbPassword).
		Database(dbName).
		Port(uint32(port)).
		Version(embeddedpostgres.V17).
		BinariesPath(binariesPath).
		RuntimePath(runtimePath).
		StartTimeout(45 * time.Second).
		StartParameters(map[string]string{"max_connections": "200"}).
		Logger(pgLogger))

	require.NoError(t, pg.Start(), "failed to start embedded postgres")

	dsn := fmt.Sprintf("postgres://%s:%s@localhost:%d/%s?sslmode=disable", dbUsername, dbPassword, port, dbName)
	cfg := &domain.Config{
		DatabaseType:        "postgres",
		DatabaseDSN:         dsn,
		DatabaseAutoMigrate: false,
	}

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""}, nil)
	db, err := database.NewDB(cfg, log)
	require.NoError(t, err)
	require.NoError(t, db.Open())

	cleanup := func() {
		_ = db.Close()
		if err := pg.Stop(); err != nil {
			t.Logf("failed to stop embedded postgres: %v\npg log:\n%s", err, pgLogger.String())
		}
	}

	return db, cleanup
}

// resetPublicSchema drops and recreates the public schema so each migration sub-test
// starts from a clean slate without paying the embedded-pg startup cost again.
func resetPublicSchema(t *testing.T, db *sql.DB) {
	t.Helper()
	_, err := db.Exec(`DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public; GRANT ALL ON SCHEMA public TO postgres; GRANT ALL ON SCHEMA public TO public;`)
	require.NoError(t, err, "failed to reset public schema")
}

func TestRunMigrationTest_Postgres(t *testing.T) {
	freePort, err := GetFreePort()
	require.NoError(t, err)

	db, cleanup := startEmbeddedPGOnPort(t, freePort)
	defer cleanup()

	tests := []MigrationTestCase{
		{
			// Solo case: irc.p2p-network.net exists only to host #dpannounce. After the
			// migration we expect a brand-new DarkPeers network with #announce (preserving
			// channel password / enabled / detached) and the old network row removed.
			Name:                "DarkPeers IRC Network Migration - solo",
			MigrationIndex:      80,
			MigrationsUntilName: "80_feed_add_tls_skip_verify",
			MigrationToRun:      "81_irc_update_darkpeers_network",

			SetupData: func(db *sql.DB) error {
				_, err := db.Exec(`
				INSERT INTO irc_network (
					id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
					auth_mechanism, auth_account, auth_password, invite_command,
					use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
					use_proxy, proxy_id, created_at, updated_at
				) VALUES (
					1, true, 'P2P-Network', 'irc.p2p-network.net', 6697, true, false, '', 'darkpeersbot',
					'SASL_PLAIN', 'darkpeersbot', 'nickservpass', '',
					false, '', false, false, NULL,
					false, NULL, '2025-01-01 00:00:00', '2025-01-01 00:00:00'
				)`)
				if err != nil {
					return err
				}

				_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (1, true, '#dpannounce', 'chanpass', false, 1)`)
				return err
			},
			ValidateResult: func(db *sql.DB, t *testing.T) {
				var (
					name, server, authMech, authAccount, authPass, nick string
					port                                                int
					tls                                                 bool
				)
				err := db.QueryRow(`SELECT name, server, port, tls, nick, auth_mechanism, auth_account, auth_password FROM irc_network WHERE server = 'irc.darkpeers.org'`).
					Scan(&name, &server, &port, &tls, &nick, &authMech, &authAccount, &authPass)
				require.NoError(t, err)
				assert.Equal(t, "DarkPeers", name)
				assert.Equal(t, "irc.darkpeers.org", server)
				assert.Equal(t, 6697, port)
				assert.True(t, tls)
				assert.Equal(t, "darkpeersbot", nick)
				assert.Equal(t, "SASL_PLAIN", authMech)
				assert.Equal(t, "darkpeersbot", authAccount)
				assert.Equal(t, "nickservpass", authPass)

				var chanName, chanPass string
				var chanEnabled bool
				err = db.QueryRow(`SELECT c.name, c.password, c.enabled FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE n.server = 'irc.darkpeers.org'`).
					Scan(&chanName, &chanPass, &chanEnabled)
				require.NoError(t, err)
				assert.Equal(t, "#announce", chanName)
				assert.Equal(t, "chanpass", chanPass)
				assert.True(t, chanEnabled)

				var count int
				err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE LOWER(name) = '#dpannounce'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count, "#dpannounce channel should be deleted")

				err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count, "old irc.p2p-network.net row should be deleted")
			},
		},
		{
			// Shared case: another indexer (bit-hdtv) shares the same irc.p2p-network.net
			// row. We must move #dpannounce → DarkPeers but keep the old network row
			// intact for #bithdtv-announce.
			Name:                "DarkPeers IRC Network Migration - shared network",
			MigrationIndex:      80,
			MigrationsUntilName: "80_feed_add_tls_skip_verify",
			MigrationToRun:      "81_irc_update_darkpeers_network",

			SetupData: func(db *sql.DB) error {
				_, err := db.Exec(`
				INSERT INTO irc_network (
					id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
					auth_mechanism, auth_account, auth_password, invite_command,
					use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
					use_proxy, proxy_id, created_at, updated_at
				) VALUES (
					1, true, 'P2P-Network', 'irc.p2p-network.net', 6697, true, false, '', 'sharedbot',
					'SASL_PLAIN', 'sharedbot', 'sharedpass', '',
					false, '', false, false, NULL,
					false, NULL, '2025-01-01 00:00:00', '2025-01-01 00:00:00'
				)`)
				if err != nil {
					return err
				}

				_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (1, true, '#dpannounce', '', false, 1)`)
				if err != nil {
					return err
				}
				_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (2, true, '#bithdtv-announce', '', false, 1)`)
				return err
			},
			ValidateResult: func(db *sql.DB, t *testing.T) {
				var count int

				err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.darkpeers.org' AND name = 'DarkPeers'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count, "DarkPeers network should be created")

				err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE c.name = '#announce' AND n.server = 'irc.darkpeers.org'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count, "#announce should exist on DarkPeers")

				err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count, "P2P-Network row should remain because it still has another channel")

				err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE n.server = 'irc.p2p-network.net'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count, "only #bithdtv-announce should remain on P2P-Network")

				var remaining string
				err = db.QueryRow(`SELECT c.name FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE n.server = 'irc.p2p-network.net'`).Scan(&remaining)
				require.NoError(t, err)
				assert.Equal(t, "#bithdtv-announce", remaining)
			},
		},
		{
			// Negative case: an irc.p2p-network.net row that doesn't have #dpannounce
			// must be left completely untouched (no DarkPeers row, no channel changes).
			Name:                "DarkPeers IRC Network Migration - unrelated network untouched",
			MigrationIndex:      80,
			MigrationsUntilName: "80_feed_add_tls_skip_verify",
			MigrationToRun:      "81_irc_update_darkpeers_network",

			SetupData: func(db *sql.DB) error {
				_, err := db.Exec(`
				INSERT INTO irc_network (
					id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
					auth_mechanism, auth_account, auth_password, invite_command,
					use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
					use_proxy, proxy_id, created_at, updated_at
				) VALUES (
					1, true, 'P2P-Network', 'irc.p2p-network.net', 6697, true, false, '', 'otherbot',
					'NONE', '', '', '',
					false, '', false, false, NULL,
					false, NULL, '2025-01-01 00:00:00', '2025-01-01 00:00:00'
				)`)
				if err != nil {
					return err
				}

				_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (1, true, '#bithdtv-announce', '', false, 1)`)
				return err
			},
			ValidateResult: func(db *sql.DB, t *testing.T) {
				var count int

				err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.darkpeers.org'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count, "no DarkPeers network should be created when #dpannounce is absent")

				err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)

				err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE name = '#bithdtv-announce'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 1, count)
			},
		},
		{
			// Multi-row case: two distinct p2p-network rows (different nicks) each carry
			// #dpannounce. Each must be migrated to its own DarkPeers row keyed by nick;
			// both old rows should be removed since #dpannounce was their only channel.
			Name:                "DarkPeers IRC Network Migration - multiple rows by nick",
			MigrationIndex:      80,
			MigrationsUntilName: "80_feed_add_tls_skip_verify",
			MigrationToRun:      "81_irc_update_darkpeers_network",

			SetupData: func(db *sql.DB) error {
				_, err := db.Exec(`
				INSERT INTO irc_network (
					id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
					auth_mechanism, auth_account, auth_password, invite_command,
					use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
					use_proxy, proxy_id, created_at, updated_at
				) VALUES
					(1, true, 'P2P-Network', 'irc.p2p-network.net', 6697, true, false, '', 'bot_a',
					 'SASL_PLAIN', 'bot_a', 'pass_a', '', false, '', false, false, NULL, false, NULL,
					 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
					(2, true, 'P2P-Network', 'irc.p2p-network.net', 6697, true, false, '', 'bot_b',
					 'SASL_PLAIN', 'bot_b', 'pass_b', '', false, '', false, false, NULL, false, NULL,
					 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`)
				if err != nil {
					return err
				}

				_, err = db.Exec(`INSERT INTO irc_channel (enabled, name, password, detached, network_id) VALUES
					(true, '#dpannounce', '', false, 1),
					(true, '#dpannounce', '', false, 2)`)
				return err
			},
			ValidateResult: func(db *sql.DB, t *testing.T) {
				var count int

				err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.darkpeers.org'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 2, count, "two DarkPeers rows should be created, one per nick")

				var authPass string
				err = db.QueryRow(`SELECT auth_password FROM irc_network WHERE server = 'irc.darkpeers.org' AND nick = 'bot_a'`).Scan(&authPass)
				require.NoError(t, err)
				assert.Equal(t, "pass_a", authPass)

				err = db.QueryRow(`SELECT auth_password FROM irc_network WHERE server = 'irc.darkpeers.org' AND nick = 'bot_b'`).Scan(&authPass)
				require.NoError(t, err)
				assert.Equal(t, "pass_b", authPass)

				err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE c.name = '#announce' AND n.server = 'irc.darkpeers.org'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 2, count)

				err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
				require.NoError(t, err)
				assert.Equal(t, 0, count, "both old p2p-network rows should be deleted")
			},
		},
		{
			Name:                "Indexer archived + deprecation migration",
			MigrationIndex:      84,
			MigrationsUntilName: "84_feeds_add_user_agent",
			MigrationToRun:      "85_add_indexer_archived_and_deprecation",

			SetupData: func(db *sql.DB) error {
				_, err := db.Exec(`INSERT INTO indexer (identifier, name, enabled) VALUES ('fnp', 'FearNoPeer', true)`)
				return err
			},
			ValidateResult: func(db *sql.DB, t *testing.T) {
				var archived bool
				err := db.QueryRow(`SELECT archived FROM indexer WHERE identifier = 'fnp'`).Scan(&archived)
				require.NoError(t, err)
				assert.False(t, archived, "existing rows must default to not archived")

				_, err = db.Exec(`INSERT INTO indexer_deprecation (identifier, name) VALUES ('fnp', 'FearNoPeer') ON CONFLICT (identifier) DO UPDATE SET name = EXCLUDED.name`)
				require.NoError(t, err)
				_, err = db.Exec(`INSERT INTO indexer_deprecation (identifier, name) VALUES ('fnp', 'FearNoPeer (updated)') ON CONFLICT (identifier) DO UPDATE SET name = EXCLUDED.name`)
				require.NoError(t, err, "indexer_deprecation should support ON CONFLICT upsert")

				var count int
				var name string
				err = db.QueryRow(`SELECT COUNT(*), MAX(name) FROM indexer_deprecation WHERE identifier = 'fnp'`).Scan(&count, &name)
				require.NoError(t, err)
				assert.Equal(t, 1, count, "upsert must not create a duplicate row")
				assert.Equal(t, "FearNoPeer (updated)", name, "upsert must update the existing row")
			},
		},
		{
			Name:                "Samaritano IRC port and TLS migration",
			MigrationIndex:      85,
			MigrationsUntilName: "85_add_indexer_archived_and_deprecation",
			MigrationToRun:      "86_irc_update_samaritano_port_and_tls",

			SetupData: func(db *sql.DB) error {
				_, err := db.Exec(`
				INSERT INTO irc_network (
					id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
					auth_mechanism, auth_account, auth_password, invite_command,
					use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
					use_proxy, proxy_id, created_at, updated_at
				) VALUES
					(1, true, 'SamaritanoNet', 'irc.samaritano.cc', 6667, false, false, '', 'bot_a',
					 'NONE', '', '', '', false, '', false, false, NULL, false, NULL,
					 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
					(2, true, 'SamaritanoNet', 'irc.samaritano.cc', 6697, true, false, '', 'bot_b',
					 'NONE', '', '', '', false, '', false, false, NULL, false, NULL,
					 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
					(3, true, 'SamaritanoNet', 'irc.samaritano.cc', 6667, false, false, '', 'bot_b',
					 'NONE', '', '', '', false, '', false, false, NULL, false, NULL,
					 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
					(4, true, 'P2P-Network', 'irc.p2p-network.net', 6667, false, false, '', 'bot_a',
					 'NONE', '', '', '', false, '', false, false, NULL, false, NULL,
					 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`)
				return err
			},
			ValidateResult: func(db *sql.DB, t *testing.T) {
				var port int
				var tls bool

				err := db.QueryRow(`SELECT port, tls FROM irc_network WHERE id = 1`).Scan(&port, &tls)
				require.NoError(t, err)
				assert.Equal(t, 6697, port)
				assert.True(t, tls)

				// Row 3 would collide with row 2 on (server, port, nick) and must be left alone.
				err = db.QueryRow(`SELECT port, tls FROM irc_network WHERE id = 3`).Scan(&port, &tls)
				require.NoError(t, err)
				assert.Equal(t, 6667, port)
				assert.False(t, tls)

				err = db.QueryRow(`SELECT port, tls FROM irc_network WHERE id = 4`).Scan(&port, &tls)
				require.NoError(t, err)
				assert.Equal(t, 6667, port, "other networks must not be touched")
				assert.False(t, tls)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			resetPublicSchema(t, db.Handler)

			log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""}, nil)
			migrate := migrations.PostgresMigrations(db.Handler, log.With().Logger())

			require.NoError(t, migrate.InitVersionTable())

			err := migrate.RunMigrations(migrate.GetUpTo(tc.MigrationsUntilName))
			require.NoError(t, err, "failed to run setup migrations")

			if tc.SetupData != nil {
				require.NoError(t, tc.SetupData(db.Handler), "failed to setup test data")
				// Align SERIAL sequences with any explicit-id rows we inserted so the
				// migration's subsequent inserts don't collide on the primary key.
				_, err := db.Handler.Exec(`SELECT setval(pg_get_serial_sequence('irc_network', 'id'), GREATEST((SELECT COALESCE(MAX(id), 0) FROM irc_network), 1));`)
				require.NoError(t, err, "failed to sync irc_network id sequence")
				_, err = db.Handler.Exec(`SELECT setval(pg_get_serial_sequence('irc_channel', 'id'), GREATEST((SELECT COALESCE(MAX(id), 0) FROM irc_channel), 1));`)
				require.NoError(t, err, "failed to sync irc_channel id sequence")
			}

			target, err := migrate.Get(tc.MigrationToRun)
			require.NoError(t, err, "failed to get target migration")

			err = migrate.RunMigrations([]*migrator.Migration{target})
			require.NoError(t, err, "failed to run target migration")

			if tc.ValidateResult != nil {
				tc.ValidateResult(db.Handler, t)
			}
		})
	}
}
