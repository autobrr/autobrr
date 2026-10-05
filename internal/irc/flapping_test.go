// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/events"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/stretchr/testify/require"
)

type recordingEventBus struct {
	m      sync.Mutex
	events []events.IRCEvent
}

func (b *recordingEventBus) EmitIRC(_ context.Context, event events.IRCEvent) {
	b.m.Lock()
	defer b.m.Unlock()

	b.events = append(b.events, event)
}

func (b *recordingEventBus) OnProxy(_ func(context.Context, events.ProxyChangeEvent) error) func() {
	return func() {}
}

func (b *recordingEventBus) snapshot() []events.IRCEvent {
	b.m.Lock()
	defer b.m.Unlock()

	return append([]events.IRCEvent(nil), b.events...)
}

func recordSessionEnd(h *Handler, lifetime time.Duration, endedAt time.Time) bool {
	h.m.Lock()
	defer h.m.Unlock()

	return h.recordSessionEndLocked(lifetime, endedAt)
}

func TestFlappingBreakerTripsOnFifthShortSession(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)

	for i := range flappingStopThreshold - 1 {
		require.Falsef(t, recordSessionEnd(h, time.Second, started.Add(time.Duration(i)*time.Minute)), "breaker tripped after %d short sessions", i+1)
	}

	require.True(t, recordSessionEnd(h, time.Second, started.Add(4*time.Minute)), "breaker did not trip on the fifth short session")

	h.m.RLock()
	defer h.m.RUnlock()
	require.Empty(t, h.shortSessionEnds, "breaker did not reset after tripping")
}

func TestFlappingBreakerResetsAfterHealthySession(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, started.Add(time.Duration(i)*time.Minute))
	}

	require.False(t, recordSessionEnd(h, flappingSessionMinLifetime, started.Add(4*time.Minute)), "healthy session tripped the breaker")

	for i := range flappingStopThreshold - 1 {
		require.Falsef(t, recordSessionEnd(h, time.Second, started.Add(time.Duration(i+5)*time.Minute)), "pre-healthy short sessions leaked into the new streak at session %d", i+1)
	}
}

func TestFlappingBreakerExpiresOldStreak(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, started.Add(time.Duration(i)*time.Minute))
	}

	windowStart := started.Add(flappingWindow)
	require.False(t, recordSessionEnd(h, time.Second, windowStart), "expired short sessions tripped the breaker")

	for i := 1; i < flappingStopThreshold; i++ {
		tripped := recordSessionEnd(h, time.Second, windowStart.Add(time.Duration(i)*time.Minute))
		require.Equalf(t, i == flappingStopThreshold-1, tripped, "new window session %d", i+1)
	}
}

func TestFlappingBreakerUsesRollingWindow(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)

	ends := []time.Time{
		started,
		started.Add(14 * time.Minute),
		started.Add(14*time.Minute + 10*time.Second),
		started.Add(14*time.Minute + 20*time.Second),
		started.Add(flappingWindow + time.Second),
	}
	for i, endedAt := range ends {
		require.Falsef(t, recordSessionEnd(h, time.Second, endedAt), "breaker tripped at session %d before five sessions fit in the rolling window", i+1)
	}

	require.True(t, recordSessionEnd(h, time.Second, started.Add(flappingWindow+2*time.Second)), "breaker did not trip when the newest five sessions fit in the rolling window")
}

func TestManualStopResetsAndDoesNotFeedFlappingBreaker(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, started.Add(time.Duration(i)*time.Minute))
	}

	h.Stop()

	h.m.Lock()
	h.connectedSince = time.Now().Add(-time.Second)
	h.m.Unlock()
	h.onDisconnect(ircmsg.Message{})

	h.m.RLock()
	defer h.m.RUnlock()
	require.Empty(t, h.shortSessionEnds, "manual stop left a flapping streak")
	for _, err := range h.connectionErrors {
		require.NotContains(t, err, "connection flapping", "manual disconnect tripped the breaker")
	}
}

func TestFlappingBreakerStopsNetworkAndEmitsEvent(t *testing.T) {
	h, _ := newTestHandler()
	bus := &recordingEventBus{}
	h.eventBus = bus
	h.stateMachine.currentState = StateFullyOperational

	now := time.Now()
	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, now.Add(time.Duration(i-flappingStopThreshold)*time.Second))
	}

	h.m.Lock()
	h.connectedSince = time.Now().Add(-time.Second)
	h.m.Unlock()

	h.onDisconnect(ircmsg.Message{})

	require.True(t, h.Stopped(), "network remained running after the flapping threshold")
	require.Truef(t, hasConnectError(h, "connection flapping"), "flapping reason was not surfaced: %v", h.connectionErrors)
	require.Equal(t, StateError, h.stateMachine.GetState())

	emitted := bus.snapshot()
	require.Len(t, emitted, 1)
	require.Equal(t, events.IRCFlapping, emitted[0].Type)
	require.Contains(t, emitted[0].Message, "TestNet stopped")
	require.Equal(t, "TestNet", emitted[0].Network)
}

func TestStaleDisconnectCannotTripBreakerOnReplacement(t *testing.T) {
	h, _ := newTestHandler()
	oldClient := &ircevent.Connection{}
	replacement := &ircevent.Connection{}
	h.client = replacement

	now := time.Now()
	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, now.Add(time.Duration(i-flappingStopThreshold)*time.Second))
	}

	h.onClientDisconnect(oldClient, ircmsg.Message{})

	h.m.RLock()
	defer h.m.RUnlock()
	require.Same(t, replacement, h.client, "stale disconnect stopped the replacement client")
	require.Equal(t, ircLive, h.clientState, "stale disconnect stopped the replacement client")
	require.Len(t, h.shortSessionEnds, flappingStopThreshold-1, "stale disconnect changed breaker strikes")
}
