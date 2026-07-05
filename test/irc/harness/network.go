// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build irc_integration_test

package harness

import "github.com/autobrr/autobrr/internal/domain"

// addressable is satisfied by *ircd.Server; kept as an interface so the harness
// need not import the server package.
type addressable interface {
	Host() string
	Port() int
}

// Network builds an IrcNetwork pointed at the given test server.
func Network(srv addressable, nick string, auth domain.IRCAuth, channels ...domain.IrcChannel) domain.IrcNetwork {
	return domain.IrcNetwork{
		ID:       1,
		Name:     "Test",
		Enabled:  true,
		Server:   srv.Host(),
		Port:     srv.Port(),
		Nick:     nick,
		Auth:     auth,
		Channels: channels,
	}
}

// None is the NONE auth mechanism (no login).
func None() domain.IRCAuth { return domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNone} }

// SASL is SASL PLAIN auth with the given account/password.
func SASL(account, password string) domain.IRCAuth {
	return domain.IRCAuth{Mechanism: domain.IRCAuthMechanismSASLPlain, Account: account, Password: password}
}

// NickServ is NickServ IDENTIFY auth with the given account/password.
func NickServ(account, password string) domain.IRCAuth {
	return domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNickServ, Account: account, Password: password}
}

// Channel is an enabled channel with no password.
func Channel(name string) domain.IrcChannel {
	return domain.IrcChannel{Name: name, Enabled: true}
}

// ChannelWithPassword is an enabled channel that supplies a key on JOIN (+k).
func ChannelWithPassword(name, password string) domain.IrcChannel {
	return domain.IrcChannel{Name: name, Enabled: true, Password: password}
}

// Defs is a convenience for building the definition slice passed to Start.
func Defs(defs ...*domain.IndexerDefinition) []*domain.IndexerDefinition { return defs }
