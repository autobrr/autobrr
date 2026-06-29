// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
)

// newTestHandler returns a Handler suitable for unit tests.
func newTestHandler(network domain.IrcNetwork) *Handler {
	return &Handler{
		log:              zerolog.Nop(),
		network:          &network,
		connectionErrors: []string{},
		channelHealth:    map[string]*channelHealth{},
		validChannels:    map[string]struct{}{},
		validAnnouncers:  map[string]struct{}{},
	}
}

// joinErrorMsg builds a JOIN-error numeric message as the server would send it.
func joinErrorMsg(numeric, botnick, channel, reason string) ircmsg.Message {
	if reason == "" {
		return ircmsg.MakeMessage(nil, "server.example.net", numeric, botnick, channel)
	}
	return ircmsg.MakeMessage(nil, "server.example.net", numeric, botnick, channel, reason)
}

// --- handleJoinError ---

func TestHandleJoinErrorTooFewParams(t *testing.T) {
	h := newTestHandler(domain.IrcNetwork{})

	// zero params
	h.handleJoinError(ircmsg.MakeMessage(nil, "server", "475"))
	// one param (botnick only — no channel)
	h.handleJoinError(ircmsg.MakeMessage(nil, "server", "475", "botnick"))

	if len(h.connectionErrors) != 0 {
		t.Errorf("expected no errors for short param lists, got: %v", h.connectionErrors)
	}
}

func TestHandleJoinErrorChannelNoReason(t *testing.T) {
	h := newTestHandler(domain.IrcNetwork{})
	h.handleJoinError(joinErrorMsg("477", "botnick", "#secretchan", ""))

	if len(h.connectionErrors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(h.connectionErrors))
	}
	got := h.connectionErrors[0]
	if !strings.Contains(got, "#secretchan") {
		t.Errorf("error missing channel name: %s", got)
	}
	if !strings.Contains(got, "477") {
		t.Errorf("error missing numeric: %s", got)
	}
}

func TestHandleJoinErrorChannelWithReason(t *testing.T) {
	h := newTestHandler(domain.IrcNetwork{})
	h.handleJoinError(joinErrorMsg("475", "botnick", "#locked", "Bad channel key"))

	if len(h.connectionErrors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(h.connectionErrors))
	}
	got := h.connectionErrors[0]
	if !strings.Contains(got, "#locked") {
		t.Errorf("error missing channel: %s", got)
	}
	if !strings.Contains(got, "Bad channel key") {
		t.Errorf("error missing reason: %s", got)
	}
	if !strings.Contains(got, "475") {
		t.Errorf("error missing numeric: %s", got)
	}
}

// TestHandleJoinErrorAllNumerics ensures that every registered JOIN-error numeric
// produces a connection error entry.
func TestHandleJoinErrorAllNumerics(t *testing.T) {
	numerics := []struct {
		code string
		desc string
	}{
		{"471", "ERR_CHANNELISFULL"},
		{"473", "ERR_INVITEONLYCHAN"},
		{"474", "ERR_BANNEDFROMCHAN"},
		{"475", "ERR_BADCHANNELKEY"},
		{"477", "ERR_NEEDREGGEDNICK"},
	}
	for _, tc := range numerics {
		t.Run(tc.code+"_"+tc.desc, func(t *testing.T) {
			h := newTestHandler(domain.IrcNetwork{})
			h.handleJoinError(joinErrorMsg(tc.code, "bot", "#chan", "reason"))
			if len(h.connectionErrors) != 1 {
				t.Errorf("%s (%s): expected 1 error, got %d", tc.code, tc.desc, len(h.connectionErrors))
			}
		})
	}
}
