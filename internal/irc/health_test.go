// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
)

// TestComputeHealthy verifies network health is driven only by the announce
// (default) channels: a failing user-added extra channel must not flip the whole
// network unhealthy, but a failing announce channel must.
func TestComputeHealthy(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	announce := NewChannel(zerolog.Nop(), h.network.ID, "#announce", true, nil) // default
	announce.SetMonitoring()
	h.channels.Set(announce.Name, announce)

	extra := NewChannel(zerolog.Nop(), h.network.ID, "#extra", false, nil) // user-added
	extra.SetConnectionError("could not join #extra: wrong or missing channel password (+k)")
	h.channels.Set(extra.Name, extra)

	if !h.computeHealthy() {
		t.Fatal("a failing non-default channel must not make the network unhealthy")
	}

	announce.SetConnectionError("boom")
	if h.computeHealthy() {
		t.Fatal("a failing default (announce) channel must make the network unhealthy")
	}
}

// TestComputeHealthy_ConnectionUnhealthy verifies the connection state machine
// still gates health even when the announce channels look fine.
func TestComputeHealthy_ConnectionUnhealthy(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateError

	announce := NewChannel(zerolog.Nop(), h.network.ID, "#announce", true, nil)
	announce.SetMonitoring()
	h.channels.Set(announce.Name, announce)

	if h.computeHealthy() {
		t.Fatal("network must be unhealthy when the connection state machine is unhealthy")
	}
}

// TestStateEventCarriesHealth is the end-to-end guard for the user's scenario:
// enabling an extra (non-default) channel that fails to join must NOT flip the
// network unhealthy, and the STATE event must carry that (healthy=true).
func TestStateEventCarriesHealth(t *testing.T) {
	h, sse := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	announce := NewChannel(zerolog.Nop(), h.network.ID, "#announce", true, nil)
	announce.SetMonitoring()
	h.channels.Set(announce.Name, announce)

	extra := NewChannel(zerolog.Nop(), h.network.ID, "#extra", false, nil)
	sm := NewChannelStateMachine(extra, h, "")
	extra.SetStateMachine(sm)
	sm.state = ChannelStateJoining
	h.channels.Set(extra.Name, extra)

	h.handleJoinError(ircmsg.MakeMessage(nil, "srv", ircevent.ERR_BADCHANNELKEY, "bot", "#extra", "Cannot join channel (+k)"))

	if !waitForState(sm, ChannelStateError, time.Second) {
		t.Fatalf("expected #extra Error, got %s", sm.CurrentState())
	}

	healthy, found := stateEventHealthy(sse, "#extra", "Error")
	if !found {
		t.Fatal("STATE event should carry a healthy field")
	}
	if !healthy {
		t.Fatal("a non-default channel failure should keep the network healthy in the STATE event")
	}
}
