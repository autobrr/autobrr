// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build irc_integration_test

// Package harness wires the real internal/irc Handler up to the in-process test
// ircd: it supplies the collaborators NewHandler needs (an SSE sink, a release
// sink, a no-op notifier), drives the connection lifecycle, and exposes helpers
// to wait on channel state and captured releases.
package harness

import (
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
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

	inst := &Instance{
		t:        t,
		sse:      &sseCapture{},
		Releases: newReleaseSink(),
	}

	inst.Handler = irc.NewHandler(log, inst.sse, network, defs, inst.Releases, noopNotifier{})

	if err := inst.Handler.Run(); err != nil {
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

// ---- SSE sink (also the state-observability hook) ----

type stateEvent struct {
	channel string
	state   string
	errors  []string
}

type sseCapture struct {
	mu     sync.Mutex
	states []stateEvent
}

func (m *sseCapture) Publish(_ string, e *sse.Event) {
	if string(e.Event) != "STATE" {
		return
	}
	var payload struct {
		Channel          string   `json:"channel"`
		State            string   `json:"state"`
		ConnectionErrors []string `json:"connection_errors"`
	}
	if err := json.Unmarshal(e.Data, &payload); err != nil {
		return
	}
	m.mu.Lock()
	m.states = append(m.states, stateEvent{channel: payload.Channel, state: payload.State, errors: payload.ConnectionErrors})
	m.mu.Unlock()
}

// CreateStreamWithOpts / RemoveStream are unused by the handler; satisfy the interface.
func (m *sseCapture) CreateStreamWithOpts(string, sse.StreamOpts) *sse.Stream { return nil }
func (m *sseCapture) RemoveStream(string)                                     {}

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
func (s *ReleaseSink) Process(release *domain.Release) {
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
