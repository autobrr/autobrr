// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newHealthTestService(t *testing.T) (*Service, *recordingEventBus) {
	t.Helper()

	bus := &recordingEventBus{}
	s := NewService(zerolog.Nop(), bus, &mockSSEServer{}, nil, nil, stubIndexerService{}, stubProxyService{})

	return s, bus
}

func addHealthTestHandler(s *Service, id int64, name string, state ConnectionState) *Handler {
	h, _ := newTestHandler()
	h.network.ID = id
	h.network.Name = name
	h.stateMachine.currentState = state
	s.networkHandlers.Set(id, h)

	return h
}

func stopWithError(h *Handler, reason string) {
	h.clientState = ircStopped
	h.connectionErrors = []string{reason}
	h.stateMachine.currentState = StateDisconnected
}

func TestCheckNetworkHealth_GracePeriod(t *testing.T) {
	s, bus := newHealthTestService(t)
	h := addHealthTestHandler(s, 1, "TorrentLeech", StateDisconnected)

	start := time.Now()

	s.checkNetworkHealth(t.Context(), start)
	s.checkNetworkHealth(t.Context(), start.Add(unhealthyGracePeriod-time.Minute))
	require.Empty(t, bus.snapshot(), "a network inside the grace period must not be reported")

	s.checkNetworkHealth(t.Context(), start.Add(unhealthyGracePeriod))
	s.checkNetworkHealth(t.Context(), start.Add(unhealthyGracePeriod+time.Minute))

	got := bus.snapshot()
	require.Len(t, got, 1, "an unhealthy episode must be reported exactly once")
	assert.Equal(t, events.IRCUnhealthy, got[0].Type)
	assert.Equal(t, "TorrentLeech", got[0].Network)
	assert.Equal(t, "TorrentLeech: connection state: Disconnected", got[0].Message)

	h.stateMachine.currentState = StateFullyOperational
	s.checkNetworkHealth(t.Context(), start.Add(unhealthyGracePeriod+2*time.Minute))

	got = bus.snapshot()
	require.Len(t, got, 2)
	assert.Equal(t, events.IRCHealthy, got[1].Type)
	assert.Equal(t, "TorrentLeech", got[1].Network)
}

func TestCheckNetworkHealth_ShortOutageIsSilent(t *testing.T) {
	s, bus := newHealthTestService(t)
	h := addHealthTestHandler(s, 1, "TorrentLeech", StateDisconnected)

	start := time.Now()
	s.checkNetworkHealth(t.Context(), start)

	h.stateMachine.currentState = StateFullyOperational
	s.checkNetworkHealth(t.Context(), start.Add(5*time.Minute))

	h.stateMachine.currentState = StateDisconnected
	s.checkNetworkHealth(t.Context(), start.Add(6*time.Minute))
	s.checkNetworkHealth(t.Context(), start.Add(6*time.Minute+unhealthyGracePeriod-time.Second))

	assert.Empty(t, bus.snapshot(), "recovering inside the grace period must restart the episode and send nothing")
}

func TestCheckNetworkHealth_FatalStopReportsImmediately(t *testing.T) {
	s, bus := newHealthTestService(t)
	h := addHealthTestHandler(s, 1, "PTP", StateFullyOperational)
	stopWithError(h, "banned from network: K-Lined")

	s.checkNetworkHealth(t.Context(), time.Now())

	got := bus.snapshot()
	require.Len(t, got, 1)
	assert.Equal(t, events.IRCUnhealthy, got[0].Type)
	assert.Equal(t, "PTP: banned from network: K-Lined", got[0].Message)
}

func TestCheckNetworkHealth_UserStopIsIgnored(t *testing.T) {
	s, bus := newHealthTestService(t)
	h := addHealthTestHandler(s, 1, "PTP", StateDisconnected)
	h.clientState = ircStopped

	start := time.Now()
	s.checkNetworkHealth(t.Context(), start)
	s.checkNetworkHealth(t.Context(), start.Add(2*unhealthyGracePeriod))

	assert.Empty(t, bus.snapshot())
}

func TestCheckNetworkHealth_BatchesNetworks(t *testing.T) {
	s, bus := newHealthTestService(t)
	stopWithError(addHealthTestHandler(s, 1, "TorrentLeech", StateDisconnected), "authentication failed: account does not exist")
	stopWithError(addHealthTestHandler(s, 2, "BroadcasTheNet", StateDisconnected), "TLS certificate verification failed: x509: certificate has expired")

	s.checkNetworkHealth(t.Context(), time.Now())

	got := bus.snapshot()
	require.Len(t, got, 1, "networks that go unhealthy together must share one notification")
	assert.Equal(t, "BroadcasTheNet, TorrentLeech", got[0].Network)
	assert.Equal(t, "BroadcasTheNet: TLS certificate verification failed: x509: certificate has expired\nTorrentLeech: authentication failed: account does not exist", got[0].Message)
}

func TestCheckNetworkHealth_AnnounceChannelReasons(t *testing.T) {
	s, bus := newHealthTestService(t)
	h := addHealthTestHandler(s, 1, "PTP", StateFullyOperational)

	announce := NewChannel(zerolog.Nop(), h.network.ID, "#ptp-announce", true, false, nil)
	announce.SetConnectionError("invite bot responded but did not grant access")
	h.channels.Set(announce.Name, announce)

	extra := NewChannel(zerolog.Nop(), h.network.ID, "#extra", false, false, nil)
	extra.SetConnectionError("could not join #extra: wrong or missing channel password (+k)")
	h.channels.Set(extra.Name, extra)

	start := time.Now()
	s.checkNetworkHealth(t.Context(), start)
	s.checkNetworkHealth(t.Context(), start.Add(unhealthyGracePeriod))

	got := bus.snapshot()
	require.Len(t, got, 1)
	assert.Equal(t, "PTP: #ptp-announce: invite bot responded but did not grant access", got[0].Message)
}

func TestCheckNetworkHealth_RemovedNetworkSendsNoRecovery(t *testing.T) {
	s, bus := newHealthTestService(t)
	stopWithError(addHealthTestHandler(s, 1, "PTP", StateDisconnected), "authentication failed: SASL negotiation failed")

	s.checkNetworkHealth(t.Context(), time.Now())
	require.Len(t, bus.snapshot(), 1)

	s.networkHandlers.Del(1)
	s.checkNetworkHealth(t.Context(), time.Now())

	assert.Len(t, bus.snapshot(), 1)
	assert.Empty(t, s.unhealthyNetworks)
}

func TestCheckNetworkHealth_StartupProxyLookupFailure(t *testing.T) {
	bus := &recordingEventBus{}
	s := NewService(zerolog.Nop(), bus, &mockSSEServer{}, newStubIrcRepo(proxiedNetwork()), nil, stubIndexerService{}, stubProxyService{err: errors.New("database is locked")})

	s.StartHandlers()
	t.Cleanup(s.StopHandlers)

	h, found := s.networkHandlers.Get(proxiedNetwork().ID)
	require.True(t, found, "a network whose proxy lookup failed must still get a handler")

	require.Eventually(t, func() bool {
		return h.healthStatus().stopped
	}, time.Second, 10*time.Millisecond)

	s.checkNetworkHealth(t.Context(), time.Now())

	got := bus.snapshot()
	require.Len(t, got, 1)
	assert.Equal(t, events.IRCUnhealthy, got[0].Type)
	assert.Equal(t, "net: configured proxy could not be loaded", got[0].Message)
}

func TestCheckNetworkHealth_ReplacedHandlerStartsNewEpisode(t *testing.T) {
	s, bus := newHealthTestService(t)
	stopWithError(addHealthTestHandler(s, 1, "PTP", StateDisconnected), "authentication failed: SASL negotiation failed")

	start := time.Now()
	s.checkNetworkHealth(t.Context(), start)
	require.Len(t, bus.snapshot(), 1)

	// disabled and re-enabled between checks: a new handler under the same id
	h := addHealthTestHandler(s, 1, "PTP", StateFullyOperational)
	s.checkNetworkHealth(t.Context(), start.Add(time.Minute))
	assert.Len(t, bus.snapshot(), 1, "the new handler must not send a recovery for the discarded episode")

	stopWithError(h, "banned from network: K-Lined")
	s.checkNetworkHealth(t.Context(), start.Add(2*time.Minute))

	got := bus.snapshot()
	require.Len(t, got, 2, "a fatal failure on the new handler must be reported")
	assert.Equal(t, "PTP: banned from network: K-Lined", got[1].Message)
}

func TestCheckNetworkHealth_ReplacedHandlerGetsFullGracePeriod(t *testing.T) {
	s, bus := newHealthTestService(t)
	addHealthTestHandler(s, 1, "PTP", StateDisconnected)

	start := time.Now()
	s.checkNetworkHealth(t.Context(), start)

	addHealthTestHandler(s, 1, "PTP", StateDisconnected)
	s.checkNetworkHealth(t.Context(), start.Add(10*time.Minute))
	s.checkNetworkHealth(t.Context(), start.Add(unhealthyGracePeriod))

	assert.Empty(t, bus.snapshot(), "the new handler's grace period starts at its first check")
}
