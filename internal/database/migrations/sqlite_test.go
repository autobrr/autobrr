// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package migrations_test

import (
	"database/sql"
	"testing"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/migrator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// MigrationTestCase defines a test case for a specific migration
type MigrationTestCase struct {
	Name                string
	MigrationsUntilName string
	MigrationToRun      string
	MigrationIndex      int
	SetupData           func(db *sql.DB) error         // Insert test data before migration
	RunMigration        func(db *sql.DB) error         // Run the specific migration
	ValidateResult      func(db *sql.DB, t *testing.T) // Validate the migration worked correctly
}

// runMigrationTestSQLite executes a pluggable migration test
func runMigrationTestSQLite(t *testing.T, testCase MigrationTestCase) {
	// Create temporary database
	//tempDir := t.TempDir()
	//dbPath := filepath.Join(tempDir, "test.db")
	//dbPath := ":memory:"
	//
	//db, err := sql.Open("sqlite", dbPath)
	//require.NoError(t, err)
	//defer func() {
	//	_ = db.Close()
	//}()

	db, cleanup := setupTestSQLiteDB(t)
	defer cleanup()

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})

	migrate := migrations.SQLiteMigrations(db.Handler, log.With().Logger())

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

func TestRunMigrationTest_SQLite(t *testing.T) {
	type fields struct {
		db *sql.DB
	}
	type args struct {
		maxIndex int
		testCase MigrationTestCase
	}
	tests := []struct {
		name   string
		fields fields
		args   MigrationTestCase
		want   string
	}{
		{
			name:   "ULCX IRC Network Migration",
			fields: fields{},
			args: MigrationTestCase{
				Name:                "ULCX IRC Network Migration",
				MigrationIndex:      74,
				MigrationsUntilName: "74_create_sessions_table",
				MigrationToRun:      "75_migrate_ulcx_network",

				SetupData: func(db *sql.DB) error {
					// Insert test IRC network that should be affected by the migration
					_, err := db.Exec(`
					INSERT INTO irc_network (
						id, enabled, name, server, port, tls, pass, nick,
						auth_mechanism, auth_account, auth_password, invite_command,
						use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
						use_proxy, proxy_id, created_at, updated_at
					) VALUES (
						1, 1, 'P2P-Network', 'irc.p2p-network.net', 6667, 0, '', 'test',
						'NONE', '', '', '',
						0, '', 0, 0, NULL,
						0, NULL, '2023-01-01 00:00:00', '2023-01-01 00:00:00'
					)`)
					if err != nil {
						return err
					}

					// Insert ULCX announce channel that should be migrated
					_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (1, 1, '#ulcx-announce', '', 0, 1)`)
					if err != nil {
						return err
					}

					_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (2, 1, '#milkie-announce', '', 0, 1)`)
					if err != nil {
						return err
					}

					return nil
				},
				//RunMigration: func(db *sql.DB) error {
				//	// Run the specific migration
				//	_, err := db.Exec(sqliteMigrations[74])
				//	return err
				//},
				ValidateResult: func(db *sql.DB, t *testing.T) {
					// Check that ULCX network was created
					var count int
					err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE name = 'ULCX' AND server = 'irc.upload.cx'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count, "ULCX network should have been created")

					// Check that new #announce channel was created
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE c.name = '#announce' AND n.name = 'ULCX'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count, "#announce channel should have been created on ULCX network")

					// Check that old #ulcx-announce channel was deleted
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE name = '#ulcx-announce'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 0, count, "#ulcx-announce channel should have been deleted")

					// Check that the old network still exists
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count, "P2P-Network should still exist")
				},
			},
			want: "",
		},

		{
			name:   "fix typo in macro",
			fields: fields{},
			args: MigrationTestCase{
				Name:                "fix typo in macro",
				MigrationIndex:      75,
				MigrationsUntilName: "75_migrate_ulcx_network",
				MigrationToRun:      "76_fix_macro_time",

				SetupData: func(db *sql.DB) error {
					_, err := db.Exec(`INSERT INTO filter (id, name) VALUES (1, 'test')`)
					if err != nil {
						return err
					}

					_, err = db.Exec(`INSERT INTO action (
					id, name, filter_id, exec_cmd, exec_args, 
					watch_folder, category, tags,
					label, save_path, webhook_data
					) VALUES (
						1, 'test action', 1, '/bin/test/"{{ .CurrenTimeUnixMS }}"', '-time="{{ .CurrenTimeUnixMS }}"',
						'/home/test/time-"{{ .CurrenTimeUnixMS }}"', 'category-"{{ .CurrenTimeUnixMS }}"', 'tag-"{{ .CurrenTimeUnixMS }}"',
						'label-"{{ .CurrenTimeUnixMS }}"', '/home/test/time-"{{ .CurrenTimeUnixMS }}"', '{"time"="{{ .CurrenTimeUnixMS }}"}'
					)`)
					if err != nil {
						return err
					}

					_, err = db.Exec(`INSERT INTO filter_external (
					id, name, filter_id, exec_cmd, exec_args, webhook_data
					) VALUES (
						1, 'test action', 1, '/bin/test/"{{ .CurrenTimeUnixMS }}"', '-time="{{ .CurrenTimeUnixMS }}"', '{"time"="{{ .CurrenTimeUnixMS }}"}'
					)`)
					if err != nil {
						return err
					}

					return nil
				},
				//RunMigration: func(db *sql.DB) error {
				//	// Run the specific migration
				//	_, err := db.Exec(sqliteMigrations[74])
				//	return err
				//},
				ValidateResult: func(db *sql.DB, t *testing.T) {
					var execCmd, execArgs, webhookData string
					err := db.QueryRow(`SELECT exec_cmd, exec_args, webhook_data FROM filter_external WHERE id = 1`).Scan(&execCmd, &execArgs, &webhookData)
					require.NoError(t, err)
					assert.Equal(t, `/bin/test/"{{ .CurrentTimeUnixMS }}"`, execCmd, "exe_cmd not matching")
					assert.Equal(t, `-time="{{ .CurrentTimeUnixMS }}"`, execArgs, "exe_args not matching")
					assert.Equal(t, `{"time"="{{ .CurrentTimeUnixMS }}"}`, webhookData, "webhook_data not matching")

					var watchFolder, category, tags, label, savePath string
					err = db.QueryRow(`SELECT exec_cmd, exec_args, watch_folder, category, tags, label, save_path, webhook_data FROM action WHERE id = 1`).Scan(&execCmd, &execArgs, &watchFolder, &category, &tags, &label, &savePath, &webhookData)
					require.NoError(t, err)
					assert.Equal(t, `/bin/test/"{{ .CurrentTimeUnixMS }}"`, execCmd, "exe_cmd not matching")
					assert.Equal(t, `-time="{{ .CurrentTimeUnixMS }}"`, execArgs, "exe_args not matching")
					assert.Equal(t, `/home/test/time-"{{ .CurrentTimeUnixMS }}"`, watchFolder, "watch_folder not matching")
					assert.Equal(t, `category-"{{ .CurrentTimeUnixMS }}"`, category, "category not matching")
					assert.Equal(t, `tag-"{{ .CurrentTimeUnixMS }}"`, tags, "tags not matching")
					assert.Equal(t, `label-"{{ .CurrentTimeUnixMS }}"`, label, "label not matching")
					assert.Equal(t, `/home/test/time-"{{ .CurrentTimeUnixMS }}"`, savePath, "save_path not matching")
					assert.Equal(t, `{"time"="{{ .CurrentTimeUnixMS }}"}`, webhookData, "webhook_data not matching")
				},
			},
			want: "",
		},
		{
			name:   "Aither IRC Network Migration",
			fields: fields{},
			args: MigrationTestCase{
				Name:                "Aither IRC Network Migration",
				MigrationIndex:      83,
				MigrationsUntilName: "83_indexers_update_reelflix_domain",
				MigrationToRun:      "84_indexers_update_aither_irc_auth",

				SetupData: func(db *sql.DB) error {
					// Insert test IRC network that should be affected by the migration
					_, err := db.Exec(`
					INSERT INTO irc_network (
						id, enabled, name, server, port, tls, pass, nick,
						auth_mechanism, auth_account, auth_password, invite_command,
						use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
						use_proxy, proxy_id, created_at, updated_at
					) VALUES (
						1, 1, 'Aither.cc', 'irc.aither.cc', 6697, 1, 'mypass', 'test',
						'NONE', '', '', '',
						0, '', 0, 0, NULL,
						0, NULL, '2025-11-01 00:00:00', '2025-11-01 00:00:00'
					)`)
					if err != nil {
						return err
					}

					return nil
				},
				ValidateResult: func(db *sql.DB, t *testing.T) {
					var pass sql.Null[string]
					var authMech, nickServAccount, nickServPass string
					err := db.QueryRow(`SELECT pass, auth_mechanism, auth_account, auth_password FROM irc_network WHERE server = 'irc.aither.cc'`).Scan(&pass, &authMech, &nickServAccount, &nickServPass)
					require.NoError(t, err)
					assert.Equal(t, "", pass.V, "pass should be empty")
					assert.Equal(t, "SASL_PLAIN", authMech, "auth mechanism should be SASL_PLAIN")
					assert.Equal(t, "test", nickServAccount, "auth_account not matching")
					assert.Equal(t, "mypass", nickServPass, "auth_password not matching")
				},
			},
			want: "",
		},
		{
			// Solo case: irc.p2p-network.net exists only to host #dpannounce. After the
			// migration we expect a brand-new DarkPeers network with #announce (preserving
			// channel password / enabled / detached) and the old network row removed.
			name:   "DarkPeers IRC Network Migration - solo",
			fields: fields{},
			args: MigrationTestCase{
				Name:                "DarkPeers IRC Network Migration - solo",
				MigrationIndex:      90,
				MigrationsUntilName: "90_feed_add_tls_skip_verify",
				MigrationToRun:      "91_irc_update_darkpeers_network",

				SetupData: func(db *sql.DB) error {
					_, err := db.Exec(`
					INSERT INTO irc_network (
						id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
						auth_mechanism, auth_account, auth_password, invite_command,
						use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
						use_proxy, proxy_id, created_at, updated_at
					) VALUES (
						1, 1, 'P2P-Network', 'irc.p2p-network.net', 6697, 1, 0, '', 'darkpeersbot',
						'SASL_PLAIN', 'darkpeersbot', 'nickservpass', '',
						0, '', 0, 0, NULL,
						0, NULL, '2025-01-01 00:00:00', '2025-01-01 00:00:00'
					)`)
					if err != nil {
						return err
					}

					_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (1, 1, '#dpannounce', 'chanpass', 0, 1)`)
					return err
				},
				ValidateResult: func(db *sql.DB, t *testing.T) {
					// New DarkPeers network row created with copied auth/connection details.
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

					// #announce channel attached to DarkPeers, password preserved.
					var chanName, chanPass string
					var chanEnabled bool
					err = db.QueryRow(`SELECT c.name, c.password, c.enabled FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE n.server = 'irc.darkpeers.org'`).
						Scan(&chanName, &chanPass, &chanEnabled)
					require.NoError(t, err)
					assert.Equal(t, "#announce", chanName)
					assert.Equal(t, "chanpass", chanPass)
					assert.True(t, chanEnabled)

					// Old #dpannounce channel is gone.
					var count int
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE LOWER(name) = '#dpannounce'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 0, count, "#dpannounce channel should be deleted")

					// Old empty P2P-Network row deleted because it only existed for DarkPeers.
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 0, count, "old irc.p2p-network.net row should be deleted")
				},
			},
			want: "",
		},
		{
			// Shared case: another indexer (bit-hdtv) shares the same irc.p2p-network.net
			// row. We must move #dpannounce → DarkPeers but keep the old network row
			// intact for #bithdtv-announce.
			name:   "DarkPeers IRC Network Migration - shared network",
			fields: fields{},
			args: MigrationTestCase{
				Name:                "DarkPeers IRC Network Migration - shared network",
				MigrationIndex:      90,
				MigrationsUntilName: "90_feed_add_tls_skip_verify",
				MigrationToRun:      "91_irc_update_darkpeers_network",

				SetupData: func(db *sql.DB) error {
					_, err := db.Exec(`
					INSERT INTO irc_network (
						id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
						auth_mechanism, auth_account, auth_password, invite_command,
						use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
						use_proxy, proxy_id, created_at, updated_at
					) VALUES (
						1, 1, 'P2P-Network', 'irc.p2p-network.net', 6697, 1, 0, '', 'sharedbot',
						'SASL_PLAIN', 'sharedbot', 'sharedpass', '',
						0, '', 0, 0, NULL,
						0, NULL, '2025-01-01 00:00:00', '2025-01-01 00:00:00'
					)`)
					if err != nil {
						return err
					}

					_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (1, 1, '#dpannounce', '', 0, 1)`)
					if err != nil {
						return err
					}
					_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (2, 1, '#bithdtv-announce', '', 0, 1)`)
					return err
				},
				ValidateResult: func(db *sql.DB, t *testing.T) {
					var count int

					// DarkPeers network now exists.
					err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.darkpeers.org' AND name = 'DarkPeers'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count, "DarkPeers network should be created")

					// #announce moved to DarkPeers.
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE c.name = '#announce' AND n.server = 'irc.darkpeers.org'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count, "#announce should exist on DarkPeers")

					// Old p2p-network row still exists.
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count, "P2P-Network row should remain because it still has another channel")

					// Old row still owns the bit-hdtv channel only.
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE n.server = 'irc.p2p-network.net'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count, "only #bithdtv-announce should remain on P2P-Network")

					var remaining string
					err = db.QueryRow(`SELECT c.name FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE n.server = 'irc.p2p-network.net'`).Scan(&remaining)
					require.NoError(t, err)
					assert.Equal(t, "#bithdtv-announce", remaining)
				},
			},
			want: "",
		},
		{
			// Negative case: an irc.p2p-network.net row that doesn't have #dpannounce
			// must be left completely untouched (no DarkPeers row, no channel changes).
			name:   "DarkPeers IRC Network Migration - unrelated network untouched",
			fields: fields{},
			args: MigrationTestCase{
				Name:                "DarkPeers IRC Network Migration - unrelated network untouched",
				MigrationIndex:      90,
				MigrationsUntilName: "90_feed_add_tls_skip_verify",
				MigrationToRun:      "91_irc_update_darkpeers_network",

				SetupData: func(db *sql.DB) error {
					_, err := db.Exec(`
					INSERT INTO irc_network (
						id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
						auth_mechanism, auth_account, auth_password, invite_command,
						use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
						use_proxy, proxy_id, created_at, updated_at
					) VALUES (
						1, 1, 'P2P-Network', 'irc.p2p-network.net', 6697, 1, 0, '', 'otherbot',
						'NONE', '', '', '',
						0, '', 0, 0, NULL,
						0, NULL, '2025-01-01 00:00:00', '2025-01-01 00:00:00'
					)`)
					if err != nil {
						return err
					}

					_, err = db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES (1, 1, '#bithdtv-announce', '', 0, 1)`)
					return err
				},
				ValidateResult: func(db *sql.DB, t *testing.T) {
					var count int

					// No DarkPeers network created.
					err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.darkpeers.org'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 0, count, "no DarkPeers network should be created when #dpannounce is absent")

					// Old network row and its channel still intact.
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count)

					err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE name = '#bithdtv-announce'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 1, count)
				},
			},
			want: "",
		},
		{
			// Multi-row case: two distinct p2p-network rows (different nicks) each carry
			// #dpannounce. Each must be migrated to its own DarkPeers row keyed by nick;
			// both old rows should be removed since #dpannounce was their only channel.
			name:   "DarkPeers IRC Network Migration - multiple rows by nick",
			fields: fields{},
			args: MigrationTestCase{
				Name:                "DarkPeers IRC Network Migration - multiple rows by nick",
				MigrationIndex:      90,
				MigrationsUntilName: "90_feed_add_tls_skip_verify",
				MigrationToRun:      "91_irc_update_darkpeers_network",

				SetupData: func(db *sql.DB) error {
					_, err := db.Exec(`
					INSERT INTO irc_network (
						id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
						auth_mechanism, auth_account, auth_password, invite_command,
						use_bouncer, bouncer_addr, bot_mode, connected, connected_since,
						use_proxy, proxy_id, created_at, updated_at
					) VALUES
						(1, 1, 'P2P-Network', 'irc.p2p-network.net', 6697, 1, 0, '', 'bot_a',
						 'SASL_PLAIN', 'bot_a', 'pass_a', '', 0, '', 0, 0, NULL, 0, NULL,
						 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
						(2, 1, 'P2P-Network', 'irc.p2p-network.net', 6697, 1, 0, '', 'bot_b',
						 'SASL_PLAIN', 'bot_b', 'pass_b', '', 0, '', 0, 0, NULL, 0, NULL,
						 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`)
					if err != nil {
						return err
					}

					_, err = db.Exec(`INSERT INTO irc_channel (enabled, name, password, detached, network_id) VALUES
						(1, '#dpannounce', '', 0, 1),
						(1, '#dpannounce', '', 0, 2)`)
					return err
				},
				ValidateResult: func(db *sql.DB, t *testing.T) {
					var count int

					// Two new DarkPeers rows, one per nick, with auth carried over.
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

					// Two #announce channels, one per DarkPeers row.
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel c JOIN irc_network n ON c.network_id = n.id WHERE c.name = '#announce' AND n.server = 'irc.darkpeers.org'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 2, count)

					// Both old p2p-network rows removed.
					err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.p2p-network.net'`).Scan(&count)
					require.NoError(t, err)
					assert.Equal(t, 0, count, "both old p2p-network rows should be deleted")
				},
			},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runMigrationTestSQLite(t, tt.args)
		})
	}
}

// Helper function to create a test database for integration tests
func setupTestSQLiteDB(t *testing.T) (*database.DB, func()) {
	//tempDir := t.TempDir()
	//dbPath := filepath.Join(tempDir, "test.db")

	dbPath := ":memory:"
	cfg := &domain.Config{
		DatabaseType: "sqlite",
		DatabaseDSN:  dbPath,
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
func TestFullMigrationSequenceSQLite(t *testing.T) {
	db, cleanup := setupTestSQLiteDB(t)
	defer cleanup()

	log := logger.New(&domain.Config{LogLevel: "ERROR", LogPath: ""})

	// This will run all migrations
	migrate := migrations.SQLiteMigrations(db.Handler, log.With().Logger())

	err := migrate.Migrate()
	require.NoError(t, err)

	//// Verify current schema version
	//var version int
	//err = db.Handler.QueryRow("PRAGMA user_version").Scan(&version)
	//require.NoError(t, err)
	//
	//expectedVersion := len(database.sqliteMigrations)
	//assert.Equal(t, expectedVersion, version, "Should be at latest migration version")
}
