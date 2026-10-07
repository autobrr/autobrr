// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bytes"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIRCCaseMapping(t *testing.T) {
	tests := []struct {
		name        string
		caseMapping ircCaseMapping
		identifier  string
		want        string
	}{
		{name: "ascii", caseMapping: ircCaseMappingASCII, identifier: `Nick[]\^`, want: `nick[]\^`},
		{name: "strict rfc1459", caseMapping: ircCaseMappingStrictRFC1459, identifier: `Nick[]\^`, want: `nick{}|^`},
		{name: "rfc1459", caseMapping: ircCaseMappingRFC1459, identifier: `Nick[]\^`, want: `nick{}|~`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.caseMapping.fold(tt.identifier), "fold(%q)", tt.identifier)
		})
	}
}

func TestParseIRCCaseMapping(t *testing.T) {
	tests := []struct {
		value string
		want  ircCaseMapping
		ok    bool
	}{
		{value: "rfc1459", want: ircCaseMappingRFC1459, ok: true},
		{value: "rfc1459-strict", want: ircCaseMappingStrictRFC1459, ok: true},
		{value: "strict-rfc1459", want: ircCaseMappingStrictRFC1459, ok: true},
		{value: "ASCII", want: ircCaseMappingASCII, ok: true},
		{value: "", want: ircCaseMappingRFC1459, ok: false},
		{value: "unicode", want: ircCaseMappingRFC1459, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			got, ok := parseIRCCaseMapping(tt.value)
			assert.Equalf(t, tt.want, got, "parseIRCCaseMapping(%q)", tt.value)
			assert.Equalf(t, tt.ok, ok, "parseIRCCaseMapping(%q)", tt.value)
		})
	}
}

func TestStrictCaseMappingAliases(t *testing.T) {
	for _, alias := range []string{"rfc1459-strict", "strict-rfc1459"} {
		t.Run(alias, func(t *testing.T) {
			h, _ := newTestHandler()
			channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce^[", true, false, nil)
			channel.RegisterAnnouncers([]string{"Bot^["})
			h.channels.Set(channelMapKey(channel.Name), channel)

			h.handleISupport(h.client, ircmsg.Message{
				Command: "005",
				Params:  []string{"autobrr", "CASEMAPPING=" + alias, "are supported"},
			})

			_, _, found := h.getChannel("#ANNOUNCE^{")
			assert.True(t, found, "strict RFC1459 should equate [ with { and ASCII letter case")

			_, _, found = h.getChannel("#announce~{")
			assert.False(t, found, "strict RFC1459 must keep ^ and ~ distinct")

			assert.False(t, channel.IsValidAnnouncer("bot~{", h.getCaseMapping()), "strict RFC1459 must keep ^ and ~ distinct in nicks")
		})
	}
}

func TestUnknownISupportCaseMappingUsesASCII(t *testing.T) {
	h, _ := newTestHandler()
	channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce^[", true, false, nil)
	channel.RegisterAnnouncers([]string{"Bot^["})
	h.channels.Set(channelMapKey(channel.Name), channel)

	h.handleISupport(h.client, ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CASEMAPPING=unicode", "are supported"},
	})

	assert.Equal(t, ircCaseMappingASCII, h.getCaseMapping(), "unknown CASEMAPPING should fall back to ASCII")

	_, _, found := h.getChannel("#announce^{")
	assert.False(t, found, "an unknown explicit CASEMAPPING must not grant RFC1459 channel equivalences")

	assert.False(t, channel.IsValidAnnouncer("bot^{", h.getCaseMapping()), "an unknown explicit CASEMAPPING must not grant RFC1459 nick equivalences")
}

func TestValuelessISupportCaseMappingUsesASCII(t *testing.T) {
	for _, token := range []string{"CASEMAPPING", "CASEMAPPING="} {
		t.Run(token, func(t *testing.T) {
			h, _ := newTestHandler()
			h.handleISupport(h.client, ircmsg.Message{
				Command: "005",
				Params:  []string{"autobrr", token, "are supported"},
			})

			assert.Equalf(t, ircCaseMappingASCII, h.getCaseMapping(), "%s should fall back to ASCII", token)
		})
	}
}

func TestRemovedISupportCaseMappingRestoresRFC1459(t *testing.T) {
	h, _ := newTestHandler()
	h.caseMapping = ircCaseMappingASCII

	h.handleISupport(h.client, ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "-CASEMAPPING", "are supported"},
	})

	assert.Equal(t, ircCaseMappingRFC1459, h.getCaseMapping(), "removed CASEMAPPING should restore the RFC1459 default")
}

func TestChannelMapKeyPreservesUnicodeLowercasing(t *testing.T) {
	assert.Equal(t, "#ännounce", channelMapKey("#ÄNNOUNCE"))
}

// TestWelcomeResetsCaseMappingForEachConnection covers an attempt that receives
// 005 and drops before end-of-MOTD: irc-go runs no disconnect callback for it, so
// the next connection's welcome is what has to restore the default.
func TestWelcomeResetsCaseMappingForEachConnection(t *testing.T) {
	h, _ := newTestHandler()
	h.handleWelcome(h.client, ircmsg.Message{Command: "001", Params: []string{"autobrr"}})
	h.handleISupport(h.client, ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CASEMAPPING=ascii", "are supported"},
	})

	h.handleWelcome(h.client, ircmsg.Message{Command: "001", Params: []string{"autobrr"}})
	h.handleISupport(h.client, ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "NETWORK=Test", "are supported"},
	})

	assert.Equal(t, ircCaseMappingRFC1459, h.getCaseMapping(), "a connection without CASEMAPPING inherited the previous attempt's mapping")
}

// TestStaleClientCallbacksDoNotChangeIdentity delivers a stopped connection's
// late callbacks after its replacement has registered.
func TestStaleClientCallbacksDoNotChangeIdentity(t *testing.T) {
	h, _ := newTestHandler()
	stale := &ircevent.Connection{}
	h.client = &ircevent.Connection{}
	h.handleWelcome(h.client, ircmsg.Message{Command: "001", Params: []string{"autobrr"}})

	h.onNick(stale, ircmsg.Message{
		Source:  "autobrr!autobrr@irc.example.test",
		Command: "NICK",
		Params:  []string{"old-session-nick"},
	})
	h.handleWelcome(stale, ircmsg.Message{Command: "001", Params: []string{"old-session-nick"}})
	h.handleISupport(stale, ircmsg.Message{
		Command: "005",
		Params:  []string{"old-session-nick", "CASEMAPPING=ascii", "are supported"},
	})

	assert.Equal(t, "autobrr", h.CurrentNick(), "a stopped connection's callbacks changed the replacement's nick")
	assert.Equal(t, ircCaseMappingRFC1459, h.getCaseMapping(), "a stopped connection's 005 changed the replacement's CASEMAPPING")
}

func TestBouncerWildcardIsLimitedToEndOfNames(t *testing.T) {
	h, _ := newTestHandler()
	h.network.UseBouncer = true
	h.currentNick = "autobrr"

	assert.True(t, h.isOurEndOfNamesTarget("*"), "bouncer wildcard should be accepted for end-of-NAMES")
	assert.False(t, h.isOurCurrentNick("*"), "bouncer wildcard must not be accepted as our nick for arbitrary events")
}

func TestChannelReconcileKeepsConfiguredChannelsSeparate(t *testing.T) {
	h, _ := newTestHandler()
	h.AddChannel(domain.IrcChannel{ID: 1, Name: "#extra[", Enabled: false, Password: "one"})
	h.AddChannel(domain.IrcChannel{ID: 2, Name: "#extra{", Enabled: false, Password: "two"})

	require.Equal(t, uintptr(2), h.channels.Len(), "each configured channel should keep its own entry")

	first, _ := h.channels.Get("#extra[")
	assert.Equal(t, int64(1), first.Snapshot().ID, "adding an RFC1459-equivalent channel reconfigured its sibling")
	assert.Equal(t, "one", first.GetPassword(), "adding an RFC1459-equivalent channel replaced its sibling's password")

	h.UpdateChannel(domain.IrcChannel{ID: 2, Name: "#extra{", Enabled: false, Password: "changed"})
	assert.Equal(t, "one", first.GetPassword(), "updating an RFC1459-equivalent channel changed its sibling")

	h.RemoveChannel("#extra{")
	_, found := h.channels.Get("#extra[")
	assert.True(t, found, "removing an RFC1459-equivalent channel removed its sibling")
}

func TestHandleISupportUpdatesIdentifierComparisons(t *testing.T) {
	h, _ := newTestHandler()
	channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce[", true, false, nil)
	channel.RegisterAnnouncers([]string{"Announce[Bot]"})
	h.channels.Set(channelMapKey(channel.Name), channel)

	_, _, found := h.getChannel("#ANNOUNCE{")
	require.True(t, found, "rfc1459 should treat [ and { as equivalent in channel names")
	require.True(t, channel.IsValidAnnouncer("announce{bot}", h.getCaseMapping()), "rfc1459 should treat [ and { as equivalent in announcer nicks")

	h.handleISupport(h.client, ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CHANTYPES=#", "CASEMAPPING=ascii", "are supported"},
	})

	_, _, found = h.getChannel("#ANNOUNCE{")
	assert.False(t, found, "ascii CASEMAPPING must keep [ and { distinct in channel names")
	assert.False(t, channel.IsValidAnnouncer("announce{bot}", h.getCaseMapping()), "ascii CASEMAPPING must keep [ and { distinct in announcer nicks")
	assert.True(t, channel.IsValidAnnouncer("ANNOUNCE[BOT]", h.getCaseMapping()), "ascii CASEMAPPING should still compare ASCII letters case-insensitively")
}

type recordingAnnounceProcessor struct {
	lines []string
}

func (p *recordingAnnounceProcessor) AddLineToQueue(_ string, line string) error {
	p.lines = append(p.lines, line)
	return nil
}

// TestOnPrivMessageChecksAnnouncerWithCurrentCaseMapping replays the losing
// interleaving of a channel registered while CASEMAPPING changes: the channel is
// built before the change and only becomes visible after it.
func TestOnPrivMessageChecksAnnouncerWithCurrentCaseMapping(t *testing.T) {
	h, _ := newTestHandler()
	processor := &recordingAnnounceProcessor{}
	channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce", true, false, processor)
	channel.RegisterAnnouncers([]string{"Bot["})

	h.handleISupport(h.client, ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CASEMAPPING=ascii", "are supported"},
	})
	h.channels.Set(channelMapKey(channel.Name), channel)

	for _, nick := range []string{"bot{", "BOT["} {
		h.onPrivMessage(ircmsg.Message{
			Source:  nick + "!bot@irc.example.test",
			Command: "PRIVMSG",
			Params:  []string{"#announce", "New Torrent: That.Movie.2017.1080p.BluRay.x264-GROUP from " + nick},
		})
	}

	assert.Equal(t, []string{"New Torrent: That.Movie.2017.1080p.BluRay.x264-GROUP from BOT["}, processor.lines, "announcer check must use the CASEMAPPING negotiated at message time")
}

func TestOnPrivMessageMatchesNickAndChannelCaseInsensitively(t *testing.T) {
	t.Run("direct message target", func(t *testing.T) {
		h, _ := newTestHandler()
		h.network.Nick = "Macley"

		var logs bytes.Buffer
		h.log = zerolog.New(&logs)
		h.onPrivMessage(ircmsg.Message{
			Source:  "SomeUser!user@host",
			Command: "PRIVMSG",
			Params:  []string{"macley", "hello"},
		})

		assert.NotContains(t, logs.String(), "channel not found", "case-variant DM target was treated as a channel")
	})

	t.Run("channel target", func(t *testing.T) {
		h, _ := newTestHandler()
		channel := NewChannel(zerolog.Nop(), h.network.ID, "#mixed", false, false, nil)
		h.channels.Set(channelMapKey(channel.Name), channel)

		h.onPrivMessage(ircmsg.Message{
			Source:  "SomeUser!user@host",
			Command: "PRIVMSG",
			Params:  []string{"#MiXeD", "hello"},
		})

		assert.Len(t, channel.Messages.GetMessages(), 1, "mixed-case channel message should be routed to the channel")
	})
}

func TestOwnNickChangeUsesServerCaseMappingAndUpdatesShadow(t *testing.T) {
	h, _ := newTestHandler()
	h.currentNick = "Nick["

	h.onNick(h.client, ircmsg.Message{
		Source:  "NICK{!user@host",
		Command: "NICK",
		Params:  []string{"Nick_"},
	})

	assert.Equal(t, "Nick_", h.CurrentNick(), "current nick should follow the server-observed new nick")
}

func TestOtherUsersNickChangeDoesNotUpdateShadow(t *testing.T) {
	h, _ := newTestHandler()
	h.currentNick = "autobrr"

	h.onNick(h.client, ircmsg.Message{
		Source:  "Someone!user@host",
		Command: "NICK",
		Params:  []string{"Else"},
	})

	assert.Equal(t, "autobrr", h.CurrentNick(), "other user's NICK must not change our current nick")
}

func TestWelcomeTracksServerSelectedNick(t *testing.T) {
	h, _ := newTestHandler()
	h.handleWelcome(h.client, ircmsg.Message{Command: "001", Params: []string{"autobrr_"}})

	assert.Equal(t, "autobrr_", h.CurrentNick())
}
