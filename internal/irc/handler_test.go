// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
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
		if got := modeAdds(c.modes, c.flag); got != c.want {
			t.Errorf("modeAdds(%q, %q) = %v, want %v", c.modes, string(c.flag), got, c.want)
		}
	}
}

// TestOnPartedSetsParted verifies parting a monitored channel moves it to the
// Parted state (previously it went to Idle, leaving Parted a dead state) and
// broadcasts it.
func TestOnPartedSetsParted(t *testing.T) {
	h, sse := newTestHandler()
	sm := addMonitoredChannel(h, "#chan", "")

	sm.OnParted()

	if got := sm.CurrentState(); got != ChannelStateParted {
		t.Fatalf("OnParted from Monitoring: state = %s, want Parted", got)
	}
	if !waitFor(func() bool { return sse.hasStateEvent("#chan", "Parted") }, time.Second) {
		t.Fatal("OnParted should broadcast the Parted state")
	}
}

// TestOnPartedIgnoredWhenNotMonitoring verifies OnParted only acts from Monitoring.
func TestOnPartedIgnoredWhenNotMonitoring(t *testing.T) {
	h, _ := newTestHandler()
	sm := newIdleSM(h, "#chan", "")

	sm.OnParted()

	if got := sm.CurrentState(); got != ChannelStateIdle {
		t.Fatalf("OnParted from Idle changed state to %s", got)
	}
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
	if !found {
		t.Fatal("configured channel #milkie-announce was not registered")
	}
	if !ch.IsEnabled() {
		t.Error("configured channel #milkie-announce should be enabled")
	}
	if ch.announceProcessor == nil {
		t.Error("configured channel #milkie-announce should have an announce processor")
	}

	// the sibling instances' channels must NOT be registered/joined
	for _, name := range []string{"#hdbits.announce", "#seedcore.net"} {
		if _, found := h.channels.Get(name); found {
			t.Errorf("channel %s belongs to another network instance and must not be joined", name)
		}
	}

	if got := h.channels.Len(); got != 1 {
		t.Errorf("handler should track exactly 1 channel, got %d", got)
	}
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
		if !found {
			t.Fatalf("configured channel %s was not registered", name)
		}
		if ch.announceProcessor == nil {
			t.Errorf("configured channel %s should have an announce processor", name)
		}
	}

	if got := h.channels.Len(); got != 2 {
		t.Errorf("handler should track exactly 2 channels, got %d", got)
	}
}
