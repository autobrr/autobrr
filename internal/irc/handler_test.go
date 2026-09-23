// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

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
	if !found {
		t.Fatal("configured channel #nordicbytes was not registered")
	}
	if !ch.IsEnabled() || ch.ID != 1 || ch.Password != "secret" {
		t.Errorf("configured channel #nordicbytes not configured: id=%d enabled=%v password=%q", ch.ID, ch.IsEnabled(), ch.Password)
	}
	if ch.StateMachine() == nil {
		t.Error("configured channel #nordicbytes should have a state machine")
	}
	if ch.announceProcessor != nil {
		t.Error("channel without a matching definition should have no announce processor")
	}
}
