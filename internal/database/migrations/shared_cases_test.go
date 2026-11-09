// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package migrations_test

import (
	"database/sql"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Migration cases shared by the SQLite and Postgres suites. Their SQL sticks to literals and
// syntax that both drivers accept.

const insertIrcNetwork = `INSERT INTO irc_network (
	id, enabled, name, server, port, tls, tls_skip_verify, pass, nick,
	auth_mechanism, auth_account, auth_password, invite_command,
	use_bouncer, bouncer_addr, bot_mode, use_proxy, proxy_id, created_at, updated_at
) VALUES `

func setupNordicBytesNotMigrated(db *sql.DB) error {
	if _, err := db.Exec(insertIrcNetwork + `
		(1, TRUE, 'P2P-Network', 'irc.p2p-network.net', 6697, TRUE, FALSE, '', 'bot_a',
		 'SASL_PLAIN', 'acct_a', 'secret_a', '', FALSE, '', TRUE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(2, TRUE, 'P2P-Network', 'irc.p2p-network.net', 6667, FALSE, FALSE, '', 'bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(3, TRUE, 'P2P-Network', 'irc.p2p-network.net', 6697, TRUE, FALSE, '', 'bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(4, TRUE, 'P2P-Network', 'irc.p2p-network.net', 6697, TRUE, FALSE, '', 'bot_c',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES
		(1, TRUE, '#nordicbytes', 'chan_pw', FALSE, 1),
		(2, TRUE, '#bithdtv', '', FALSE, 1),
		(3, FALSE, '#NordicBytes', '', TRUE, 2),
		(4, TRUE, '#nordicbytes', '', FALSE, 3),
		(5, TRUE, '#ncore', '', FALSE, 4)`)

	return err
}

func validateNordicBytesNotMigrated(db *sql.DB, t *testing.T) {
	var port int
	var tls, botMode bool
	var mechanism, account, password string

	err := db.QueryRow(`SELECT port, tls, bot_mode, auth_mechanism, auth_account, auth_password FROM irc_network WHERE server = 'irc.nordicbytes.org' AND nick = 'bot_a'`).
		Scan(&port, &tls, &botMode, &mechanism, &account, &password)
	require.NoError(t, err)
	assert.Equal(t, 6697, port)
	assert.True(t, tls)
	assert.True(t, botMode)
	assert.Equal(t, "SASL_PLAIN", mechanism)
	assert.Equal(t, "acct_a", account)
	assert.Equal(t, "secret_a", password)

	var chanPassword string
	err = db.QueryRow(`SELECT password FROM irc_channel WHERE network_id = (SELECT id FROM irc_network WHERE server = 'irc.nordicbytes.org' AND nick = 'bot_a') AND name = '#announce'`).Scan(&chanPassword)
	require.NoError(t, err)
	assert.Equal(t, "chan_pw", chanPassword)

	// bot_b had two p2p-network rows; only the lowest id is carried over so UNIQUE (server, port, nick) holds.
	var enabled, detached bool
	err = db.QueryRow(`SELECT n.port, n.tls, c.enabled, c.detached FROM irc_network n JOIN irc_channel c ON c.network_id = n.id WHERE n.server = 'irc.nordicbytes.org' AND n.nick = 'bot_b' AND c.name = '#announce'`).
		Scan(&port, &tls, &enabled, &detached)
	require.NoError(t, err)
	assert.Equal(t, 6697, port)
	assert.True(t, tls, "new network must use TLS even if the p2p-network row did not")
	assert.False(t, enabled)
	assert.True(t, detached)

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.nordicbytes.org'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE id IN (2, 3)`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "p2p-network rows only used by NordicBytes must be removed")

	var name string
	err = db.QueryRow(`SELECT name FROM irc_channel WHERE network_id = 1`).Scan(&name)
	require.NoError(t, err, "shared p2p-network row must keep its other channels and nothing else")
	assert.Equal(t, "#bithdtv", name)

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE network_id = 4 AND name = '#ncore'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "other p2p-network rows must not be touched")
}

func setupNordicBytesAlreadyMigrated(db *sql.DB) error {
	if _, err := db.Exec(insertIrcNetwork + `
		(1, TRUE, 'NordicBytes', 'irc.nordicbytes.org', 6697, TRUE, FALSE, '', 'bot_x',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(2, TRUE, 'P2P-Network', 'irc.p2p-network.net', 6697, TRUE, FALSE, '', 'bot_a',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(3, TRUE, 'P2P-Network', 'irc.p2p-network.net', 6697, TRUE, FALSE, '', 'bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES
		(1, TRUE, '#announce', '', FALSE, 1),
		(2, TRUE, '#nordicbytes', '', FALSE, 2),
		(3, TRUE, '#bithdtv', '', FALSE, 2),
		(4, TRUE, '#nordicbytes', '', FALSE, 3)`)

	return err
}

func validateNordicBytesAlreadyMigrated(db *sql.DB, t *testing.T) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.nordicbytes.org'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "an existing NordicBytes network must not get a duplicate")

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE network_id = 1`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var name string
	err = db.QueryRow(`SELECT name FROM irc_channel WHERE network_id = 2`).Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "#bithdtv", name)

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE id = 3`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "stale p2p-network row must be removed")
}

func setupNordicBytesNotUsed(db *sql.DB) error {
	if _, err := db.Exec(insertIrcNetwork + `
		(1, TRUE, 'P2P-Network', 'irc.p2p-network.net', 6697, TRUE, FALSE, '', 'bot_a',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(2, TRUE, 'P2P-Network', 'irc.p2p-network.net', 6697, TRUE, FALSE, '', 'bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES
		(1, TRUE, '#bithdtv', '', FALSE, 1)`)

	return err
}

func validateNordicBytesNotUsed(db *sql.DB, t *testing.T) {
	var networks, channels int
	err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM irc_network), (SELECT COUNT(*) FROM irc_channel)`).Scan(&networks, &channels)
	require.NoError(t, err)
	assert.Equal(t, 2, networks, "nothing must change without NordicBytes, including empty p2p-network rows")
	assert.Equal(t, 1, channels)
}

func setupBuiltinNotificationExisting(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO notification (name, type, enabled, events, webhook)
		VALUES ('Discord', 'DISCORD', TRUE, '{PUSH_APPROVED}', 'https://discord.example/hook')`)

	return err
}

func validateBuiltinNotification(db *sql.DB, t *testing.T) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM notification WHERE type = 'BUILTIN'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var name string
	var enabled bool
	var events []string
	err = db.QueryRow(`SELECT name, enabled, events FROM notification WHERE type = 'BUILTIN'`).Scan(&name, &enabled, pq.Array(&events))
	require.NoError(t, err)
	assert.Equal(t, "Built-in", name)
	assert.True(t, enabled)
	assert.Equal(t, []string{"PUSH_ERROR", "IRC_DISCONNECTED", "APP_UPDATE_AVAILABLE"}, events)

	_, err = db.Exec(`INSERT INTO notification (name, type, enabled) VALUES ('Second', 'BUILTIN', TRUE)`)
	assert.Error(t, err, "the unique index must reject a second built-in notification")

	_, err = db.Exec(`INSERT INTO notification (name, type, enabled) VALUES ('Another hook', 'WEBHOOK', TRUE)`)
	assert.NoError(t, err, "the unique index must only cover the built-in type")
}

func validateBuiltinNotificationKeepsExisting(db *sql.DB, t *testing.T) {
	validateBuiltinNotification(db, t)

	var webhook string
	err := db.QueryRow(`SELECT webhook FROM notification WHERE type = 'DISCORD'`).Scan(&webhook)
	require.NoError(t, err)
	assert.Equal(t, "https://discord.example/hook", webhook)
}

func setupBuiltinNotificationPresent(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO notification (name, type, enabled, events)
		VALUES ('Existing', 'BUILTIN', FALSE, '{PUSH_REJECTED}')`)

	return err
}

func validateBuiltinNotificationNotDuplicated(db *sql.DB, t *testing.T) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM notification WHERE type = 'BUILTIN'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var name string
	err = db.QueryRow(`SELECT name FROM notification WHERE type = 'BUILTIN'`).Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "Existing", name, "the seed must not replace a built-in row that already exists")
}

func setupPlaintextAPIKeys(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO api_key (name, key, scopes, created_at) VALUES
		('sonarr', 'plain-key-one', '{"releases:read"}', '2025-01-01 00:00:00'),
		('radarr', 'plain-key-two', '{"read","write"}', '2025-02-01 00:00:00'),
		('lidarr', 'plain-key-three', '{}', '2025-03-01 00:00:00')`)

	return err
}

func validateAPIKeysHashed(db *sql.DB, t *testing.T) {
	rows, err := db.Query(`SELECT id, name, key_hash, scopes FROM api_key ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	type row struct {
		id      int
		name    string
		keyHash string
		scopes  []string
	}

	var got []row
	for rows.Next() {
		var r row
		require.NoError(t, rows.Scan(&r.id, &r.name, &r.keyHash, pq.Array(&r.scopes)))
		got = append(got, r)
	}
	require.NoError(t, rows.Err())

	require.Len(t, got, 3)
	integrationScopes := []string{"filters:write", "lists:invoke"}
	assert.Equal(t, row{id: 1, name: "sonarr", keyHash: domain.HashAPIKey("plain-key-one"), scopes: integrationScopes}, got[0])
	assert.Equal(t, row{id: 2, name: "radarr", keyHash: domain.HashAPIKey("plain-key-two"), scopes: integrationScopes}, got[1])
	assert.Equal(t, row{id: 3, name: "lidarr", keyHash: domain.HashAPIKey("plain-key-three"), scopes: integrationScopes}, got[2])

	_, err = db.Exec(`SELECT key FROM api_key`)
	assert.Error(t, err, "the plaintext key column must be gone")

	var id int
	require.NoError(t, db.QueryRow(`INSERT INTO api_key (name, key_hash) VALUES ('readarr', 'mock-hash') RETURNING id`).Scan(&id))
	assert.Equal(t, 4, id)

	_, err = db.Exec(`INSERT INTO api_key (name, key_hash) VALUES ('duplicate', 'mock-hash')`)
	assert.Error(t, err, "key hashes must be unique")
}

func setupReleaseBeforeTableV2(db *sql.DB) error {
	if _, err := db.Exec(`INSERT INTO indexer (id, identifier, implementation, name, enabled) VALUES (5, 'btn', 'irc', 'BTN', TRUE)`); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO "release" (id, filter_status, indexer, torrent_name, torrent_id, hdr) VALUES
		(1, 'FILTER_APPROVED', 'btn', 'That.Show.S01E01.1080p.WEB-DL-GRP', '123', 'DV,HDR'),
		(2, 'FILTER_APPROVED', 'removed-tracker', 'Other.Show.S01E01.1080p.WEB-DL-GRP', '', '')`)

	return err
}

func validateReleaseTableV2(db *sql.DB, t *testing.T) {
	var name, externalID, dynamicRange string
	var indexerID sql.NullInt64

	require.NoError(t, db.QueryRow(`SELECT release_name, external_id, dynamic_range, indexer_id FROM "release" WHERE id = 1`).Scan(&name, &externalID, &dynamicRange, &indexerID))
	assert.Equal(t, "That.Show.S01E01.1080p.WEB-DL-GRP", name)
	assert.Equal(t, "123", externalID)
	assert.Equal(t, "DV,HDR", dynamicRange)
	assert.Equal(t, sql.NullInt64{Int64: 5, Valid: true}, indexerID)

	require.NoError(t, db.QueryRow(`SELECT indexer_id FROM "release" WHERE id = 2`).Scan(&indexerID))
	assert.False(t, indexerID.Valid, "an indexer that no longer exists leaves indexer_id empty")
}

func setupReleaseKeysBackfill(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO "release" (id, release_name, title, sub_title) VALUES
		(1, '[SubsPlease] Frieren - 05 (1080p) [A1B2C3D4].mkv', 'Frieren: Beyond Journey''s End', ''),
		(2, 'Franchise: Example Show S02E05 1080p WEB H264-RG1', 'Franchise: Example Show', 'The Pilot'),
		(3, 'Never.Pushed.S01E01.1080p.WEB-DL-GRP', 'Never Pushed', '')`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`INSERT INTO release_action_status (status, action, type, release_id, rejections) VALUES
		('PUSH_APPROVED', 'push', 'QBITTORRENT', 1, '{}'),
		('PUSH_REJECTED', 'push', 'QBITTORRENT', 2, '{}'),
		('PUSH_APPROVED', 'arr', 'RADARR', 2, '{}'),
		('PUSH_ERROR', 'push', 'QBITTORRENT', 3, '{}')`)

	return err
}

func validateReleaseKeysBackfill(db *sql.DB, t *testing.T) {
	rows, err := db.Query(`SELECT id, name_key, title_key, sub_title_key FROM "release" ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	type keys struct {
		name, title, subTitle string
	}

	got := map[int]keys{}
	for rows.Next() {
		var id int
		var k keys
		require.NoError(t, rows.Scan(&id, &k.name, &k.title, &k.subTitle))
		got[id] = k
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, map[int]keys{
		1: {name: domain.NameKey("[SubsPlease] Frieren - 05 (1080p) [A1B2C3D4].mkv"), title: "frierenbeyondjourneysend"},
		2: {name: domain.NameKey("Franchise: Example Show S02E05 1080p WEB H264-RG1"), title: "franchiseexampleshow", subTitle: "thepilot"},
		3: {},
	}, got)
}

func setupLegacyDuplicateProfiles(db *sql.DB) error {
	_, err := db.Exec(`INSERT INTO release_profile_duplicate (id, name, release_name, hash, title, year, resolution, season) VALUES
		(10, 'name and hash', TRUE, TRUE, FALSE, FALSE, FALSE, FALSE),
		(11, 'hash only', FALSE, TRUE, FALSE, FALSE, FALSE, FALSE),
		(12, 'name only', TRUE, FALSE, FALSE, FALSE, FALSE, FALSE),
		(13, 'no flags', FALSE, FALSE, FALSE, FALSE, FALSE, FALSE),
		(14, 'fields', FALSE, FALSE, TRUE, TRUE, TRUE, FALSE),
		(15, 'name and fields', TRUE, FALSE, TRUE, FALSE, FALSE, TRUE)`)

	return err
}

func validateDuplicateProfilesMapped(db *sql.DB, t *testing.T) {
	rows, err := db.Query(`SELECT id, release_name, media_processing FROM release_profile_duplicate WHERE id >= 10 ORDER BY id`)
	require.NoError(t, err)
	defer rows.Close()

	type flags struct {
		releaseName, mediaProcessing bool
	}

	got := map[int]flags{}
	for rows.Next() {
		var id int
		var f flags
		require.NoError(t, rows.Scan(&id, &f.releaseName, &f.mediaProcessing))
		got[id] = f
	}
	require.NoError(t, rows.Err())

	assert.Equal(t, map[int]flags{
		10: {releaseName: true},
		11: {releaseName: true},
		12: {releaseName: true},
		13: {releaseName: true},
		14: {mediaProcessing: true},
		15: {mediaProcessing: true},
	}, got)
}
