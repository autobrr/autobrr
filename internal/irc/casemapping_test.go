// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bytes"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

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

			h.handleISupport(ircmsg.Message{
				Command: "005",
				Params:  []string{"autobrr", "CASEMAPPING=" + alias, "are supported"},
			})

			_, _, found := h.getChannel("#ANNOUNCE^{")
			assert.True(t, found, "strict RFC1459 should equate [ with { and ASCII letter case")

			_, _, found = h.getChannel("#announce~{")
			assert.False(t, found, "strict RFC1459 must keep ^ and ~ distinct")

			assert.False(t, channel.IsValidAnnouncer("bot~{"), "strict RFC1459 must keep ^ and ~ distinct in nicks")
		})
	}
}

func TestUnknownISupportCaseMappingUsesASCII(t *testing.T) {
	h, _ := newTestHandler()
	channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce^[", true, false, nil)
	channel.RegisterAnnouncers([]string{"Bot^["})
	h.channels.Set(channelMapKey(channel.Name), channel)

	h.handleISupport(ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CASEMAPPING=unicode", "are supported"},
	})

	assert.Equal(t, ircCaseMappingASCII, h.getCaseMapping(), "unknown CASEMAPPING should fall back to ASCII")

	_, _, found := h.getChannel("#announce^{")
	assert.False(t, found, "an unknown explicit CASEMAPPING must not grant RFC1459 channel equivalences")

	assert.False(t, channel.IsValidAnnouncer("bot^{"), "an unknown explicit CASEMAPPING must not grant RFC1459 nick equivalences")
}

func TestValuelessISupportCaseMappingUsesASCII(t *testing.T) {
	for _, token := range []string{"CASEMAPPING", "CASEMAPPING="} {
		t.Run(token, func(t *testing.T) {
			h, _ := newTestHandler()
			h.handleISupport(ircmsg.Message{
				Command: "005",
				Params:  []string{"autobrr", token, "are supported"},
			})

			assert.Equalf(t, ircCaseMappingASCII, h.getCaseMapping(), "%s should fall back to ASCII", token)
		})
	}
}

func TestRemovedISupportCaseMappingRestoresRFC1459(t *testing.T) {
	h, _ := newTestHandler()
	h.setCaseMapping(ircCaseMappingASCII)

	h.handleISupport(ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "-CASEMAPPING", "are supported"},
	})

	assert.Equal(t, ircCaseMappingRFC1459, h.getCaseMapping(), "removed CASEMAPPING should restore the RFC1459 default")
}

func TestChannelMapKeyPreservesUnicodeLowercasing(t *testing.T) {
	assert.Equal(t, "#ännounce", channelMapKey("#ÄNNOUNCE"))
}

func TestRejectedRunDoesNotResetNegotiatedCaseMapping(t *testing.T) {
	h, _ := newTestHandler()
	h.setCaseMapping(ircCaseMappingASCII)

	require.ErrorIs(t, h.Run(), connectionInProgress)

	assert.Equal(t, ircCaseMappingASCII, h.getCaseMapping(), "rejected Run must not reset CASEMAPPING")
}

func TestDisconnectResetsPerConnectionCaseMapping(t *testing.T) {
	h, _ := newTestHandler()
	channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce[", true, false, nil)
	channel.RegisterAnnouncers([]string{"Bot["})
	h.channels.Set(channelMapKey(channel.Name), channel)
	h.setCurrentNick("autobrr")
	h.setCaseMapping(ircCaseMappingASCII)

	h.onDisconnect(ircmsg.Message{})

	assert.Equal(t, ircCaseMappingRFC1459, h.getCaseMapping(), "disconnect should restore the RFC1459 default")
	assert.True(t, channel.IsValidAnnouncer("bot{"), "disconnect did not rebuild announcer keys for the default mapping")
}

func TestBouncerWildcardIsLimitedToEndOfNames(t *testing.T) {
	h, _ := newTestHandler()
	h.network.UseBouncer = true
	h.setCurrentNick("autobrr")

	assert.True(t, h.isOurEndOfNamesTarget("*"), "bouncer wildcard should be accepted for end-of-NAMES")
	assert.False(t, h.isOurCurrentNick("*"), "bouncer wildcard must not be accepted as our nick for arbitrary events")
}

func TestChannelReconcileUsesNegotiatedCaseMapping(t *testing.T) {
	t.Run("rfc1459 equivalent", func(t *testing.T) {
		h, _ := newTestHandler()
		h.AddChannel(domain.IrcChannel{Name: "#extra[", Enabled: false})
		h.AddChannel(domain.IrcChannel{Name: "#extra{", Enabled: false})

		require.Equal(t, uintptr(1), h.channels.Len(), "RFC1459-equivalent channel add should create one entry")

		h.RemoveChannel("#extra{")
		assert.Equal(t, uintptr(0), h.channels.Len(), "equivalent channel remove should delete the entry")
	})

	t.Run("ascii distinct", func(t *testing.T) {
		h, _ := newTestHandler()
		h.setCaseMapping(ircCaseMappingASCII)
		h.AddChannel(domain.IrcChannel{Name: "#extra[", Enabled: false})
		h.AddChannel(domain.IrcChannel{Name: "#extra{", Enabled: false})

		assert.Equal(t, uintptr(2), h.channels.Len(), "ASCII-distinct channel add should create two entries")
	})
}

func TestHandleISupportUpdatesIdentifierComparisons(t *testing.T) {
	h, _ := newTestHandler()
	channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce[", true, false, nil)
	channel.RegisterAnnouncers([]string{"Announce[Bot]"})
	h.channels.Set(channelMapKey(channel.Name), channel)

	_, _, found := h.getChannel("#ANNOUNCE{")
	require.True(t, found, "rfc1459 should treat [ and { as equivalent in channel names")
	require.True(t, channel.IsValidAnnouncer("announce{bot}"), "rfc1459 should treat [ and { as equivalent in announcer nicks")

	h.handleISupport(ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CHANTYPES=#", "CASEMAPPING=ascii", "are supported"},
	})

	_, _, found = h.getChannel("#ANNOUNCE{")
	assert.False(t, found, "ascii CASEMAPPING must keep [ and { distinct in channel names")
	assert.False(t, channel.IsValidAnnouncer("announce{bot}"), "ascii CASEMAPPING must keep [ and { distinct in announcer nicks")
	assert.True(t, channel.IsValidAnnouncer("ANNOUNCE[BOT]"), "ascii CASEMAPPING should still compare ASCII letters case-insensitively")
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
	h.setCurrentNick("Nick[")

	h.onNick(ircmsg.Message{
		Source:  "NICK{!user@host",
		Command: "NICK",
		Params:  []string{"Nick_"},
	})

	assert.Equal(t, "Nick_", h.CurrentNick(), "current nick should follow the server-observed new nick")
}

func TestOtherUsersNickChangeDoesNotUpdateShadow(t *testing.T) {
	h, _ := newTestHandler()
	h.setCurrentNick("autobrr")

	h.onNick(ircmsg.Message{
		Source:  "Someone!user@host",
		Command: "NICK",
		Params:  []string{"Else"},
	})

	assert.Equal(t, "autobrr", h.CurrentNick(), "other user's NICK must not change our current nick")
}

func TestWelcomeTracksServerSelectedNick(t *testing.T) {
	h, _ := newTestHandler()
	h.handleWelcome(ircmsg.Message{Command: "001", Params: []string{"autobrr_"}})

	assert.Equal(t, "autobrr_", h.CurrentNick())
}
