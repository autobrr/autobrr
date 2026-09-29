// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdateChannelRejoinsOnPasswordChangeWhenNotMonitoring covers the +k fix:
// correcting a channel password on a channel that failed to join re-joins it on
// the fly (no network restart) with the new key.
func TestUpdateChannelRejoinsOnPasswordChangeWhenNotMonitoring(t *testing.T) {
	h, sse := newTestHandler()
	newIdleSM(h, "#chan", "") // enabled, Idle, no password
	ch, _ := h.channels.Get("#chan")

	h.UpdateChannel(domain.IrcChannel{Name: "#chan", Enabled: true, Password: "newkey"})

	assert.Equal(t, "newkey", ch.GetPassword())
	require.True(t, waitFor(func() bool { return sse.hasStateEvent("#chan", "Joining") }, time.Second), "password change on a non-monitoring channel should trigger a (re)join")
}

// TestUpdateChannelEnableJoins verifies enabling a disabled channel joins it.
func TestUpdateChannelEnableJoins(t *testing.T) {
	h, sse := newTestHandler()
	newIdleSM(h, "#chan", "")
	ch, _ := h.channels.Get("#chan")
	ch.Configure(0, false, "") // start disabled

	h.UpdateChannel(domain.IrcChannel{Name: "#chan", Enabled: true})

	assert.True(t, ch.IsEnabled(), "channel should be enabled")
	require.True(t, waitFor(func() bool { return sse.hasStateEvent("#chan", "Joining") }, time.Second), "enabling a channel should trigger a join")
}

// TestUpdateChannelDisableParts verifies disabling a monitored channel parts it.
func TestUpdateChannelDisableParts(t *testing.T) {
	h, _ := newTestHandler()
	sm := addMonitoredChannel(h, "#chan", "") // Monitoring, enabled
	ch, _ := h.channels.Get("#chan")

	h.UpdateChannel(domain.IrcChannel{Name: "#chan", Enabled: false})

	assert.False(t, ch.IsEnabled(), "channel should be disabled")
	assert.False(t, ch.IsMonitoring(), "disabled channel should no longer be monitoring")
	require.Truef(t, waitForState(sm, ChannelStateIdle, time.Second), "disabled channel should reset to Idle, got %s", sm.CurrentState())
}

// TestUpdateChannelPasswordWhileMonitoringDoesNotDisrupt verifies a password
// change on an actively-monitored channel stores the new key for the next
// reconnect but does NOT part/rejoin (which risks losing a working channel).
func TestUpdateChannelPasswordWhileMonitoringDoesNotDisrupt(t *testing.T) {
	h, sse := newTestHandler()
	sm := addMonitoredChannel(h, "#chan", "") // Monitoring
	ch, _ := h.channels.Get("#chan")

	h.UpdateChannel(domain.IrcChannel{Name: "#chan", Enabled: true, Password: "newkey"})

	assert.Equal(t, "newkey", ch.GetPassword(), "new password should be stored")
	time.Sleep(50 * time.Millisecond)
	require.Equal(t, ChannelStateMonitoring, sm.CurrentState(), "monitoring channel should stay Monitoring")
	require.False(t, sse.hasStateEvent("#chan", "Joining") || sse.hasStateEvent("#chan", "Idle"), "a password change while monitoring must not part/rejoin the channel")
}

// TestUpdateChannelNoChangeNoOp verifies an update with no config change is inert.
func TestUpdateChannelNoChangeNoOp(t *testing.T) {
	h, sse := newTestHandler()
	sm := addMonitoredChannel(h, "#chan", "")
	ch, _ := h.channels.Get("#chan")
	ch.Configure(1, true, "key")

	h.UpdateChannel(domain.IrcChannel{ID: 1, Name: "#chan", Enabled: true, Password: "key"})

	time.Sleep(50 * time.Millisecond)
	require.Equal(t, ChannelStateMonitoring, sm.CurrentState(), "unchanged channel should stay Monitoring")
	require.False(t, sse.hasStateEvent("#chan", "Joining") || sse.hasStateEvent("#chan", "Idle"), "a no-op update must not disrupt the channel")
}

// TestAddChannelRegistersBeforeJoin is a regression test for the add-channel
// JOIN-then-PART bug: a channel added to a running network must be registered in
// h.channels before the JOIN is sent, otherwise the JOIN echo is parted as an
// unwanted channel and the channel is never monitored until a restart.
func TestAddChannelRegistersBeforeJoin(t *testing.T) {
	h, _ := newTestHandler()

	h.AddChannel(domain.IrcChannel{Name: "#New-Announce", Enabled: true, Password: "secret"})

	got, found := h.channels.Get("#new-announce")
	require.True(t, found, "AddChannel did not register the channel — its JOIN echo would be parted as unwanted")
	assert.True(t, got.Enabled, "added channel should be enabled")
	assert.Equal(t, "secret", got.Password)
	assert.NotNil(t, got.StateMachine(), "added channel should have a state machine")
}

// TestAddChannelDisabledDoesNotStart verifies a disabled channel is registered
// but not joined.
func TestAddChannelDisabledDoesNotStart(t *testing.T) {
	h, _ := newTestHandler()

	h.AddChannel(domain.IrcChannel{Name: "#chan", Enabled: false})

	got, found := h.channels.Get("#chan")
	require.True(t, found, "disabled channel should still be registered")
	assert.False(t, got.Enabled, "channel should be disabled")
	if sm := got.StateMachine(); sm != nil {
		assert.Equal(t, ChannelStateIdle, sm.CurrentState(), "disabled channel should stay Idle")
	}
}

// TestRemoveChannelUnregisters verifies a removed channel is parted and dropped
// from tracking so it stops counting toward network health.
func TestRemoveChannelUnregisters(t *testing.T) {
	h, _ := newTestHandler()
	addMonitoredChannel(h, "#chan", "")

	_, found := h.channels.Get("#chan")
	require.True(t, found, "precondition: channel should be registered")

	h.RemoveChannel("#Chan") // mixed case on purpose

	_, found = h.channels.Get("#chan")
	require.False(t, found, "RemoveChannel did not unregister the channel; it would keep counting toward health")
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
		Source:  "someuser!u@host",
	}

	h.handleJoin(msg) // must not panic and must leave the channel registered

	_, found := h.channels.Get("#known")
	require.True(t, found, "registered channel was dropped by handleJoin")
}
