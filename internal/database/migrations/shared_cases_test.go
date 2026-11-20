// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package migrations_test

import (
	"database/sql"
	"io/fs"
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
		(1, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, 'server_pw', 'user|bot_a',
		 'SASL_PLAIN', 'acct_a', 'secret_a', 'Sauron bot #ant-announce user_a key_a', TRUE, 'znc.local:6697', TRUE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(2, TRUE, 'Nebulance', 'irc.nebulance.io', 6667, FALSE, FALSE, '', 'user|bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(3, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_b',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(4, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_c',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(5, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_d',
		 'NICKSERV', 'acct_d', 'secret_d', '', FALSE, '', FALSE, FALSE, NULL,
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
	var name, pass string
	var port int
	var tls, botMode, useBouncer bool
	var mechanism, account, password, inviteCommand, bouncerAddr string

	err := db.QueryRow(`SELECT name, port, tls, pass, bot_mode, auth_mechanism, auth_account, auth_password, invite_command, use_bouncer, bouncer_addr FROM irc_network WHERE server = 'irc.anthelion.me' AND nick = 'user|bot_a'`).
		Scan(&name, &port, &tls, &pass, &botMode, &mechanism, &account, &password, &inviteCommand, &useBouncer, &bouncerAddr)
	require.NoError(t, err)
	assert.Equal(t, "Anthelion", name)
	assert.Equal(t, 6697, port)
	assert.True(t, tls)
	assert.True(t, botMode)
	assert.Equal(t, "Sauron bot #ant-announce user_a key_a", inviteCommand)
	assert.Empty(t, pass, "a bouncer row's server password belongs to the bouncer")
	assert.Equal(t, "NONE", mechanism, "a bouncer row's SASL login may belong to the bouncer")
	assert.Empty(t, account)
	assert.Empty(t, password)
	assert.False(t, useBouncer, "the bouncer still points at Nebulance")
	assert.Equal(t, "znc.local:6697", bouncerAddr)

	var chanPassword string
	err = db.QueryRow(`SELECT password FROM irc_channel WHERE network_id = (SELECT id FROM irc_network WHERE server = 'irc.anthelion.me' AND nick = 'user|bot_a') AND name = '#ant-announce'`).Scan(&chanPassword)
	require.NoError(t, err)
	assert.Equal(t, "chan_pw", chanPassword)

	// user|bot_b had two nebulance rows; one is carried over so UNIQUE (server, port, nick) holds,
	// and the enabled channel wins over the disabled one on the lower id.
	var enabled, detached bool
	err = db.QueryRow(`SELECT n.port, n.tls, c.enabled, c.detached FROM irc_network n JOIN irc_channel c ON c.network_id = n.id WHERE n.server = 'irc.anthelion.me' AND n.nick = 'user|bot_b' AND c.name = '#ant-announce'`).
		Scan(&port, &tls, &enabled, &detached)
	require.NoError(t, err)
	assert.Equal(t, 6697, port)
	assert.True(t, tls)
	assert.True(t, enabled)
	assert.False(t, detached)

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

	err = db.QueryRow(`SELECT auth_account FROM irc_network WHERE id = 5`).Scan(&account)
	require.NoError(t, err, "a nebulance row that was already empty must be kept")
	assert.Equal(t, "acct_d", account)
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
	var networks, channels int
	err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM irc_network), (SELECT COUNT(*) FROM irc_channel)`).Scan(&networks, &channels)
	require.NoError(t, err)
	assert.Equal(t, 3, networks, "an existing Anthelion network must skip the whole migration")
	assert.Equal(t, 4, channels)
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

func setupNebulanceMove(db *sql.DB) error {
	if _, err := db.Exec(`INSERT INTO proxy (id, enabled, name, type, addr) VALUES
		(1, TRUE, 'proxy', 'SOCKS5', 'proxy.local:1080'),
		(2, FALSE, 'disabled proxy', 'SOCKS5', 'old.local:1080')`); err != nil {
		return err
	}

	if _, err := db.Exec(insertIrcNetwork + `
		(1, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, 'server_pw', 'user|bot_a',
		 'SASL_PLAIN', 'acct_a', 'secret_a', 'Muffit bot #nbl-announce user_a key_a', FALSE, '', TRUE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(2, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_a',
		 'NONE', '', '', 'Sauron bot #ant-announce user_a key_a', FALSE, '', TRUE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(3, TRUE, 'Nebulance', 'irc.nebulance.io', 6667, FALSE, TRUE, 'zncuser/nbl:zncpw', 'user|bot_b',
		 'NICKSERV', 'acct_b', 'secret_b', 'Muffit bot #nbl-announce user_b key_b', TRUE, 'znc.local:6697', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(4, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_c',
		 'NICKSERV', 'acct_manual', 'secret_manual', 'Muffit bot #nbl-announce user_c key_c', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(5, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_c',
		 'SASL_PLAIN', 'acct_c', 'secret_c', 'Muffit bot #nbl-announce user_c key_c', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(6, FALSE, 'Nebulance', 'irc.nebulance.io', 6667, FALSE, FALSE, '', 'user|bot_d',
		 'SASL_PLAIN', 'acct_old', 'secret_old', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(7, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_d',
		 'NICKSERV', 'acct_d', 'secret_d', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(8, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_e',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(9, FALSE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_f',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(10, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_f',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(11, TRUE, 'Other', 'irc.other.net', 6697, TRUE, FALSE, '', 'user|bot_a',
		 'SASL_PLAIN', 'acct_other', 'secret_other', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(12, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_g',
		 'NONE', '', '', 'Sauron bot #ant-announce user_g key_g', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(13, FALSE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_g',
		 'NONE', '', '', 'Muffit bot #nbl-announce user_g old_key', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(14, TRUE, 'Nebulance', 'irc.nebulance.io', 6667, FALSE, FALSE, '', 'user|bot_g',
		 'NONE', '', '', 'Muffit bot #nbl-announce user_g key_g', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(15, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_h',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(16, FALSE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_h',
		 'SASL_PLAIN', 'acct_h', 'secret_h', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(17, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_i',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(18, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, 'zncuser/nbl:zncpw', 'user|bot_i',
		 'SASL_PLAIN', 'zncuser', 'zncpw', '', TRUE, 'znc.local:6697', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(19, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, 'zncuser/nbl:zncpw', 'user|bot_j',
		 'SASL_PLAIN', 'zncuser', 'zncpw', '', TRUE, 'znc.local:6697', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(20, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_k',
		 'NONE', '', '', 'Sauron bot #ant-announce user_k key_k', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(21, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_k',
		 'NONE', '', '', '/msg Muffit bot #nbl-announce user_k key_k', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(22, FALSE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_m',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(23, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_m',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(24, TRUE, 'Anthelion', 'irc.anthelion.me', 6667, FALSE, FALSE, '', 'user|bot_l',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(25, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_l',
		 'SASL_PLAIN', 'acct_l', 'secret_l', '', FALSE, '', FALSE, TRUE, 1,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(26, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, TRUE, 'zncuser/nbl:zncpw', 'user|bot_n',
		 '', '', 'secret_n', '', TRUE, 'znc.local:6697', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(27, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_o',
		 '', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(28, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_o',
		 'SASL_PLAIN', 'acct_o', 'secret_o', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(29, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_p',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(30, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_p',
		 'SASL_PLAIN', 'acct_broken', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(31, TRUE, 'Nebulance', 'irc.nebulance.io', 6667, FALSE, FALSE, '', 'user|bot_p',
		 'NICKSERV', 'acct_p', 'secret_p', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(32, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_q',
		 'NONE', '', '', 'Sauron bot #ant-announce user_q key_q,Muffit bot #nbl-announce user_q old_key', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(33, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_q',
		 'NONE', '', '', 'Sauron bot #ant-announce user_q stale_key,Muffit bot #nbl-announce user_q new_key', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(34, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_r',
		 'NONE', '', '', 'Muffit bot #nbl-announce user_r own_key', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(35, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_r',
		 'NONE', '', '', 'Muffit bot #nbl-announce user_r nebulance_key', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(36, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_s',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(37, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_s',
		 '', '', 'secret_s', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(38, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, FALSE, FALSE, '', 'user|bot_t',
		 'NONE', '', '', '', TRUE, 'znc.local:6667', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(39, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_t',
		 'NONE', '', '', '', FALSE, '', FALSE, TRUE, 1,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(40, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, NULL, FALSE, '', 'user|bot_u',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(41, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_u',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(42, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_v',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(43, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_v',
		 'NONE', '', '', '', FALSE, '', FALSE, TRUE, 2,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(44, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_w',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(45, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_w',
		 'NONE', '', '', '', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(46, TRUE, 'Anthelion', 'irc.anthelion.me', 6697, TRUE, FALSE, '', 'user|bot_x',
		 'NONE', '', '', 'Sauron', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
		(47, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'user|bot_x',
		 'NONE', '', '', 'Sauron bot #ant-announce user_x key_x', FALSE, '', FALSE, FALSE, NULL,
		 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`); err != nil {
		return err
	}

	_, err := db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES
		(1, TRUE, '#nbl-announce', 'chan_pw', FALSE, 1),
		(2, TRUE, '#ant-announce', '', FALSE, 2),
		(3, TRUE, '#nbl-announce', '', FALSE, 3),
		(4, TRUE, '#ant-announce', '', FALSE, 4),
		(5, TRUE, '#nbl-announce', '', FALSE, 5),
		(6, TRUE, '#NBL-announce', '', FALSE, 6),
		(7, TRUE, '#nbl-announce', '', FALSE, 7),
		(8, TRUE, '#ant-announce', '', FALSE, 8),
		(9, TRUE, '#ant-announce', '', FALSE, 9),
		(10, TRUE, '#nbl-announce', '', FALSE, 10),
		(11, TRUE, '#other', '', FALSE, 11),
		(12, TRUE, '#ant-announce', '', FALSE, 12),
		(13, TRUE, '#nbl-announce', '', FALSE, 13),
		(14, TRUE, '#NBL-announce', '', FALSE, 14),
		(15, TRUE, '#ant-announce', '', FALSE, 15),
		(16, TRUE, '#nbl-announce', '', FALSE, 16),
		(17, TRUE, '#ant-announce', '', FALSE, 17),
		(18, TRUE, '#nbl-announce', '', FALSE, 18),
		(19, TRUE, '#nbl-announce', '', FALSE, 19),
		(20, TRUE, '#ant-announce', '', FALSE, 20),
		(21, FALSE, '#nbl-announce', '', FALSE, 20),
		(22, TRUE, '#nbl-announce', '', FALSE, 21),
		(23, TRUE, '#ant-announce', '', FALSE, 22),
		(24, TRUE, '#nbl-announce', '', FALSE, 22),
		(25, TRUE, '#NBL-announce', '', FALSE, 23),
		(26, TRUE, '#ant-announce', '', FALSE, 24),
		(27, TRUE, '#nbl-announce', '', FALSE, 25),
		(28, TRUE, '#nbl-announce', '', FALSE, 26),
		(29, TRUE, '#nbl-announce', '', FALSE, 28),
		(30, TRUE, '#nbl-announce', '', FALSE, 30),
		(31, TRUE, '#ant-announce', '', FALSE, 32),
		(32, TRUE, '#nbl-announce', '', FALSE, 33),
		(33, TRUE, '#nbl-announce', 'own_key', FALSE, 34),
		(34, TRUE, '#nbl-announce', 'nebulance_key', FALSE, 35),
		(35, TRUE, '#nbl-announce', '', FALSE, 37),
		(36, TRUE, '#nbl-announce', '', FALSE, 39),
		(37, TRUE, '#nbl-announce', '', FALSE, 41),
		(38, TRUE, '#nbl-announce', '', FALSE, 43),
		(39, FALSE, '#nbl-announce', 'stale_key', FALSE, 44),
		(40, TRUE, '#nbl-announce', 'fresh_key', FALSE, 45),
		(41, TRUE, '#nbl-announce', '', FALSE, 47)`)

	return err
}

func validateNebulanceMove(db *sql.DB, t *testing.T) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.nebulance.io'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "no rows may be left on irc.nebulance.io")

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE id IN (1, 5, 6, 10, 13, 14, 16, 18, 21, 23, 25, 28, 30, 31, 33, 35, 37, 39, 41, 43, 45, 47)`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count, "nebulance rows merged into an Anthelion row must be removed")

	networks := []struct {
		id            int
		enabled       bool
		pass          string
		mechanism     string
		account       string
		password      string
		inviteCommand string
		msg           string
	}{
		{2, true, "server_pw", "SASL_PLAIN", "acct_a", "secret_a", "Sauron bot #ant-announce user_a key_a,Muffit bot #nbl-announce user_a key_a", "auth and invite are merged in"},
		{3, true, "", "NICKSERV", "acct_b", "secret_b", "Muffit bot #nbl-announce user_b key_b", "an NBL-only row is moved in place without its bouncer password"},
		{4, true, "", "NICKSERV", "acct_manual", "secret_manual", "Muffit bot #nbl-announce user_c key_c", "manual auth is kept and a known invite is not repeated"},
		{7, true, "", "NICKSERV", "acct_d", "secret_d", "", "the enabled nebulance row is the one moved in place"},
		{8, true, "", "NONE", "", "", "", "an Anthelion row without a nebulance row is left alone"},
		{9, true, "", "NONE", "", "", "", "a running channel enables the row it lands on"},
		{11, true, "", "SASL_PLAIN", "acct_other", "secret_other", "", "other networks must not be touched"},
		{12, true, "", "NONE", "", "", "Sauron bot #ant-announce user_g key_g,Muffit bot #nbl-announce user_g key_g", "the enabled row's Muffit command wins"},
		{15, true, "", "SASL_PLAIN", "acct_h", "secret_h", "", "auth comes over from a disabled row too"},
		{17, true, "", "NONE", "", "", "", "a bouncer row's password and SASL login are not carried"},
		{19, true, "", "NONE", "", "", "", "a bouncer row moved in place drops its password and SASL login"},
		{20, true, "", "NONE", "", "", "Sauron bot #ant-announce user_k key_k,Muffit bot #nbl-announce user_k key_k", "a bot without an invite gets one even when its channel was a duplicate"},
		{22, true, "", "NONE", "", "", "", "a running duplicate channel enables the row"},
		{24, true, "", "SASL_PLAIN", "acct_l", "secret_l", "", "TLS and the proxy come over with the credentials"},
		{26, true, "", "", "", "secret_n", "", "legacy NickServ auth on a bouncer row is kept"},
		{27, true, "", "SASL_PLAIN", "acct_o", "secret_o", "", "an empty mechanism counts as no auth"},
		{29, true, "", "NICKSERV", "acct_p", "secret_p", "", "only a source with usable credentials is picked"},
		{32, true, "", "NONE", "", "", "Sauron bot #ant-announce user_q key_q,Muffit bot #nbl-announce user_q new_key", "the nebulance Muffit command replaces the Anthelion one"},
		{34, true, "", "NONE", "", "", "Muffit bot #nbl-announce user_r own_key", "a row running its own #nbl-announce keeps its Muffit command"},
		{36, true, "", "", "", "secret_s", "", "legacy NickServ auth comes over"},
		{40, true, "", "NONE", "", "", "", "a NULL tls counts as plaintext"},
		{42, true, "", "NONE", "", "", "", ""},
		{44, true, "", "NONE", "", "", "", ""},
		{46, true, "", "NONE", "", "", "Sauron bot #ant-announce user_x key_x", "a command without a message does not claim its bot"},
	}

	for _, tt := range networks {
		var enabled, tls bool
		var port int
		var server, pass, mechanism, account, password, inviteCommand string
		err := db.QueryRow(`SELECT enabled, server, port, tls, pass, auth_mechanism, auth_account, auth_password, invite_command FROM irc_network WHERE id = $1`, tt.id).
			Scan(&enabled, &server, &port, &tls, &pass, &mechanism, &account, &password, &inviteCommand)
		require.NoError(t, err, tt.msg)
		assert.Equal(t, tt.enabled, enabled, tt.msg)
		assert.Equal(t, tt.pass, pass, tt.msg)
		assert.Equal(t, tt.mechanism, mechanism, tt.msg)
		assert.Equal(t, tt.account, account, tt.msg)
		assert.Equal(t, tt.password, password, tt.msg)
		assert.Equal(t, tt.inviteCommand, inviteCommand, tt.msg)
		if tt.id != 11 {
			assert.Equal(t, "irc.anthelion.me", server, tt.msg)
			assert.Equal(t, 6697, port, tt.msg)
			assert.True(t, tls, tt.msg)
		}
	}

	transports := []struct {
		id       int
		tls      bool
		useProxy bool
		proxyID  int64
		msg      string
	}{
		{24, true, true, 1, "a merged row's proxy must keep being used"},
		{38, false, false, 0, "a bouncer row keeps its own transport"},
		{42, true, false, 0, "a disabled proxy is not carried"},
	}

	for _, tt := range transports {
		var tls, useProxy bool
		var proxyID sql.NullInt64
		err := db.QueryRow(`SELECT tls, use_proxy, proxy_id FROM irc_network WHERE id = $1`, tt.id).Scan(&tls, &useProxy, &proxyID)
		require.NoError(t, err, tt.msg)
		assert.Equal(t, tt.tls, tls, tt.msg)
		assert.Equal(t, tt.useProxy, useProxy, tt.msg)
		assert.Equal(t, tt.proxyID, proxyID.Int64, tt.msg)
	}

	var useBouncer, tlsSkipVerify bool
	var bouncerAddr string
	err = db.QueryRow(`SELECT use_bouncer, bouncer_addr, tls_skip_verify FROM irc_network WHERE id = 3`).
		Scan(&useBouncer, &bouncerAddr, &tlsSkipVerify)
	require.NoError(t, err)
	assert.False(t, useBouncer, "the bouncer still points at Nebulance")
	assert.Equal(t, "znc.local:6697", bouncerAddr)
	assert.False(t, tlsSkipVerify, "a skip for the bouncer's certificate must not carry over")

	channels := []struct {
		id        int
		networkID int
		enabled   bool
		msg       string
	}{
		{1, 2, true, "a moved channel keeps its row"},
		{2, 2, true, ""},
		{3, 3, true, "channels of a row moved in place stay put"},
		{4, 4, true, ""},
		{5, 4, true, ""},
		{7, 7, true, ""},
		{8, 8, true, ""},
		{9, 9, false, "channels already on a woken row stay off"},
		{10, 9, true, ""},
		{11, 11, true, ""},
		{12, 12, true, ""},
		{14, 12, true, "the channel from the enabled nebulance row wins a merge"},
		{15, 15, true, ""},
		{16, 15, false, "a channel from a disabled row must not start"},
		{17, 17, true, ""},
		{18, 17, true, ""},
		{19, 19, true, ""},
		{20, 20, true, ""},
		{21, 20, true, "a duplicate runs when its nebulance copy ran"},
		{23, 22, false, "channels already on a woken row stay off"},
		{24, 22, true, "a duplicate on a woken row runs when its nebulance copy ran"},
		{26, 24, true, ""},
		{27, 24, true, ""},
		{28, 26, true, ""},
		{29, 27, true, ""},
		{30, 29, true, ""},
		{31, 32, true, ""},
		{32, 32, true, ""},
		{33, 34, true, ""},
		{35, 36, true, ""},
		{36, 38, true, ""},
		{37, 40, true, ""},
		{38, 42, true, ""},
		{39, 44, true, "a duplicate runs when its nebulance copy ran"},
		{41, 46, true, ""},
	}

	for _, tt := range channels {
		var networkID int
		var enabled bool
		err := db.QueryRow(`SELECT network_id, enabled FROM irc_channel WHERE id = $1`, tt.id).Scan(&networkID, &enabled)
		require.NoError(t, err, "channel %d", tt.id)
		assert.Equal(t, tt.networkID, networkID, "channel %d: %s", tt.id, tt.msg)
		assert.Equal(t, tt.enabled, enabled, "channel %d: %s", tt.id, tt.msg)
	}

	channelKeys := []struct {
		id       int
		password string
		msg      string
	}{
		{1, "chan_pw", "a moved channel keeps its key"},
		{33, "own_key", "a running duplicate keeps its own key"},
		{39, "fresh_key", "a woken duplicate takes the key of the copy that ran"},
	}

	for _, tt := range channelKeys {
		var password string
		err := db.QueryRow(`SELECT password FROM irc_channel WHERE id = $1`, tt.id).Scan(&password)
		require.NoError(t, err, tt.msg)
		assert.Equal(t, tt.password, password, tt.msg)
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_channel`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(channels), count, "only one channel per name may survive a merge")
}

func mustReadMigration(t *testing.T, fsys fs.FS, name string) string {
	t.Helper()

	data, err := fs.ReadFile(fsys, name)
	require.NoError(t, err)

	return string(data)
}

// setupNebulanceBeforeAnthelion stores rows as they were before v1.91.0 and runs the given Anthelion
// network migration on them, so the Nebulance move is tested on what that migration leaves behind.
// Ids start at 101 because Postgres sequences do not see explicit ids and the migration inserts rows.
func setupNebulanceBeforeAnthelion(anthelionMigration string) func(db *sql.DB) error {
	return func(db *sql.DB) error {
		if _, err := db.Exec(insertIrcNetwork + `
			(101, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, 'server_pw', 'shared|bot',
			 'SASL_PLAIN', 'acct_shared', 'secret_shared', 'Sauron bot #ant-announce s k,Muffit bot #nbl-announce s k', FALSE, '', TRUE, FALSE, NULL,
			 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
			(102, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'ant|bot',
			 'NICKSERV', 'acct_ant', 'secret_ant', 'Sauron bot #ant-announce a k', FALSE, '', TRUE, FALSE, NULL,
			 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
			(103, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, FALSE, '', 'nbl|bot',
			 'SASL_PLAIN', 'acct_nbl', 'secret_nbl', 'Muffit bot #nbl-announce n k', FALSE, '', TRUE, FALSE, NULL,
			 '2025-01-01 00:00:00', '2025-01-01 00:00:00'),
			(104, TRUE, 'Nebulance', 'irc.nebulance.io', 6697, TRUE, TRUE, 'zncuser/nbl:zncpw', 'bnc|bot',
			 'SASL_PLAIN', 'zncuser', 'zncpw', 'Sauron bot #ant-announce b k,Muffit bot #nbl-announce b k', TRUE, 'znc.local:6697', TRUE, FALSE, NULL,
			 '2025-01-01 00:00:00', '2025-01-01 00:00:00')`); err != nil {
			return err
		}

		if _, err := db.Exec(`INSERT INTO irc_channel (id, enabled, name, password, detached, network_id) VALUES
			(101, TRUE, '#ant-announce', '', FALSE, 101),
			(102, TRUE, '#nbl-announce', '', FALSE, 101),
			(103, TRUE, '#ant-announce', '', FALSE, 102),
			(104, TRUE, '#nbl-announce', '', FALSE, 103),
			(105, TRUE, '#ant-announce', '', FALSE, 104),
			(106, TRUE, '#nbl-announce', '', FALSE, 104)`); err != nil {
			return err
		}

		_, err := db.Exec(anthelionMigration)

		return err
	}
}

func validateNebulanceAfterAnthelion(db *sql.DB, t *testing.T) {
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM irc_network WHERE server = 'irc.nebulance.io'`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	networks := []struct {
		nick          string
		pass          string
		mechanism     string
		account       string
		password      string
		inviteCommand string
		channels      []string
	}{
		{"shared|bot", "server_pw", "SASL_PLAIN", "acct_shared", "secret_shared", "Sauron bot #ant-announce s k,Muffit bot #nbl-announce s k", []string{"#ant-announce", "#nbl-announce"}},
		{"ant|bot", "", "NICKSERV", "acct_ant", "secret_ant", "Sauron bot #ant-announce a k", []string{"#ant-announce"}},
		{"nbl|bot", "", "SASL_PLAIN", "acct_nbl", "secret_nbl", "Muffit bot #nbl-announce n k", []string{"#nbl-announce"}},
		{"bnc|bot", "", "NONE", "", "", "Sauron bot #ant-announce b k,Muffit bot #nbl-announce b k", []string{"#ant-announce", "#nbl-announce"}},
	}

	for _, tt := range networks {
		var id int
		var enabled, useBouncer, tlsSkipVerify bool
		var pass, mechanism, account, password, inviteCommand string
		err := db.QueryRow(`SELECT id, enabled, use_bouncer, tls_skip_verify, pass, auth_mechanism, auth_account, auth_password, invite_command FROM irc_network WHERE server = 'irc.anthelion.me' AND port = 6697 AND nick = $1`, tt.nick).
			Scan(&id, &enabled, &useBouncer, &tlsSkipVerify, &pass, &mechanism, &account, &password, &inviteCommand)
		require.NoError(t, err, tt.nick)
		assert.True(t, enabled, tt.nick)
		assert.False(t, useBouncer, tt.nick)
		assert.False(t, tlsSkipVerify, tt.nick)
		assert.Equal(t, tt.pass, pass, tt.nick)
		assert.Equal(t, tt.mechanism, mechanism, tt.nick)
		assert.Equal(t, tt.account, account, tt.nick)
		assert.Equal(t, tt.password, password, tt.nick)
		assert.Equal(t, tt.inviteCommand, inviteCommand, tt.nick)

		rows, err := db.Query(`SELECT name FROM irc_channel WHERE network_id = $1 AND enabled`, id)
		require.NoError(t, err)

		var names []string
		for rows.Next() {
			var name string
			require.NoError(t, rows.Scan(&name))
			names = append(names, name)
		}
		require.NoError(t, rows.Err())
		rows.Close()

		assert.ElementsMatch(t, tt.channels, names, tt.nick)
	}

	err = db.QueryRow(`SELECT COUNT(*) FROM irc_network`).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, len(networks), count)
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
