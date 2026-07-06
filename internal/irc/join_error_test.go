// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
)

// TestHandleJoinError_SurfacesChannelError verifies a failed JOIN (here a 475
// bad-key) is surfaced on the specific channel immediately, with a clear reason,
// instead of leaving it in Joining until the join timeout fires.
func TestHandleJoinError_SurfacesChannelError(t *testing.T) {
	h, sse := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational
	sm := newIdleSM(h, "#locked", "")
	sm.state = ChannelStateJoining // as if we just sent JOIN

	msg := ircmsg.MakeMessage(nil, "irc.example.test", ircevent.ERR_BADCHANNELKEY, "bot", "#locked", "Cannot join channel (+k)")
	h.handleJoinError(msg)

	if !waitForState(sm, ChannelStateError, time.Second) {
		t.Fatalf("expected channel Error after 475, got %s", sm.CurrentState())
	}

	ch, _ := h.channels.Get("#locked")
	if !ch.HasConnectionErrors() {
		t.Fatal("expected a connection error recorded on the channel")
	}
	if !sse.hasStateEvent("#locked", "Error") {
		t.Error("expected a STATE=Error broadcast")
	}

	// the Error STATE event must carry the reason so the UI can show it in real
	// time (not only on the next health poll)
	if !stateEventHasError(sse, "#locked", "Error", "+k") {
		t.Error("STATE=Error broadcast should include connection_errors with the reason")
	}

	// a channel-scoped JOIN error must NOT leak into the network-level bucket - that
	// bucket is reserved for network-wide failures (NickServ/SASL) and only clears
	// on re-auth, so a per-channel join error there would never clear and would
	// misrepresent an otherwise-healthy network. The reason lives on the channel.
	h.m.RLock()
	errs := slices.Clone(h.connectionErrors)
	h.m.RUnlock()

	for _, e := range errs {
		if strings.Contains(e, "#locked") {
			t.Fatalf("channel join error must not be added to the network-level errors, got %v", errs)
		}
	}
}

// TestHandleJoinError_AllNumerics ensures every registered JOIN-error numeric
// drives the channel into Error.
func TestHandleJoinError_AllNumerics(t *testing.T) {
	numerics := []string{
		ircevent.ERR_CHANNELISFULL,  // 471
		ircevent.ERR_INVITEONLYCHAN, // 473
		ircevent.ERR_BANNEDFROMCHAN, // 474
		ircevent.ERR_BADCHANNELKEY,  // 475
		ircevent.ERR_NEEDREGGEDNICK, // 477
	}
	for _, numeric := range numerics {
		t.Run(numeric, func(t *testing.T) {
			h, _ := newTestHandler()
			h.stateMachine.currentState = StateFullyOperational
			sm := newIdleSM(h, "#chan", "")
			sm.state = ChannelStateJoining

			h.handleJoinError(ircmsg.MakeMessage(nil, "srv", numeric, "bot", "#chan", "reason"))

			if !waitForState(sm, ChannelStateError, time.Second) {
				t.Fatalf("numeric %s: expected channel Error, got %s", numeric, sm.CurrentState())
			}
		})
	}
}

// TestHandleJoinError_TooFewParams verifies malformed/short numerics are ignored
// without panicking or recording an error.
func TestHandleJoinError_TooFewParams(t *testing.T) {
	h, _ := newTestHandler()

	h.handleJoinError(ircmsg.MakeMessage(nil, "srv", ircevent.ERR_BADCHANNELKEY))        // no channel
	h.handleJoinError(ircmsg.MakeMessage(nil, "srv", ircevent.ERR_BADCHANNELKEY, "bot")) // botnick only

	h.m.RLock()
	n := len(h.connectionErrors)
	h.m.RUnlock()
	if n != 0 {
		t.Errorf("expected no errors for short param lists, got %d", n)
	}
}

// TestHandleBannedStopsAndSurfacesReason verifies a 465 (ERR_YOUREBANNEDCREEP,
// e.g. a G-Line) stops the network and surfaces the ban reason at the network
// level (both stored for the poll and broadcast in real time), instead of silently
// reconnecting into a ban loop.
func TestHandleBannedStopsAndSurfacesReason(t *testing.T) {
	h, sse := newTestHandler()

	// a live state from which the connection SM can transition to Error
	h.stateMachine.m.Lock()
	h.stateMachine.currentState = StateConnected
	h.stateMachine.m.Unlock()

	msg := ircmsg.MakeMessage(nil, "irc.orpheus.network", ircevent.ERR_YOUREBANNEDCREEP,
		"indokiwijuice|bot", "You are not welcome on this network. G-Lined: reconnect loop.")
	h.handleBanned(msg)

	// the ban reason is surfaced at the network level
	h.m.RLock()
	errs := slices.Clone(h.connectionErrors)
	h.m.RUnlock()
	found := false
	for _, e := range errs {
		if strings.Contains(e, "banned from network") && strings.Contains(e, "G-Lined") {
			found = true
		}
	}
	if !found {
		t.Fatalf("ban reason should be surfaced in the network errors, got %v", errs)
	}

	// the network is stopped so it does not reconnect into the ban
	if !h.Stopped() {
		t.Fatal("network should be stopped after a 465 ban")
	}

	// and the reason is broadcast in real time (HEALTH event)
	if !waitFor(func() bool { return healthEventHasError(sse, h.network.ID, "banned from network") }, time.Second) {
		t.Fatal("a ban should broadcast a HEALTH event carrying the reason")
	}
}

// TestHandleBannedNoReason verifies a 465 with no trailing reason still stops the
// network and records a generic ban error (no panic on short params).
func TestHandleBannedNoReason(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.m.Lock()
	h.stateMachine.currentState = StateConnected
	h.stateMachine.m.Unlock()

	// no params at all
	h.handleBanned(ircmsg.MakeMessage(nil, "srv", ircevent.ERR_YOUREBANNEDCREEP))

	h.m.RLock()
	errs := slices.Clone(h.connectionErrors)
	h.m.RUnlock()
	if len(errs) != 1 || !strings.Contains(errs[0], "banned from network") {
		t.Fatalf("expected a generic ban error, got %v", errs)
	}
	if !h.Stopped() {
		t.Fatal("network should be stopped after a 465 ban")
	}
}

// TestInitIndexersAppliesChannelPassword is a regression test for the +k channel:
// a user-defined channel's persisted password must be applied on startup so the
// JOIN is sent with the key.
func TestInitIndexersAppliesChannelPassword(t *testing.T) {
	h, _ := newTestHandler()
	h.network.Channels = []domain.IrcChannel{
		{ID: 7, Name: "#Locked", Enabled: true, Password: "sekret"},
	}

	// a definition with no channels of its own still drives the user-defined
	// channel reconciliation in InitIndexers.
	def := &domain.IndexerDefinition{
		Identifier: "test",
		IRC:        &domain.IndexerIRCV2{Network: "irc.example.test"},
	}

	h.InitIndexers([]*domain.IndexerDefinition{def})

	ch, found := h.channels.Get("#locked")
	if !found {
		t.Fatal("user-defined channel #locked was not registered")
	}
	if got := ch.GetPassword(); got != "sekret" {
		t.Errorf("channel password = %q, want %q", got, "sekret")
	}
	if !ch.IsEnabled() {
		t.Error("channel should be enabled")
	}
}
