// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build irc_integration_test

// Package harness wires the real internal/irc Handler up to the in-process test
// ircd: it supplies the collaborators NewHandler needs (an SSE sink, a release
// sink, a no-op notifier), drives the connection lifecycle, and exposes helpers
// to wait on channel state and captured releases.
package harness

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/irc"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

// Instance is a running Handler plus the hooks tests assert against.
type Instance struct {
	Handler  *irc.Handler
	Releases *ReleaseSink

	t   testing.TB
	sse *sseCapture
}

// Options tunes a harness instance.
type Options struct {
	// Verbose routes handler logs to the test log at trace level. Off by default.
	Verbose bool
	// AllowRunError tolerates a non-nil error from Handler.Run instead of failing
	// the test. Use it when the connection is expected to fail fatally (e.g. a ban
	// that aborts registration): the handler still surfaces the reason and stops.
	AllowRunError bool
}

// Start builds a Handler for network/defs, connects it to the (already running)
// test server addressed by network.Server/Port, and registers teardown.
func Start(t testing.TB, network domain.IrcNetwork, defs []*domain.IndexerDefinition, opts ...Options) *Instance {
	t.Helper()

	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}

	level := zerolog.Disabled
	if o.Verbose {
		level = zerolog.TraceLevel
	}
	log := zerolog.New(zerolog.NewTestWriter(t)).Level(level).With().Timestamp().Logger()

	bus := events.NewEventBus(log)

	inst := &Instance{
		t:        t,
		sse:      &sseCapture{},
		Releases: newReleaseSink(),
	}

	inst.Handler = irc.NewHandler(log, bus, inst.sse, network, defs, inst.Releases)

	if err := inst.Handler.Run(); err != nil && !o.AllowRunError {
		t.Fatalf("harness: handler.Run: %v", err)
	}
	t.Cleanup(inst.Handler.Stop)

	return inst
}

// WaitForMonitoring blocks until the given channel reports Monitoring, or fails.
func (i *Instance) WaitForMonitoring(channel string, timeout time.Duration) {
	i.t.Helper()
	i.WaitForState(channel, "Monitoring", timeout)
}

// WaitForState blocks until a STATE event for channel with the wanted state has
// been observed, or fails the test with the states actually seen.
func (i *Instance) WaitForState(channel, state string, timeout time.Duration) {
	i.t.Helper()
	if !waitFor(func() bool { return i.sse.hasState(channel, state) }, timeout) {
		i.t.Fatalf("channel %s never reached state %q; states seen: %v", channel, state, i.sse.statesFor(channel))
	}
}

// LastError returns the most recent connection_errors entry broadcast for a
// channel (empty if none), useful for asserting a specific failure reason.
func (i *Instance) LastError(channel string) string {
	return i.sse.lastError(channel)
}

// WaitForHealthy blocks until a STATE event reports the network healthy (the
// network-level ConnectionStateMachine reached an operational state and all
// announce channels are monitoring), or fails. This asserts the whole-network
// outcome, not just a single channel's state.
func (i *Instance) WaitForHealthy(timeout time.Duration) {
	i.t.Helper()
	if !waitFor(i.sse.sawHealthy, timeout) {
		i.t.Fatal("network never reported healthy")
	}
}

// WaitForNetworkError blocks until a network-level HEALTH event carries a
// connection error containing substr, or fails. Use it to assert a network-wide
// failure reason (e.g. a ban or auth failure) is surfaced to the UI.
func (i *Instance) WaitForNetworkError(substr string, timeout time.Duration) {
	i.t.Helper()
	if !waitFor(func() bool { return i.sse.sawNetworkError(substr) }, timeout) {
		i.t.Fatalf("network never surfaced an error containing %q", substr)
	}
}

// WaitForStopped blocks until the handler has stopped (e.g. after a fatal
// network-level failure such as a ban), or fails.
func (i *Instance) WaitForStopped(timeout time.Duration) {
	i.t.Helper()
	if !waitFor(i.Handler.Stopped, timeout) {
		i.t.Fatal("handler never stopped")
	}
}

// ---- SSE sink (also the state-observability hook) ----

type stateEvent struct {
	channel string
	state   string
	errors  []string
	healthy bool
}

type sseCapture struct {
	mu           sync.Mutex
	states       []stateEvent
	healthErrors []string // connection_errors observed on network-level HEALTH events
}

func (m *sseCapture) Publish(_ string, e *sse.Event) {
	switch string(e.Event) {
	case "STATE":
		var payload struct {
			Channel          string   `json:"channel"`
			State            string   `json:"state"`
			ConnectionErrors []string `json:"connection_errors"`
			Healthy          bool     `json:"healthy"`
		}
		if err := json.Unmarshal(e.Data, &payload); err != nil {
			return
		}
		m.mu.Lock()
		m.states = append(m.states, stateEvent{channel: payload.Channel, state: payload.State, errors: payload.ConnectionErrors, healthy: payload.Healthy})
		m.mu.Unlock()

	case "HEALTH":
		var payload struct {
			ConnectionErrors []string `json:"connection_errors"`
			Healthy          bool     `json:"healthy"`
		}
		if err := json.Unmarshal(e.Data, &payload); err != nil {
			return
		}
		m.mu.Lock()
		m.healthErrors = append(m.healthErrors, payload.ConnectionErrors...)
		m.mu.Unlock()
	}
}

// CreateStreamWithOpts / RemoveStream are unused by the handler; satisfy the interface.
func (m *sseCapture) CreateStreamWithOpts(string, sse.StreamOpts) *sse.Stream { return nil }
func (m *sseCapture) RemoveStream(string)                                     {}

func (m *sseCapture) sawHealthy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.states {
		if s.healthy {
			return true
		}
	}
	return false
}

func (m *sseCapture) sawNetworkError(substr string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.healthErrors {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}

func (m *sseCapture) hasState(channel, state string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.states {
		if s.channel == channel && s.state == state {
			return true
		}
	}
	return false
}

func (m *sseCapture) statesFor(channel string) []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for _, s := range m.states {
		if s.channel == channel {
			out = append(out, s.state)
		}
	}
	return out
}

func (m *sseCapture) lastError(channel string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	last := ""
	for _, s := range m.states {
		if s.channel == channel && len(s.errors) > 0 {
			last = s.errors[len(s.errors)-1]
		}
	}
	return last
}

// ---- release sink ----

// ReleaseSink captures releases the handler produces from parsed announces.
type ReleaseSink struct {
	mu       sync.Mutex
	releases []*domain.Release
	ch       chan *domain.Release
}

func newReleaseSink() *ReleaseSink {
	return &ReleaseSink{ch: make(chan *domain.Release, 32)}
}

// Process implements the handler's releaseService.
func (s *ReleaseSink) Process(_ context.Context, release *domain.Release) {
	s.mu.Lock()
	s.releases = append(s.releases, release)
	s.mu.Unlock()
	select {
	case s.ch <- release:
	default:
	}
}

// Wait returns the next release produced, or (nil, false) on timeout.
func (s *ReleaseSink) Wait(timeout time.Duration) (*domain.Release, bool) {
	select {
	case r := <-s.ch:
		return r, true
	case <-time.After(timeout):
		return nil, false
	}
}

// Count returns how many releases have been captured so far.
func (s *ReleaseSink) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.releases)
}

// ---- no-op notifier ----

type noopNotifier struct{}

func (noopNotifier) Send(domain.NotificationEvent, domain.NotificationPayload) {}

// ---- util ----

func waitFor(cond func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}
