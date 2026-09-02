// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/events"

	"github.com/ergochat/irc-go/ircevent"
	"github.com/ergochat/irc-go/ircmsg"
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
		if recordSessionEnd(h, time.Second, started.Add(time.Duration(i)*time.Minute)) {
			t.Fatalf("breaker tripped after %d short sessions", i+1)
		}
	}

	if !recordSessionEnd(h, time.Second, started.Add(4*time.Minute)) {
		t.Fatal("breaker did not trip on the fifth short session")
	}

	h.m.RLock()
	defer h.m.RUnlock()
	if len(h.shortSessionEnds) != 0 {
		t.Fatalf("breaker did not reset after tripping: %v", h.shortSessionEnds)
	}
}

func TestFlappingBreakerResetsAfterHealthySession(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, started.Add(time.Duration(i)*time.Minute))
	}

	if recordSessionEnd(h, flappingSessionMinLifetime, started.Add(4*time.Minute)) {
		t.Fatal("healthy session tripped the breaker")
	}

	for i := range flappingStopThreshold - 1 {
		if recordSessionEnd(h, time.Second, started.Add(time.Duration(i+5)*time.Minute)) {
			t.Fatalf("pre-healthy short sessions leaked into the new streak at session %d", i+1)
		}
	}
}

func TestFlappingBreakerExpiresOldStreak(t *testing.T) {
	h, _ := newTestHandler()
	started := time.Date(2026, time.August, 13, 12, 0, 0, 0, time.UTC)

	for i := range flappingStopThreshold - 1 {
		recordSessionEnd(h, time.Second, started.Add(time.Duration(i)*time.Minute))
	}

	windowStart := started.Add(flappingWindow)
	if recordSessionEnd(h, time.Second, windowStart) {
		t.Fatal("expired short sessions tripped the breaker")
	}

	for i := 1; i < flappingStopThreshold; i++ {
		tripped := recordSessionEnd(h, time.Second, windowStart.Add(time.Duration(i)*time.Minute))
		if tripped != (i == flappingStopThreshold-1) {
			t.Fatalf("new window session %d: tripped=%t", i+1, tripped)
		}
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
		if recordSessionEnd(h, time.Second, endedAt) {
			t.Fatalf("breaker tripped at session %d before five sessions fit in the rolling window", i+1)
		}
	}

	if !recordSessionEnd(h, time.Second, started.Add(flappingWindow+2*time.Second)) {
		t.Fatal("breaker did not trip when the newest five sessions fit in the rolling window")
	}
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
	if len(h.shortSessionEnds) != 0 {
		t.Fatalf("manual stop left a flapping streak: %v", h.shortSessionEnds)
	}
	for _, err := range h.connectionErrors {
		if strings.Contains(err, "connection flapping") {
			t.Fatalf("manual disconnect tripped the breaker: %q", err)
		}
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

	if !h.Stopped() {
		t.Fatal("network remained running after the flapping threshold")
	}
	if !hasConnectError(h, "connection flapping") {
		t.Fatalf("flapping reason was not surfaced: %v", h.connectionErrors)
	}
	if got := h.stateMachine.GetState(); got != StateError {
		t.Fatalf("state=%s, want Error", got)
	}

	emitted := bus.snapshot()
	if len(emitted) != 1 || emitted[0].Type != events.IRCFlapping {
		t.Fatalf("emitted=%v, want one IRC flapping event", emitted)
	}
	if !strings.Contains(emitted[0].Message, "TestNet stopped") {
		t.Fatalf("emitted payload=%v", emitted[0])
	}
	if emitted[0].Network != "TestNet" {
		t.Fatalf("network=%q, want TestNet", emitted[0].Network)
	}
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
	if h.client != replacement || h.clientState != ircLive {
		t.Fatal("stale disconnect stopped the replacement client")
	}
	if len(h.shortSessionEnds) != flappingStopThreshold-1 {
		t.Fatalf("stale disconnect changed breaker strikes: %v", h.shortSessionEnds)
	}
}
