// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"errors"
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

func recordNickLoss(h *Handler, at time.Time) bool {
	h.m.Lock()
	defer h.m.Unlock()

	return h.recordNickLossLocked(at)
}

func TestNickLossBreakerUsesRollingWindow(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.November, 7, 4, 0, 0, 0, time.UTC)

	require.False(t, recordNickLoss(h, started))
	require.False(t, recordNickLoss(h, started.Add(5*time.Minute)))
	require.False(t, recordNickLoss(h, started.Add(nickLossWindow)), "a loss outside the window counted toward the breaker")
	require.True(t, recordNickLoss(h, started.Add(nickLossWindow+time.Minute)), "breaker did not trip on three losses within the window")
}

func TestStopResetsNickLossBreaker(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.November, 7, 4, 0, 0, 0, time.UTC)

	recordNickLoss(h, started)
	recordNickLoss(h, started.Add(time.Minute))

	h.Stop()

	require.False(t, recordNickLoss(h, started.Add(2*time.Minute)), "losses from before a stop counted toward the breaker")
}

func TestFlappingBreakerHalfOpenRecovery(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.runFn = func() error { return nil }
	h.stateMachine.currentState = StateFullyOperational

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, initial.Add(time.Duration(i)*time.Second))
	}
	h.m.Lock()
	h.connectedSince = initial.Add(4 * time.Second)
	h.m.Unlock()
	h.onDisconnect(ircmsg.Message{})

	require.True(t, h.Stopped(), "network should be stopped after flapping")
	h.m.RLock()
	require.Equal(t, flappingInitialCooldown, h.flappingCooldown)
	require.Equal(t, initial.Add(flappingInitialCooldown), h.flappingRetryAt)
	h.m.RUnlock()

	clock.Advance(flappingInitialCooldown + 10*time.Second)

	h.m.RLock()
	require.True(t, h.halfOpenAttempt, "halfOpenAttempt should be true after cooldown expires")
	h.m.RUnlock()

	h.m.Lock()
	h.clientState = ircLive
	client := &ircevent.Connection{}
	h.client = client
	h.connectedSince = clock.Now()
	h.m.Unlock()
	h.onConnect(client, ircmsg.Message{})

	clock.Advance(flappingSessionMinLifetime)

	h.m.RLock()
	defer h.m.RUnlock()
	require.False(t, h.halfOpenAttempt, "halfOpenAttempt should be cleared after surviving")
	require.Equal(t, flappingInitialCooldown, h.flappingCooldown, "cooldown should be reset to initial")
	require.True(t, h.flappingRetryAt.IsZero(), "retryAt should be zero")
	require.Empty(t, h.shortSessionEnds, "breaker strikes should be empty")
}

func TestFlappingBreakerBackoffOnRepeatedFailure(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.runFn = func() error { return nil }
	h.stateMachine.currentState = StateFullyOperational

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, initial.Add(time.Duration(i)*time.Second))
	}
	h.m.Lock()
	h.connectedSince = initial.Add(4 * time.Second)
	h.m.Unlock()
	h.onDisconnect(ircmsg.Message{})

	h.m.RLock()
	require.Equal(t, flappingInitialCooldown, h.flappingCooldown)
	h.m.RUnlock()

	clock.Advance(flappingInitialCooldown + time.Second)
	h.m.RLock()
	require.True(t, h.halfOpenAttempt)
	h.m.RUnlock()

	client1 := &ircevent.Connection{}
	h.m.Lock()
	h.clientState = ircLive
	h.client = client1
	h.connectedSince = clock.Now()
	h.m.Unlock()
	clock.Advance(10 * time.Second)
	h.onClientDisconnect(client1, ircmsg.Message{})

	h.m.RLock()
	require.Equal(t, 30*time.Minute, h.flappingCooldown)
	h.m.RUnlock()

	clock.Advance(30*time.Minute + time.Second)
	h.m.RLock()
	require.True(t, h.halfOpenAttempt)
	h.m.RUnlock()

	client2 := &ircevent.Connection{}
	h.m.Lock()
	h.clientState = ircLive
	h.client = client2
	h.connectedSince = clock.Now()
	h.m.Unlock()
	clock.Advance(10 * time.Second)
	h.onClientDisconnect(client2, ircmsg.Message{})

	h.m.RLock()
	require.Equal(t, flappingMaxCooldown, h.flappingCooldown)
	h.m.RUnlock()
}

func TestFlappingBreakerStopCancelsTimer(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.runFn = func() error { return nil }
	h.stateMachine.currentState = StateFullyOperational

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, initial.Add(time.Duration(i)*time.Second))
	}
	h.m.Lock()
	h.connectedSince = initial.Add(4 * time.Second)
	h.m.Unlock()
	h.onDisconnect(ircmsg.Message{})

	h.m.RLock()
	require.NotNil(t, h.flappingRetryTimer)
	h.m.RUnlock()

	h.Stop()

	h.m.RLock()
	require.Nil(t, h.flappingRetryTimer, "retry timer should be cleared on Stop()")
	require.False(t, h.halfOpenAttempt, "halfOpenAttempt should be false")
	require.Equal(t, flappingInitialCooldown, h.flappingCooldown, "cooldown should reset to initial")
	h.m.RUnlock()

	clock.Advance(2 * time.Hour)
	h.m.RLock()
	require.False(t, h.halfOpenAttempt)
	h.m.RUnlock()
}

func TestFlappingBreakerHalfOpenRunFailureBacksOff(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.stateMachine.currentState = StateFullyOperational

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, initial.Add(time.Duration(i)*time.Second))
	}
	h.m.Lock()
	h.connectedSince = initial.Add(4 * time.Second)
	h.m.Unlock()
	h.onDisconnect(ircmsg.Message{})

	h.m.RLock()
	require.Equal(t, flappingInitialCooldown, h.flappingCooldown)
	h.m.RUnlock()

	done := make(chan struct{})
	h.runFn = func() error {
		defer close(done)
		return errors.New("dial tcp: connection refused")
	}

	clock.Advance(flappingInitialCooldown + time.Second)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for runFn")
	}

	require.Eventually(t, func() bool {
		h.m.RLock()
		defer h.m.RUnlock()
		return !h.halfOpenAttempt && h.flappingRetryTimer != nil
	}, time.Second, 10*time.Millisecond)

	h.m.RLock()
	defer h.m.RUnlock()
	require.Equal(t, 30*time.Minute, h.flappingCooldown, "cooldown should double after failed run")
}

func TestFlappingBreakerHalfOpenRunFailurePreservesManualStop(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.stateMachine.currentState = StateFullyOperational

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, initial.Add(time.Duration(i)*time.Second))
	}
	h.m.Lock()
	h.connectedSince = initial.Add(4 * time.Second)
	h.m.Unlock()
	h.onDisconnect(ircmsg.Message{})

	started := make(chan struct{})
	unblock := make(chan struct{})
	done := make(chan struct{})
	h.halfOpenDone = func() { close(done) }
	h.runFn = func() error {
		close(started)
		<-unblock
		return errors.New("dial failed")
	}

	clock.Advance(flappingInitialCooldown + time.Second)

	<-started
	h.Stop()
	close(unblock)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for half-open reconnect completion")
	}

	h.m.RLock()
	defer h.m.RUnlock()
	require.Nil(t, h.flappingRetryTimer, "flapping retry timer must NOT be re-armed after manual Stop")
	require.False(t, h.halfOpenAttempt)
}

func TestFlappingBreakerStaleTimerCallbackIgnoredAfterStop(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.stateMachine.currentState = StateFullyOperational

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, initial.Add(time.Duration(i)*time.Second))
	}
	h.m.Lock()
	h.connectedSince = initial.Add(4 * time.Second)
	h.m.Unlock()
	h.onDisconnect(ircmsg.Message{})

	h.m.RLock()
	staleGen := h.flappingRetryGen
	h.m.RUnlock()

	h.Stop()

	h.tryHalfOpenReconnect(staleGen)

	h.m.RLock()
	defer h.m.RUnlock()
	require.False(t, h.halfOpenAttempt, "stale timer callback must not trigger half-open attempt")
	require.Equal(t, ircStopped, h.clientState)
}

func TestFlappingBreakerPreferredNickReconnectCarriesOver(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.stateMachine.currentState = StateFullyOperational

	client := &ircevent.Connection{}
	reconnected := false
	h.reconnectFn = func(c *ircevent.Connection) {
		reconnected = true
	}

	h.m.Lock()
	h.clientState = ircLive
	h.client = client
	h.halfOpenAttempt = true
	h.flappingCooldown = flappingInitialCooldown
	h.connectedSince = initial
	h.network.Nick = "autobrr"
	h.currentNick = "autobrr_"
	if h.refusedOnFallback == nil {
		h.refusedOnFallback = make(map[string]bool)
	}
	h.refusedOnFallback["#channel"] = true
	h.m.Unlock()

	ch := &Channel{Name: "#channel", Enabled: true}
	h.channels.Set("#channel", ch)

	// Regain nick -> triggers intentional reconnect
	msg := ircmsg.MakeMessage(nil, "autobrr_", "NICK", "autobrr")
	h.onNick(client, msg)

	require.True(t, reconnected, "client.Reconnect should be triggered")
	h.m.RLock()
	require.True(t, h.handlerRequestedReconnect, "handlerRequestedReconnect must be set")
	h.m.RUnlock()

	// Short session disconnect due to intentional reconnect (e.g. at 5s)
	clock.Advance(5 * time.Second)
	h.onClientDisconnect(client, ircmsg.Message{})

	h.m.RLock()
	require.False(t, h.handlerRequestedReconnect, "handlerRequestedReconnect must be cleared")
	require.True(t, h.halfOpenAttempt, "halfOpenAttempt must carry over")
	require.Equal(t, flappingInitialCooldown, h.flappingCooldown, "cooldown must not escalate on intentional reconnect")
	require.Empty(t, h.shortSessionEnds, "intentional reconnect must not record a short session strike")
	h.m.RUnlock()

	// Reconnect comes up
	h.onConnect(client, ircmsg.Message{})

	// Advance 30s on the new connection -> probe survives
	clock.Advance(flappingSessionMinLifetime)

	h.m.RLock()
	defer h.m.RUnlock()
	require.False(t, h.halfOpenAttempt, "halfOpenAttempt must be false after surviving")
	require.Nil(t, h.flappingSurvivalTimer, "survival timer must be cleared after healing")
}

func TestFlappingBreakerPreferredNickReconnectSecondDropPenalized(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.stateMachine.currentState = StateFullyOperational

	client := &ircevent.Connection{}
	h.reconnectFn = func(c *ircevent.Connection) {}

	h.m.Lock()
	h.clientState = ircLive
	h.client = client
	h.halfOpenAttempt = true
	h.flappingCooldown = flappingInitialCooldown
	h.connectedSince = initial
	h.network.Nick = "autobrr"
	h.currentNick = "autobrr_"
	if h.refusedOnFallback == nil {
		h.refusedOnFallback = make(map[string]bool)
	}
	h.refusedOnFallback["#channel"] = true
	h.m.Unlock()

	ch := &Channel{Name: "#channel", Enabled: true}
	h.channels.Set("#channel", ch)

	// Regain nick -> triggers intentional reconnect
	msg := ircmsg.MakeMessage(nil, "autobrr_", "NICK", "autobrr")
	h.onNick(client, msg)

	// Disconnect 1: intentional reconnect
	clock.Advance(5 * time.Second)
	h.onClientDisconnect(client, ircmsg.Message{})

	// Reconnect comes up
	h.onConnect(client, ircmsg.Message{})

	// Disconnect 2: drops unexpectedly after 5s
	clock.Advance(5 * time.Second)
	h.onClientDisconnect(client, ircmsg.Message{})

	h.m.RLock()
	defer h.m.RUnlock()
	require.False(t, h.halfOpenAttempt, "halfOpenAttempt must be cleared after drop")
	require.Equal(t, flappingInitialCooldown*2, h.flappingCooldown, "cooldown must double after second drop")
	require.Equal(t, ircStopped, h.clientState)
}

func TestFlappingBreakerInterveningClaimPreventsProbeErrorClobber(t *testing.T) {
	h, _ := newTestHandler()
	h.m.Lock()
	h.flappingCooldown = flappingInitialCooldown
	h.clientState = ircStopped
	gen := h.flappingRetryGen
	h.m.Unlock()

	done := make(chan struct{})
	h.halfOpenDone = func() { close(done) }
	h.runFn = func() error {
		// Simulate an intervening Run() or startHandler claiming a new connection
		h.m.Lock()
		h.connectGen++
		h.clientState = ircConnecting
		h.m.Unlock()
		return errors.New("probe dial failure")
	}

	h.tryHalfOpenReconnect(gen)

	<-done

	h.m.RLock()
	defer h.m.RUnlock()
	// Should not have touched clientState or scheduled cooldown
	require.Equal(t, ircConnecting, h.clientState, "newer connection state must not be clobbered by stale probe")
}

func TestFlappingBreakerPreferredNickReconnectDialFailure(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.stateMachine.currentState = StateFullyOperational

	client := &ircevent.Connection{}
	h.reconnectFn = func(c *ircevent.Connection) {}

	h.m.Lock()
	h.clientState = ircLive
	h.client = client
	h.halfOpenAttempt = true
	h.flappingCooldown = flappingInitialCooldown
	h.connectedSince = initial
	h.network.Nick = "autobrr"
	h.currentNick = "autobrr_"
	if h.refusedOnFallback == nil {
		h.refusedOnFallback = make(map[string]bool)
	}
	h.refusedOnFallback["#channel"] = true
	h.m.Unlock()

	ch := &Channel{Name: "#channel", Enabled: true}
	h.channels.Set("#channel", ch)

	// Regain nick -> triggers intentional reconnect
	msg := ircmsg.MakeMessage(nil, "autobrr_", "NICK", "autobrr")
	h.onNick(client, msg)

	// Intentional reconnect disconnect at 5s
	clock.Advance(5 * time.Second)
	h.onClientDisconnect(client, ircmsg.Message{})

	h.m.RLock()
	require.True(t, h.halfOpenAttempt, "halfOpenAttempt must carry over")
	require.True(t, h.connectedSince.IsZero(), "connectedSince must be reset")
	require.NotNil(t, h.flappingReconnectTimer, "reconnect watchdog must be armed")
	h.m.RUnlock()

	// Replacement dial fails
	h.handleReplacementDialError(client, errors.New("connection refused"))

	h.m.RLock()
	require.False(t, h.halfOpenAttempt, "halfOpenAttempt must be false after dial failure")
	require.Equal(t, flappingInitialCooldown*2, h.flappingCooldown, "cooldown must double after replacement dial failure")
	require.Equal(t, ircStopped, h.clientState)
	require.Nil(t, h.flappingReconnectTimer, "reconnect timer must be cancelled")
	require.NotNil(t, h.flappingRetryTimer, "retry timer must be scheduled")
	h.m.RUnlock()
}

func TestFlappingBreakerPreferredNickReconnectTimeout(t *testing.T) {
	initial := time.Date(2026, time.October, 19, 12, 0, 0, 0, time.UTC)
	clock := newMockClock(initial)

	h, _ := newTestHandler()
	h.clock = clock
	h.stateMachine.currentState = StateFullyOperational

	client := &ircevent.Connection{}
	h.reconnectFn = func(c *ircevent.Connection) {}

	h.m.Lock()
	h.clientState = ircLive
	h.client = client
	h.halfOpenAttempt = true
	h.flappingCooldown = flappingInitialCooldown
	h.connectedSince = initial
	h.network.Nick = "autobrr"
	h.currentNick = "autobrr_"
	if h.refusedOnFallback == nil {
		h.refusedOnFallback = make(map[string]bool)
	}
	h.refusedOnFallback["#channel"] = true
	h.m.Unlock()

	ch := &Channel{Name: "#channel", Enabled: true}
	h.channels.Set("#channel", ch)

	// Regain nick -> triggers intentional reconnect
	msg := ircmsg.MakeMessage(nil, "autobrr_", "NICK", "autobrr")
	h.onNick(client, msg)

	// Intentional reconnect disconnect at 5s
	clock.Advance(5 * time.Second)
	h.onClientDisconnect(client, ircmsg.Message{})

	h.m.RLock()
	require.True(t, h.halfOpenAttempt, "halfOpenAttempt must carry over")
	require.NotNil(t, h.flappingReconnectTimer, "reconnect watchdog must be armed")
	h.m.RUnlock()

	// Advance clock past flappingReconnectTimeout without onConnect
	clock.Advance(flappingReconnectTimeout)

	h.m.RLock()
	require.False(t, h.halfOpenAttempt, "halfOpenAttempt must be cleared after timeout")
	require.Equal(t, flappingInitialCooldown*2, h.flappingCooldown, "cooldown must double after replacement timeout")
	require.Equal(t, ircStopped, h.clientState)
	require.Nil(t, h.flappingReconnectTimer, "reconnect timer must be cleared")
	require.NotNil(t, h.flappingRetryTimer, "retry timer must be scheduled")
	h.m.RUnlock()
}
