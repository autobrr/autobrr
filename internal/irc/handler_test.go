// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestModeAdds verifies the mode-string parser used by handleMode, including the
// cases a naive strings.Contains(modes, "+r") check gets wrong.
func TestModeAdds(t *testing.T) {
	cases := []struct {
		modes string
		flag  byte
		want  bool
	}{
		{"+r", 'r', true},
		{"-r", 'r', false},
		{"+ir", 'r', true},
		{"+nrt", 'r', true},  // combined add: substring "+r" is absent but r IS added
		{"+r-x", 'r', true},  // added, then a different mode removed
		{"+x-r", 'r', false}, // r removed
		{"+r-r", 'r', false}, // added then removed -> net removed
		{"-i+r", 'r', true},
		{"+i", 'r', false}, // no r at all
		{"", 'r', false},
		{"+B", 'B', true}, // bot-mode char
		{"-B", 'B', false},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, modeAdds(c.modes, c.flag), "modeAdds(%q, %q)", c.modes, string(c.flag))
	}
}

// TestOnPartedSetsParted verifies parting a monitored channel moves it to the
// Parted state (previously it went to Idle, leaving Parted a dead state) and
// broadcasts it.
func TestOnPartedSetsParted(t *testing.T) {
	h, sse := newTestHandler()
	sm := addMonitoredChannel(h, "#chan", "")

	sm.OnParted()

	require.Equal(t, ChannelStateParted, sm.CurrentState(), "OnParted from Monitoring")
	require.True(t, waitFor(func() bool { return sse.hasStateEvent("#chan", "Parted") }, time.Second), "OnParted should broadcast the Parted state")
}

// TestOnPartedIgnoredWhenNotMonitoring verifies OnParted only acts from Monitoring.
func TestOnPartedIgnoredWhenNotMonitoring(t *testing.T) {
	h, _ := newTestHandler()
	sm := newIdleSM(h, "#chan", "")

	sm.OnParted()

	require.Equal(t, ChannelStateIdle, sm.CurrentState(), "OnParted from Idle changed state")
}

// defWithChannel builds a minimal indexer definition whose IRC settings match a
// single server and announce a single channel. Several of these can share one
// server, mirroring multiple indexers that resolve to the same IRC server.
func defWithChannel(identifier, server, channel, announcer string) *domain.IndexerDefinition {
	return &domain.IndexerDefinition{
		Identifier: identifier,
		IRC: &domain.IndexerIRCV2{
			Network: server,
			Server:  server,
			Channels: []domain.IndexerIRCV2Channel{
				{Name: channel, Announcers: []string{announcer}},
			},
		},
	}
}

// TestInitIndexersOnlyJoinsConfiguredChannels is a regression test for the
// multi-instance bug: three separate network instances share one server
// (different nicks/usernames), so each handler is handed the definitions of all
// three indexers on that server (GetIndexersByIRCNetwork matches by server, not
// network ID). A handler must only create/join the announce channels actually
// configured on its own network instance, not every sibling instance's channel.
func TestInitIndexersOnlyJoinsConfiguredChannels(t *testing.T) {
	h, _ := newTestHandler()
	// this instance is configured with a single announce channel
	h.network.Channels = []domain.IrcChannel{
		{ID: 1, Name: "#milkie-announce", Enabled: true},
	}

	// but it receives the definitions of all indexers on the shared server
	defs := []*domain.IndexerDefinition{
		defWithChannel("hdbits", "irc.p2p-network.net", "#hdbits.announce", "hdbot"),
		defWithChannel("milkie", "irc.p2p-network.net", "#milkie-announce", "milkiebot"),
		defWithChannel("seedcore", "irc.p2p-network.net", "#SeedCore.net", "corebot"),
	}

	h.InitIndexers(defs)

	// the configured channel is registered, enabled, and has an announce
	// processor wired from its matching definition
	ch, found := h.channels.Get("#milkie-announce")
	require.True(t, found, "configured channel #milkie-announce was not registered")
	assert.True(t, ch.IsEnabled(), "configured channel #milkie-announce should be enabled")
	assert.NotNil(t, ch.announceProcessor, "configured channel #milkie-announce should have an announce processor")

	// the sibling instances' channels must NOT be registered/joined
	for _, name := range []string{"#hdbits.announce", "#seedcore.net"} {
		_, found := h.channels.Get(name)
		assert.Falsef(t, found, "channel %s belongs to another network instance and must not be joined", name)
	}

	assert.Equal(t, uintptr(1), h.channels.Len(), "handler should track exactly 1 channel")
}

// TestInitIndexersSharedNetworkJoinsAllConfiguredChannels guards the legitimate
// shared-network case: when a single network instance genuinely serves multiple
// indexers, all of their announce channels are stored on the network and must
// all be joined (the fix must not over-restrict).
func TestInitIndexersSharedNetworkJoinsAllConfiguredChannels(t *testing.T) {
	h, _ := newTestHandler()
	h.network.Channels = []domain.IrcChannel{
		{ID: 1, Name: "#hdbits.announce", Enabled: true},
		{ID: 2, Name: "#milkie-announce", Enabled: true},
	}

	defs := []*domain.IndexerDefinition{
		defWithChannel("hdbits", "irc.p2p-network.net", "#hdbits.announce", "hdbot"),
		defWithChannel("milkie", "irc.p2p-network.net", "#milkie-announce", "milkiebot"),
	}

	h.InitIndexers(defs)

	for _, name := range []string{"#hdbits.announce", "#milkie-announce"} {
		ch, found := h.channels.Get(name)
		require.Truef(t, found, "configured channel %s was not registered", name)
		assert.NotNilf(t, ch.announceProcessor, "configured channel %s should have an announce processor", name)
	}

	assert.Equal(t, uintptr(2), h.channels.Len(), "handler should track exactly 2 channels")
}

// TestInitIndexersNoDefinitionsRegistersConfiguredChannels covers a network whose
// server matches no indexer definition (e.g. an alias hostname): its configured
// channels must still be registered, without an announce processor.
func TestInitIndexersNoDefinitionsRegistersConfiguredChannels(t *testing.T) {
	h, _ := newTestHandler()
	h.network.Channels = []domain.IrcChannel{
		{ID: 1, Name: "#NordicBytes", Enabled: true, Password: "secret"},
	}

	h.InitIndexers(nil)

	ch, found := h.channels.Get("#nordicbytes")
	require.True(t, found, "configured channel #nordicbytes was not registered")

	assert.True(t, ch.IsEnabled())
	assert.Equal(t, int64(1), ch.ID)
	assert.Equal(t, "secret", ch.Password)
	assert.NotNil(t, ch.StateMachine(), "configured channel #nordicbytes should have a state machine")
	assert.Nil(t, ch.announceProcessor, "channel without a matching definition should have no announce processor")
}

func TestInitIndexersMatchesInviteBotNickExactly(t *testing.T) {
	tests := []struct {
		name           string
		inviteCommand  string
		defaultCommand string
		want           string
	}{
		{
			name:           "case-insensitive exact match",
			inviteCommand:  "vOyAgEr enter user key",
			defaultCommand: "Voyager enter USER IRCKEY",
			want:           "vOyAgEr enter user key",
		},
		{
			name:           "prefix is a different nick",
			inviteCommand:  "voy enter user key",
			defaultCommand: "voyager enter USER IRCKEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()
			h.network.InviteCommand = tt.inviteCommand
			h.network.Channels = []domain.IrcChannel{{Name: "#chan", Enabled: true}}

			definition := defWithChannel("test", h.network.Server, "#chan", "announcer")
			definition.IRC.Settings = []domain.IndexerSetting{{Name: "invite_command", Default: tt.defaultCommand}}
			h.InitIndexers([]*domain.IndexerDefinition{definition})

			channel, _, found := h.getChannel("#chan")
			if !found {
				t.Fatal("configured channel was not registered")
			}
			if got := channel.InviteCommand(); got != tt.want {
				t.Errorf("invite command = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCaseMappingChangeReassociatesInviteCommand(t *testing.T) {
	h, _ := newTestHandler()
	h.network.InviteCommand = "Gate~ enter user key"
	h.network.Channels = []domain.IrcChannel{{Name: "#chan", Enabled: true}}

	definition := defWithChannel("test", h.network.Server, "#chan", "announcer")
	definition.IRC.Settings = []domain.IndexerSetting{{Name: "invite_command", Default: "Gate^ enter USER IRCKEY"}}
	h.InitIndexers([]*domain.IndexerDefinition{definition})

	channel, _, found := h.getChannel("#chan")
	if !found {
		t.Fatal("configured channel was not registered")
	}
	if got := channel.InviteCommand(); got != h.network.InviteCommand {
		t.Fatalf("RFC1459 invite command = %q, want %q", got, h.network.InviteCommand)
	}

	h.handleISupport(ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CASEMAPPING=rfc1459-strict", "are supported"},
	})
	if got := channel.InviteCommand(); got != "" {
		t.Fatalf("strict RFC1459 retained non-equivalent invite bot command %q", got)
	}
	if got := channel.StateMachine().inviteCommand; got != "" {
		t.Fatalf("state machine retained non-equivalent invite bot command %q", got)
	}

	h.handleISupport(ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CASEMAPPING=rfc1459", "are supported"},
	})
	if got := channel.InviteCommand(); got != h.network.InviteCommand {
		t.Fatalf("restored RFC1459 invite command = %q, want %q", got, h.network.InviteCommand)
	}
	if got := channel.StateMachine().inviteCommand; got != h.network.InviteCommand {
		t.Fatalf("state machine RFC1459 invite command = %q, want %q", got, h.network.InviteCommand)
	}
}

func TestInviteCommandOwnerStaysWithSelectedChannelDefinition(t *testing.T) {
	h, _ := newTestHandler()
	h.network.InviteCommand = "FirstBot enter one,SecondBot enter two"
	h.network.Channels = []domain.IrcChannel{{Name: "#shared", Enabled: true}}

	first := defWithChannel("first", h.network.Server, "#shared", "announcer")
	first.IRC.Settings = []domain.IndexerSetting{{Name: "invite_command", Default: "FirstBot enter USER IRCKEY"}}
	second := defWithChannel("second", h.network.Server, "#shared", "announcer")
	second.IRC.Settings = []domain.IndexerSetting{{Name: "invite_command", Default: "SecondBot enter USER IRCKEY"}}
	h.InitIndexers([]*domain.IndexerDefinition{first, second})

	channel, _, found := h.getChannel("#shared")
	if !found {
		t.Fatal("shared channel was not registered")
	}
	if got, want := channel.InviteCommand(), "SecondBot enter two"; got != want {
		t.Fatalf("initial invite command = %q, want %q", got, want)
	}

	h.handleISupport(ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CASEMAPPING=ascii", "are supported"},
	})
	if got, want := channel.InviteCommand(), "SecondBot enter two"; got != want {
		t.Fatalf("invite command after remap = %q, want %q", got, want)
	}
}
