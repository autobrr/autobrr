// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"bytes"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
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
			if got := tt.caseMapping.fold(tt.identifier); got != tt.want {
				t.Errorf("fold(%q) = %q, want %q", tt.identifier, got, tt.want)
			}
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
			if got != tt.want || ok != tt.ok {
				t.Fatalf("parseIRCCaseMapping(%q) = (%d, %v), want (%d, %v)", tt.value, got, ok, tt.want, tt.ok)
			}
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

			if _, _, found := h.getChannel("#ANNOUNCE^{"); !found {
				t.Fatal("strict RFC1459 should equate [ with { and ASCII letter case")
			}
			if _, _, found := h.getChannel("#announce~{"); found {
				t.Fatal("strict RFC1459 must keep ^ and ~ distinct")
			}
			if channel.IsValidAnnouncer("bot~{") {
				t.Fatal("strict RFC1459 must keep ^ and ~ distinct in nicks")
			}
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

	if got := h.getCaseMapping(); got != ircCaseMappingASCII {
		t.Fatalf("unknown CASEMAPPING selected %d, want ASCII", got)
	}
	if _, _, found := h.getChannel("#announce^{"); found {
		t.Fatal("an unknown explicit CASEMAPPING must not grant RFC1459 channel equivalences")
	}
	if channel.IsValidAnnouncer("bot^{") {
		t.Fatal("an unknown explicit CASEMAPPING must not grant RFC1459 nick equivalences")
	}
}

func TestValuelessISupportCaseMappingUsesASCII(t *testing.T) {
	for _, token := range []string{"CASEMAPPING", "CASEMAPPING="} {
		t.Run(token, func(t *testing.T) {
			h, _ := newTestHandler()
			h.handleISupport(ircmsg.Message{
				Command: "005",
				Params:  []string{"autobrr", token, "are supported"},
			})

			if got := h.getCaseMapping(); got != ircCaseMappingASCII {
				t.Fatalf("%s selected %d, want ASCII", token, got)
			}
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

	if got := h.getCaseMapping(); got != ircCaseMappingRFC1459 {
		t.Fatalf("removed CASEMAPPING selected %d, want RFC1459 default", got)
	}
}

func TestChannelMapKeyPreservesUnicodeLowercasing(t *testing.T) {
	if got, want := channelMapKey("#ÄNNOUNCE"), "#ännounce"; got != want {
		t.Fatalf("channelMapKey() = %q, want %q", got, want)
	}
}

func TestRejectedRunDoesNotResetNegotiatedCaseMapping(t *testing.T) {
	h, _ := newTestHandler()
	h.setCaseMapping(ircCaseMappingASCII)

	if err := h.Run(); err != connectionInProgress {
		t.Fatalf("Run() error = %v, want %v", err, connectionInProgress)
	}
	if got := h.getCaseMapping(); got != ircCaseMappingASCII {
		t.Fatalf("rejected Run reset CASEMAPPING to %d", got)
	}
}

func TestDisconnectResetsPerConnectionCaseMapping(t *testing.T) {
	h, _ := newTestHandler()
	channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce[", true, false, nil)
	channel.RegisterAnnouncers([]string{"Bot["})
	h.channels.Set(channelMapKey(channel.Name), channel)
	h.setCurrentNick("autobrr")
	h.setCaseMapping(ircCaseMappingASCII)

	h.onDisconnect(ircmsg.Message{})

	if got := h.getCaseMapping(); got != ircCaseMappingRFC1459 {
		t.Fatalf("disconnect left CASEMAPPING at %d, want RFC1459 default", got)
	}
	if !channel.IsValidAnnouncer("bot{") {
		t.Fatal("disconnect did not rebuild announcer keys for the default mapping")
	}
}

func TestBouncerWildcardIsLimitedToEndOfNames(t *testing.T) {
	h, _ := newTestHandler()
	h.network.UseBouncer = true
	h.setCurrentNick("autobrr")

	if !h.isOurEndOfNamesTarget("*") {
		t.Fatal("bouncer wildcard should be accepted for end-of-NAMES")
	}
	if h.isOurCurrentNick("*") {
		t.Fatal("bouncer wildcard must not be accepted as our nick for arbitrary events")
	}
}

func TestChannelReconcileUsesNegotiatedCaseMapping(t *testing.T) {
	t.Run("rfc1459 equivalent", func(t *testing.T) {
		h, _ := newTestHandler()
		h.AddChannel(domain.IrcChannel{Name: "#extra[", Enabled: false})
		h.AddChannel(domain.IrcChannel{Name: "#extra{", Enabled: false})

		if got := h.channels.Len(); got != 1 {
			t.Fatalf("RFC1459-equivalent channel add created %d entries, want 1", got)
		}
		h.RemoveChannel("#extra{")
		if got := h.channels.Len(); got != 0 {
			t.Fatalf("equivalent channel remove left %d entries", got)
		}
	})

	t.Run("ascii distinct", func(t *testing.T) {
		h, _ := newTestHandler()
		h.setCaseMapping(ircCaseMappingASCII)
		h.AddChannel(domain.IrcChannel{Name: "#extra[", Enabled: false})
		h.AddChannel(domain.IrcChannel{Name: "#extra{", Enabled: false})

		if got := h.channels.Len(); got != 2 {
			t.Fatalf("ASCII-distinct channel add created %d entries, want 2", got)
		}
	})
}

func TestHandleISupportUpdatesIdentifierComparisons(t *testing.T) {
	h, _ := newTestHandler()
	channel := NewChannel(zerolog.Nop(), h.network.ID, "#announce[", true, false, nil)
	channel.RegisterAnnouncers([]string{"Announce[Bot]"})
	h.channels.Set(channelMapKey(channel.Name), channel)

	if _, _, found := h.getChannel("#ANNOUNCE{"); !found {
		t.Fatal("rfc1459 should treat [ and { as equivalent in channel names")
	}
	if !channel.IsValidAnnouncer("announce{bot}") {
		t.Fatal("rfc1459 should treat [ and { as equivalent in announcer nicks")
	}

	h.handleISupport(ircmsg.Message{
		Command: "005",
		Params:  []string{"autobrr", "CHANTYPES=#", "CASEMAPPING=ascii", "are supported"},
	})

	if _, _, found := h.getChannel("#ANNOUNCE{"); found {
		t.Fatal("ascii CASEMAPPING must keep [ and { distinct in channel names")
	}
	if channel.IsValidAnnouncer("announce{bot}") {
		t.Fatal("ascii CASEMAPPING must keep [ and { distinct in announcer nicks")
	}
	if !channel.IsValidAnnouncer("ANNOUNCE[BOT]") {
		t.Fatal("ascii CASEMAPPING should still compare ASCII letters case-insensitively")
	}
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

		if strings.Contains(logs.String(), "channel not found") {
			t.Fatalf("case-variant DM target was treated as a channel: %s", logs.String())
		}
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

		if got := len(channel.Messages.GetMessages()); got != 1 {
			t.Fatalf("mixed-case channel message count = %d, want 1", got)
		}
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

	if got := h.CurrentNick(); got != "Nick_" {
		t.Fatalf("current nick = %q, want server-observed new nick", got)
	}
}

func TestOtherUsersNickChangeDoesNotUpdateShadow(t *testing.T) {
	h, _ := newTestHandler()
	h.setCurrentNick("autobrr")

	h.onNick(ircmsg.Message{
		Source:  "Someone!user@host",
		Command: "NICK",
		Params:  []string{"Else"},
	})

	if got := h.CurrentNick(); got != "autobrr" {
		t.Fatalf("other user's NICK changed current nick to %q", got)
	}
}

func TestWelcomeTracksServerSelectedNick(t *testing.T) {
	h, _ := newTestHandler()
	h.handleWelcome(ircmsg.Message{Command: "001", Params: []string{"autobrr_"}})

	if got := h.CurrentNick(); got != "autobrr_" {
		t.Fatalf("current nick = %q, want %q", got, "autobrr_")
	}
}
