// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build irc_integration_test

package harness

import "github.com/autobrr/autobrr/internal/domain"

// MinimalDefinition builds a self-contained indexer definition that parses a
// simple "New torrent: <name> in <category>" announce into a release. It is used
// by tests that care about the connection/join/announce pipeline rather than a
// specific tracker's parse rules; Prepare() is called so the announce processor
// can resolve the channel.
func MinimalDefinition(identifier, channel, announcer string) *domain.IndexerDefinition {
	def := &domain.IndexerDefinition{
		Name:           identifier,
		Identifier:     identifier,
		Implementation: domain.IndexerImplementationIRC,
		BaseURL:        "https://" + identifier + ".test/",
		Enabled:        true,
		IRC: &domain.IndexerIRCV2{
			Network: "Test",
			Server:  "127.0.0.1",
			Channels: []domain.IndexerIRCV2Channel{
				{
					Name:       channel,
					Announcers: []string{announcer},
					Parse: &domain.IndexerIRCV2Parse{
						Type: "single",
						Lines: []domain.IndexerIRCParseLine{
							{Pattern: `New torrent: (?P<torrentName>.+?) in (?P<category>.+)`},
						},
						Match: domain.IndexerIRCV2ParseMatch{
							InfoURL:     "/details?name={{ .torrentName }}",
							ReleaseName: "{{ .torrentName }}",
						},
					},
				},
			},
		},
	}
	def.Prepare()
	return def
}

// InviteDefinition is like MinimalDefinition but declares an invite_command
// setting, so the handler treats the channel as invite-gated and matches the
// network's invite command to this indexer by the bot nick (the first token).
func InviteDefinition(identifier, channel, announcer, inviteBot string) *domain.IndexerDefinition {
	def := MinimalDefinition(identifier, channel, announcer)
	def.IRC.Settings = []domain.IndexerSetting{
		{Name: "invite_command", Default: inviteBot + " enter USERNAME IRCKEY"},
	}
	return def
}
