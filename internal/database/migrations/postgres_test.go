// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package migrations_test

import (
	"bytes"
	"database/sql"
	"fmt"
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

// runMigrationTestPostgres executes a pluggable migration test
func runMigrationTestPostgres(t *testing.T, testCase MigrationTestCase) {
	db, cleanup := setupTestPostgresDB(t)
	defer cleanup()

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})

	migrate := migrations.PostgresMigrations(db.Handler, log.With().Logger())

	err := migrate.InitVersionTable()
	require.NoError(t, err)

	// Run initial schema setup (all migrations up to the target migration - 1)
	m := migrate.GetUpTo(testCase.MigrationsUntilName)

	err = migrate.RunMigrations(m)
	require.NoError(t, err)

	// Insert test data
	if testCase.SetupData != nil {
		err = testCase.SetupData(db.Handler)
		require.NoError(t, err, "Failed to setup test data")
	}

	// Get the migration to run
	currentMigration, err := migrate.Get(testCase.MigrationToRun)
	require.NoError(t, err, "Failed to get migration")

	// Run the specific migration being tested
	err = migrate.RunMigrations([]*migrator.Migration{currentMigration})
	require.NoError(t, err, "Failed to run target migration")

	// Validate the results
	if testCase.ValidateResult != nil {
		testCase.ValidateResult(db.Handler, t)
	}
}

func setupPGTestDB() (*database.DB, func(), error) {
	var (
		dbUsername = "postgres"
		dbPassword = "postgres"
		dbName     = "autobrr"
		dbPort     = 9876
	)

	pgLogger := &bytes.Buffer{}
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username(dbUsername).
		Password(dbPassword).
		Database(dbName).
		Port(uint32(dbPort)).
		Version(embeddedpostgres.V17).
		//RuntimePath("/tmp").
		//BinaryRepositoryURL("https://repo.local/central.proxy").
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

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})
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

func setupTestPostgresDB(t *testing.T) (*database.DB, func()) {
	dsn := "postgres://postgres:postgres@localhost:9876/autobrr?sslmode=disable"

	cfg := &domain.Config{
		DatabaseType:        "postgres",
		DatabaseDSN:         dsn,
		DatabaseAutoMigrate: false,
	}

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})
	db, err := database.NewDB(cfg, log)
	require.NoError(t, err)

	err = db.Open()
	require.NoError(t, err)

	cleanup := func() {
		_ = db.Close()
		//_ = os.Remove(dbPath)
	}

	return db, cleanup
}

// Test full migration sequence
func TestFullMigrationSequencePostgres(t *testing.T) {
	//db, cleanup := setupTestPostgresDB(t)
	db, cleanup, err := setupPGTestDB()
	defer cleanup()
	require.NoError(t, err)

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})

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

	pgLogger := &bytes.Buffer{}
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username(dbUsername).
		Password(dbPassword).
		Database(dbName).
		Port(uint32(port)).
		Version(embeddedpostgres.V17).
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

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})
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
	db, cleanup := startEmbeddedPGOnPort(t, 9879)
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
	}

	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			resetPublicSchema(t, db.Handler)

			log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})
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
