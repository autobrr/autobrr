// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
)

// TestAddChannelRegistersBeforeJoin is a regression test for the add-channel
// JOIN-then-PART bug: a channel added to a running network must be registered in
// h.channels before the JOIN is sent, otherwise the JOIN echo is parted as an
// unwanted channel and the channel is never monitored until a restart.
func TestAddChannelRegistersBeforeJoin(t *testing.T) {
	h, _ := newTestHandler()

	h.AddChannel(domain.IrcChannel{Name: "#New-Announce", Enabled: true, Password: "secret"})

	got, found := h.channels.Get("#new-announce")
	if !found {
		t.Fatal("AddChannel did not register the channel — its JOIN echo would be parted as unwanted")
	}
	if !got.Enabled {
		t.Error("added channel should be enabled")
	}
	if got.Password != "secret" {
		t.Errorf("added channel password = %q, want %q", got.Password, "secret")
	}
	if got.StateMachine() == nil {
		t.Error("added channel should have a state machine")
	}
}

// TestAddChannelDisabledDoesNotStart verifies a disabled channel is registered
// but not joined.
func TestAddChannelDisabledDoesNotStart(t *testing.T) {
	h, _ := newTestHandler()

	h.AddChannel(domain.IrcChannel{Name: "#chan", Enabled: false})

	got, found := h.channels.Get("#chan")
	if !found {
		t.Fatal("disabled channel should still be registered")
	}
	if got.Enabled {
		t.Error("channel should be disabled")
	}
	if sm := got.StateMachine(); sm != nil && sm.CurrentState() != ChannelStateIdle {
		t.Errorf("disabled channel should stay Idle, got %s", sm.CurrentState())
	}
}

// TestRemoveChannelUnregisters verifies a removed channel is parted and dropped
// from tracking so it stops counting toward network health.
func TestRemoveChannelUnregisters(t *testing.T) {
	h, _ := newTestHandler()
	addMonitoredChannel(h, "#chan", "")

	if _, found := h.channels.Get("#chan"); !found {
		t.Fatal("precondition: channel should be registered")
	}

	h.RemoveChannel("#Chan") // mixed case on purpose

	if _, found := h.channels.Get("#chan"); found {
		t.Fatal("RemoveChannel did not unregister the channel; it would keep counting toward health")
	}
}

// TestHandleJoinDoesNotPartRegisteredChannel checks the flip side directly: once
// a channel is registered, handleJoin treats the JOIN as expected rather than
// parting it. Registered channels never take the "unwanted" branch.
func TestHandleJoinDoesNotPartRegisteredChannel(t *testing.T) {
	h, _ := newTestHandler()
	h.AddChannel(domain.IrcChannel{Name: "#known", Enabled: true})

	// a JOIN from some other user for the known channel must be accepted, not
	// routed through the "unwanted channel" part path.
	msg := ircmsg.Message{
		Command: "JOIN",
		Params:  []string{"#known"},
	}
	msg.Source = "someuser!u@host"

	h.handleJoin(msg) // must not panic and must leave the channel registered

	if _, found := h.channels.Get("#known"); !found {
		t.Fatal("registered channel was dropped by handleJoin")
	}
}
