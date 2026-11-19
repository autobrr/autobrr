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

func setupAnthelionNotMigrated(db *sql.DB) error {
	if _, err := db.Exec(insertIrcNetwork + `
		(1, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_a',
		 'SASL_PLAIN', 'acct_a', 'secret_a', 'Sauron bot #ant-announce user_a key_a', FALSE, '', TRUE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(2, TRUE, 'Nebulance', 'irc.nebulance.io', 6667, FALSE, FALSE, '', 'user|bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(3, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(4, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_c',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES
		(1, TRUE, '#ant-announce', 'chan_pw', FALSE, 1),
		(2, TRUE, '#nbl-announce', '', FALSE, 1),
		(3, FALSE, '#ANT-Announce', '', TRUE, 2),
		(4, TRUE, '#ant-announce', '', FALSE, 3),
		(5, TRUE, '#nbl-announce', '', FALSE, 4)`)

	return err
}

func validateAnthelionNotMigrated(db *sql.DB, t *testing.T) {
	var name string
	var port int
	var tls, botMode bool
	var mechanism, account, password, inviteCommand string

	err := db.QueryRow(`SELECT name, port, tls, bot_mode, auth_mechanism, auth_account, auth_password, invite_command FROM irc_network WHERE server = 'irc.anthelion.me' AND nick = 'user|bot_a'`).
		Scan(&name, &port, &tls, &botMode, &mechanism, &account, &password, &inviteCommand)
	require.NoError(t, err)
	assert.Equal(t, "Anthelion", name)
	assert.Equal(t, 6697, port)
	assert.True(t, tls)
	assert.True(t, botMode)
	assert.Equal(t, "SASL_PLAIN", mechanism)
	assert.Equal(t, "acct_a", account)
	assert.Equal(t, "secret_a", password)
	assert.Equal(t, "Sauron bot #ant-announce user_a key_a", inviteCommand)

	var chanPassword string
	err = db.QueryRow(`SELECT password FROM irc_channel WHERE network_id = (SELECT id FROM irc_network WHERE server = 'irc.anthelion.me' AND nick = 'user|bot_a') AND name = '#ant-announce'`).Scan(&chanPassword)
	require.NoError(t, err)
	assert.Equal(t, "chan_pw", chanPassword)

	// user|bot_b had two nebulance rows; only the lowest id is carried over so UNIQUE (server, port, nick) holds.
	var enabled, detached bool
	err = db.QueryRow(`SELECT n.port, n.tls, c.enabled, c.detached FROM irc_network n JOIN irc_channel c ON c.network_id = n.id WHERE n.server = 'irc.anthelion.me' AND n.nick = 'user|bot_b' AND c.name = '#ant-announce'`).
		Scan(&port, &tls, &enabled, &detached)
	require.NoError(t, err)
	assert.Equal(t, 6697, port)
	assert.True(t, tls, "new network must use TLS even if the nebulance row did not")
	assert.False(t, enabled)
	assert.True(t, detached)

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.anthelion.me'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 2, count)

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE id IN (2, 3)`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "nebulance rows only used by Anthelion must be removed")

	err = db.QueryRow(`SELECT name FROM irc_channel WHERE network_id = 1`).Scan(&name)
	require.NoError(t, err, "shared nebulance row must keep its other channels and nothing else")
	assert.Equal(t, "#nbl-announce", name)

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE network_id = 4 AND name = '#nbl-announce'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "other nebulance rows must not be touched")
}

func setupAnthelionAlreadyMigrated(db *sql.DB) error {
	if _, err := db.Exec(insertIrcNetwork + `
		(1, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_x',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(2, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_a',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(3, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES
		(1, TRUE, '#ant-announce', '', FALSE, 1),
		(2, TRUE, '#ant-announce', '', FALSE, 2),
		(3, TRUE, '#nbl-announce', '', FALSE, 2),
		(4, TRUE, '#ant-announce', '', FALSE, 3)`)

	return err
}

func validateAnthelionAlreadyMigrated(db *sql.DB, t *testing.T) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.anthelion.me'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "an existing Anthelion network must not get a duplicate")

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel WHERE network_id = 1`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)

	var name string
	err = db.QueryRow(`SELECT name FROM irc_channel WHERE network_id = 2`).Scan(&name)
	require.NoError(t, err)
	assert.Equal(t, "#nbl-announce", name)

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE id = 3`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "stale nebulance row must be removed")
}

func setupAnthelionNotUsed(db *sql.DB) error {
	if _, err := db.Exec(insertIrcNetwork + `
		(1, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_a',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(2, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES
		(1, TRUE, '#nbl-announce', '', FALSE, 1)`)

	return err
}

func validateAnthelionNotUsed(db *sql.DB, t *testing.T) {
	var networks, channels int
	err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM irc_network), (SELECT COUNT(*) FROM irc_channel)`).Scan(&networks, &channels)
	require.NoError(t, err)
	assert.Equal(t, 2, networks, "nothing must change without Anthelion, including empty nebulance rows")
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
