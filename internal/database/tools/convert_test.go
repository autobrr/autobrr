// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package tools

import (
	"bytes"
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/database/migrations"
	"github.com/autobrr/autobrr/internal/domain"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const seedSQLite = `
INSERT INTO users (username, password) VALUES ('admin', 'hash');
INSERT INTO proxy (name, type, addr, enabled) VALUES ('proxy', 'SOCKS5', 'socks5://127.0.0.1:1080', TRUE);
INSERT INTO indexer (identifier, identifier_external, implementation, base_url, enabled, name, settings, use_proxy, proxy_id)
VALUES ('ggn', 'GazelleGames', 'irc', 'https://mock.local/', TRUE, 'GazelleGames', '{"api_key":"mock-key"}', TRUE, 1);
INSERT INTO indexer_deprecation (identifier, name, reason, deprecated_at) VALUES ('oldtracker', 'OldTracker', 'closed', '2026-01-02T03:04:05Z');
INSERT INTO irc_network (enabled, name, server, port, tls, nick, use_proxy, proxy_id) VALUES (TRUE, 'GGn', 'irc.mock.local', 6697, TRUE, 'autobrr', FALSE, 1);
INSERT INTO irc_channel (enabled, name, network_id) VALUES (TRUE, '#announce', 1);
UPDATE release_profile_duplicate SET title = TRUE WHERE id = 1;
INSERT INTO release_profile_duplicate (id, name, title, year) VALUES (4, 'Custom', TRUE, TRUE);
INSERT INTO filter (enabled, name, delay, smart_episode, shows, release_types_match, tags, release_profile_duplicate_id)
VALUES (TRUE, 'filter one', 10, TRUE, 'That Show', '{Album,"Single EP"}', 'tag1,tag2', 4);
INSERT INTO filter (enabled, name) VALUES (FALSE, 'filter two');
INSERT INTO filter_external (name, filter_id, enabled, type, exec_cmd) VALUES ('ext', 1, TRUE, 'EXEC', '/bin/true');
INSERT INTO filter_indexer (filter_id, indexer_id) VALUES (1, 1);
INSERT INTO client (name, type, enabled, host, port, tls, settings) VALUES ('qbit', 'QBITTORRENT', TRUE, 'localhost', 8080, FALSE, '{}');
INSERT INTO action (name, type, enabled, filter_id, client_id) VALUES ('push', 'QBITTORRENT', TRUE, 1, 1);
INSERT INTO "release" (filter_status, rejections, indexer, filter, protocol, implementation, timestamp, torrent_name, size, title, filter_id)
VALUES ('FILTER_APPROVED', '{"size too big"}', 'ggn', 'filter one', 'torrent', 'IRC', '2026-10-21T10:11:12+02:00', 'That.Show.S01E01.1080p.WEB-DL-GRP', 1234, 'That Show', 1);
INSERT INTO release_action_status (status, action, action_id, type, client, filter, filter_id, rejections, timestamp, release_id)
VALUES ('PUSH_APPROVED', 'push', 1, 'QBITTORRENT', 'qbit', 'filter one', 1, '{}', '2026-10-21T10:11:13+02:00', 1);
INSERT INTO release_cleanup_job (name, enabled, schedule, older_than, last_run) VALUES ('cleanup', TRUE, '0 * * * *', 24, '2026-10-20T00:00:00Z');
INSERT INTO notification (name, type, enabled, events, webhook) VALUES ('discord', 'DISCORD', TRUE, '{PUSH_APPROVED,PUSH_REJECTED}', 'https://mock.local/hook');
INSERT INTO filter_notification (filter_id, notification_id, events) VALUES (1, 2, '{PUSH_APPROVED}');
INSERT INTO notification_inbox (event, title, message, filter_id, rejections, read_at, created_at)
VALUES ('PUSH_ERROR', 'Push error', 'could not push', 1, '{"one","two"}', '2026-10-21T11:00:00Z', '2026-10-21T10:00:00Z');
INSERT INTO feed (indexer, name, type, enabled, url, interval, capabilities, indexer_id)
VALUES ('ggn', 'GGn feed', 'TORZNAB', TRUE, 'https://mock.local/api', 15, '{"search":{"available":"yes"}}', 1);
INSERT INTO feed_cache (feed_id, key, value, ttl) VALUES (1, 'guid', 'value', '2026-11-21T10:00:00Z');
INSERT INTO api_key (name, key, scopes) VALUES ('key', 'mock-key', '{read,write}');
INSERT INTO list (name, type, enabled, client_id, tags_included) VALUES ('list', 'RADARR', TRUE, 1, '{a,b}');
INSERT INTO list_filter (list_id, filter_id) VALUES (1, 1);
`

func TestConverter_RoundTrip(t *testing.T) {
	log := zerolog.Nop()
	dir := t.TempDir()
	postgresURL := startEmbeddedPG(t)

	sourcePath := filepath.Join(dir, "source.db")
	targetPath := filepath.Join(dir, "target.db")

	// Setup
	source, err := sql.Open("sqlite", sourcePath)
	require.NoError(t, err)
	source.SetMaxOpenConns(1)
	require.NoError(t, migrations.SQLiteMigrations(source, log).Migrate())
	_, err = source.Exec(seedSQLite)
	require.NoError(t, err)
	require.NoError(t, source.Close())

	// Execute
	err = NewConverter(log, sourcePath, postgresURL).Convert(t.Context(), Opts{})
	require.NoError(t, err)

	err = NewConverter(log, targetPath, postgresURL).ConvertToSQLite(t.Context(), Opts{})
	require.NoError(t, err)

	sourceCounts := tableCounts(t, "sqlite", sourcePath)
	assert.Equal(t, sourceCounts, tableCounts(t, "postgres", postgresURL), "postgres row counts")
	assert.Equal(t, sourceCounts, tableCounts(t, "sqlite", targetPath), "sqlite row counts")

	for dbType, dsn := range map[string]string{"postgres": postgresURL, "sqlite": targetPath} {
		t.Run(fmt.Sprintf("Read_Converted [%s]", dbType), func(t *testing.T) {
			db := openAppDB(t, dsn)

			filter, err := database.NewFilterRepo(log, db).FindByID(t.Context(), 1)
			require.NoError(t, err)
			assert.True(t, filter.Enabled)
			assert.True(t, filter.SmartEpisode)
			assert.Equal(t, 10, filter.Delay)
			assert.Equal(t, []string{"Album", "Single EP"}, filter.MatchReleaseTypes)

			keys, err := database.NewAPIRepo(log, db).GetAllAPIKeys(t.Context())
			require.NoError(t, err)
			require.Len(t, keys, 1)
			assert.Equal(t, []string{"read", "write"}, keys[0].Scopes)

			notifications, err := database.NewNotificationRepo(log, db).List(t.Context())
			require.NoError(t, err)
			require.Len(t, notifications, 2)
			assert.Equal(t, domain.NotificationType("BUILTIN"), notifications[0].Type)
			assert.Equal(t, []string{"PUSH_APPROVED", "PUSH_REJECTED"}, notifications[1].Events)

			inbox, err := database.NewNotificationInboxRepo(log, db).Find(t.Context(), domain.InboxQueryParams{Limit: 10})
			require.NoError(t, err)
			require.Len(t, inbox.Data, 1)
			assert.Equal(t, []string{"one", "two"}, inbox.Data[0].Rejections)
			assert.True(t, inbox.Data[0].CreatedAt.Equal(time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)), "created_at %s", inbox.Data[0].CreatedAt)

			feed, err := database.NewFeedRepo(log, db).FindByID(t.Context(), 1)
			require.NoError(t, err)
			assert.NotNil(t, feed.Capabilities)

			deprecations, err := database.NewIndexerRepo(log, db).ListDeprecations(t.Context())
			require.NoError(t, err)
			require.Len(t, deprecations, 1)
			assert.Equal(t, "oldtracker", deprecations[0].Identifier)

			// ids must continue after the copied rows
			var id int
			require.NoError(t, db.Handler.QueryRow(`INSERT INTO filter (name) VALUES ('filter three') RETURNING id`).Scan(&id))
			assert.Equal(t, 3, id)
		})
	}

	t.Run("SQLite_Stores_Text", func(t *testing.T) {
		db := openAppDB(t, targetPath)

		var scopesType, timestamp string
		require.NoError(t, db.Handler.QueryRow(`SELECT typeof(scopes) FROM api_key`).Scan(&scopesType))
		require.NoError(t, db.Handler.QueryRow(`SELECT timestamp FROM "release"`).Scan(&timestamp))

		assert.Equal(t, "text", scopesType)
		assert.Equal(t, "2026-10-21T08:11:12Z", timestamp)
	})

	t.Run("Convert_Fails_Target_Not_Empty", func(t *testing.T) {
		err := NewConverter(log, sourcePath, postgresURL).Convert(t.Context(), Opts{})
		assert.ErrorContains(t, err, "not empty")
	})
}

func TestGetTables_CoversSchema(t *testing.T) {
	log := zerolog.Nop()
	postgresURL := startEmbeddedPG(t)

	sqliteDB, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer sqliteDB.Close()
	sqliteDB.SetMaxOpenConns(1)
	require.NoError(t, migrations.SQLiteMigrations(sqliteDB, log).Migrate())

	postgresDB, err := sql.Open("postgres", postgresURL)
	require.NoError(t, err)
	defer postgresDB.Close()
	require.NoError(t, migrations.PostgresMigrations(postgresDB, log).Migrate())

	ignored := []string{"sessions", "schema_migrations"}
	want := slices.Sorted(slices.Values(append(GetTables(), ignored...)))

	assert.Equal(t, want, listTables(t, sqliteDB, `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`), "sqlite tables")
	assert.Equal(t, want, listTables(t, postgresDB, `SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename`), "postgres tables")

	// dead columns the SQLite release table rebuilds dropped but Postgres never did
	postgresOnly := map[string][]string{
		"release": {"artists", "format", "freeleech", "freeleech_percent", "has_cue", "has_log", "is_scene", "log_score", "quality", "raw", "unrated"},
	}

	for _, table := range GetTables() {
		sqliteColumns, err := tableColumns(t.Context(), conn{db: sqliteDB, engine: engineSQLite}, table)
		require.NoError(t, err)

		postgresColumns, err := tableColumns(t.Context(), conn{db: postgresDB, engine: enginePostgres}, table)
		require.NoError(t, err)

		postgresColumns = slices.DeleteFunc(postgresColumns, func(col string) bool {
			return slices.Contains(postgresOnly[table], col)
		})

		slices.Sort(sqliteColumns)
		slices.Sort(postgresColumns)
		assert.Equal(t, postgresColumns, sqliteColumns, "columns of table %s", table)
	}
}

func listTables(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()

	rows, err := db.Query(query)
	require.NoError(t, err)
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables = append(tables, name)
	}

	return tables
}

func tableCounts(t *testing.T, driver, dsn string) map[string]int {
	t.Helper()

	db, err := sql.Open(driver, dsn)
	require.NoError(t, err)
	defer db.Close()

	counts := make(map[string]int)
	for _, table := range GetTables() {
		var count int
		require.NoError(t, db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %q", table)).Scan(&count))
		counts[table] = count
	}

	return counts
}

func openAppDB(t *testing.T, dsn string) *database.DB {
	t.Helper()

	db, err := database.NewDB(&domain.Config{DatabaseDSN: dsn, LogLevel: "ERROR"}, zerolog.Nop())
	require.NoError(t, err)
	require.NoError(t, db.Open())
	t.Cleanup(func() { _ = db.Close() })

	return db
}

// startEmbeddedPG boots an empty embedded Postgres for the test and returns its DSN.
func startEmbeddedPG(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}

	pgLogger := &bytes.Buffer{}
	pg := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().
		Username("testdb").
		Password("testdb").
		Database("autobrr").
		Port(uint32(port)).
		Version(embeddedpostgres.V17).
		BinariesPath(filepath.Join(cacheDir, "autobrr", "embedded-postgres", "v17-tools")).
		RuntimePath(t.TempDir()).
		StartTimeout(45 * time.Second).
		Logger(pgLogger))
	require.NoError(t, pg.Start())

	t.Cleanup(func() {
		if err := pg.Stop(); err != nil {
			t.Logf("failed to stop embedded postgres: %v\npg log:\n%s", err, pgLogger.String())
		}
	})

	return fmt.Sprintf("postgres://testdb:testdb@localhost:%d/autobrr?sslmode=disable", port)
}
