// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

// TestComputeHealthy verifies network health is driven only by the announce
// (default) channels: a failing user-added extra channel must not flip the whole
// network unhealthy, but a failing announce channel must.
func TestComputeHealthy(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	announce := NewChannel(zerolog.Nop(), h.network.ID, "#announce", true, false, nil) // default
	announce.SetMonitoring()
	h.channels.Set(announce.Name, announce)

	extra := NewChannel(zerolog.Nop(), h.network.ID, "#extra", false, false, nil) // user-added
	extra.SetConnectionError("could not join #extra: wrong or missing channel password (+k)")
	h.channels.Set(extra.Name, extra)

	require.True(t, h.computeHealthy(), "a failing non-default channel must not make the network unhealthy")

	announce.SetConnectionError("boom")
	require.False(t, h.computeHealthy(), "a failing default (announce) channel must make the network unhealthy")
}

// TestComputeHealthy_ConnectionUnhealthy verifies the connection state machine
// still gates health even when the announce channels look fine.
func TestComputeHealthy_ConnectionUnhealthy(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateError

	announce := NewChannel(zerolog.Nop(), h.network.ID, "#announce", true, false, nil)
	announce.SetMonitoring()
	h.channels.Set(announce.Name, announce)

	require.False(t, h.computeHealthy(), "network must be unhealthy when the connection state machine is unhealthy")
}

// TestStateEventCarriesHealth is the end-to-end guard for the user's scenario:
// enabling an extra (non-default) channel that fails to join must NOT flip the
// network unhealthy, and the STATE event must carry that (healthy=true).
func TestStateEventCarriesHealth(t *testing.T) {
	h, sse := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	announce := NewChannel(zerolog.Nop(), h.network.ID, "#announce", true, false, nil)
	announce.SetMonitoring()
	h.channels.Set(announce.Name, announce)

	extra := NewChannel(zerolog.Nop(), h.network.ID, "#extra", false, false, nil)
	sm := NewChannelStateMachine(extra, h, "")
	extra.SetStateMachine(sm)
	sm.state = ChannelStateJoining
	h.channels.Set(extra.Name, extra)

	h.handleJoinError(ircmsg.MakeMessage(nil, "srv", ircevent.ERR_BADCHANNELKEY, "bot", "#extra", "Cannot join channel (+k)"))

	require.Truef(t, waitForState(sm, ChannelStateError, time.Second), "expected #extra Error, got %s", sm.CurrentState())

	healthy, found := stateEventHealthy(sse, "#extra", "Error")
	require.True(t, found, "STATE event should carry a healthy field")
	require.True(t, healthy, "a non-default channel failure should keep the network healthy in the STATE event")
}
