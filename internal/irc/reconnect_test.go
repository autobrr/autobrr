// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"

	"github.com/alphadose/haxmap"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type noopEventBus struct{}

func (noopEventBus) EmitIRC(_ context.Context, _ events.IRCEvent) {}

func (noopEventBus) OnProxy(_ func(context.Context, events.ProxyChangeEvent) error) func() {
	return func() {}
}

// mockSSEServer records published events so tests can assert what was broadcast.
type mockSSEServer struct {
	mu        sync.Mutex
	published []*sse.Event
}

func (m *mockSSEServer) Publish(_ string, event *sse.Event) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.published = append(m.published, event)
}

func (m *mockSSEServer) CreateStreamWithOpts(_ string, _ sse.StreamOpts) *sse.Stream { return nil }
func (m *mockSSEServer) RemoveStream(_ string)                                       {}

// stateEvents returns the decoded payloads of all published STATE events.
func (m *mockSSEServer) stateEvents() []map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	var out []map[string]any
	for _, e := range m.published {
		if string(e.Event) != "STATE" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(e.Data, &payload); err != nil {
			continue
		}
		out = append(out, payload)
	}
	return out
}

func (m *mockSSEServer) hasStateEvent(channel, state string) bool {
	for _, ev := range m.stateEvents() {
		if ev["channel"] == channel && ev["state"] == state {
			return true
		}
	}
	return false
}

// stateEventHealthy returns the healthy flag from a STATE event for channel/state.
func stateEventHealthy(m *mockSSEServer, channel, state string) (healthy bool, found bool) {
	for _, ev := range m.stateEvents() {
		if ev["channel"] != channel || ev["state"] != state {
			continue
		}
		if hv, ok := ev["healthy"].(bool); ok {
			return hv, true
		}
	}
	return false, false
}

// stateEventHasError reports whether a STATE event for channel/state carried a
// connection_errors entry containing substr.
func stateEventHasError(m *mockSSEServer, channel, state, substr string) bool {
	for _, ev := range m.stateEvents() {
		if ev["channel"] != channel || ev["state"] != state {
			continue
		}
		errs, ok := ev["connection_errors"].([]any)
		if !ok {
			continue
		}
		for _, e := range errs {
			if s, ok := e.(string); ok && strings.Contains(s, substr) {
				return true
			}
		}
	}
	return false
}

// newTestHandler builds a Handler wired up just enough to exercise the
// disconnect/reconnect bookkeeping without a live IRC connection.
func newTestHandler() (*Handler, *mockSSEServer) {
	sseMock := &mockSSEServer{}
	h := &Handler{
		log:         zerolog.Nop(),
		sse:         sseMock,
		network:     &domain.IrcNetwork{ID: 1, Name: "TestNet", Server: "irc.example.test"},
		eventBus:    noopEventBus{},
		definitions: map[string]*domain.IndexerDefinition{},
		channels:    haxmap.New[string, *Channel](),
		clientState: ircLive,
	}
	h.stateMachine = NewConnectionStateMachine(h)
	return h, sseMock
}

// addMonitoredChannel registers a channel that is already joined and being
// monitored, i.e. the steady state right before a connection drop.
func addMonitoredChannel(h *Handler, name, inviteCommand string) *ChannelStateMachine {
	ch := NewChannel(zerolog.Nop(), h.network.ID, name, true, false, nil)
	sm := NewChannelStateMachine(ch, h, inviteCommand)
	ch.SetStateMachine(sm)
	h.channels.Set(ch.Name, ch)

	// simulate a successful join: end-of-NAMES -> Monitoring
	ch.SetMonitoring()
	sm.state = ChannelStateMonitoring

	return sm
}

// TestReconnectResetsChannelStateMachines is a regression test for the reconnect
// trap: ChannelStateMonitoring has no outgoing transitions, so unless the state
// machine is reset on disconnect a channel can never rejoin and announces stop
// after the first unexpected disconnect. See channel_state_machine.go Reset().
func TestReconnectResetsChannelStateMachines(t *testing.T) {
	tests := []struct {
		name          string
		channel       string
		inviteCommand string
		// the transition Start() attempts on the next connect for this channel
		restartTo ChannelState
	}{
		{
			name:      "direct join channel",
			channel:   "#announce",
			restartTo: ChannelStateJoining,
		},
		{
			name:          "invite channel",
			channel:       "#invite-announce",
			inviteCommand: "invitebot !invite user pass",
			restartTo:     ChannelStateAwaitingInvite,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, sseMock := newTestHandler()
			// put the connection SM in a state a disconnect can legally leave
			h.stateMachine.currentState = StateFullyOperational

			sm := addMonitoredChannel(h, tt.channel, tt.inviteCommand)
			ch, _ := h.channels.Get(tt.channel)

			// sanity: the channel is monitored before the drop
			require.Equal(t, ChannelStateMonitoring, sm.CurrentState(), "precondition")

			// an unexpected disconnect (ircevent will auto-reconnect in-process)
			h.onDisconnect(ircmsg.Message{})

			// the channel must no longer be considered monitored
			assert.False(t, ch.Monitoring, "channel still Monitoring after disconnect")

			// and the state machine must be back at Idle so a rejoin can run
			require.Equal(t, ChannelStateIdle, sm.CurrentState(), "state after disconnect (reconnect trap: channel would never rejoin)")

			// the exact transition Start() performs on reconnect must be valid
			// from the post-disconnect state; if it isn't, the channel is stuck
			// and no JOIN/invite is ever re-sent.
			from := sm.CurrentState()
			require.Truef(t, sm.isValidTransition(from, tt.restartTo), "reconnect trap: cannot restart join workflow (%s -> %s is invalid)", from, tt.restartTo)

			// the UI must be told the channel left Monitoring, otherwise the pill
			// stays stale until the channel rejoins.
			assert.Truef(t, sseMock.hasStateEvent(tt.channel, "Idle"), "expected a STATE=Idle SSE event for %s on disconnect, got %+v", tt.channel, sseMock.stateEvents())
		})
	}
}

// TestResetFromMonitoring verifies Reset() escapes the dead-end Monitoring state
// directly, clears the invite bookkeeping, and broadcasts the new state.
func TestResetFromMonitoring(t *testing.T) {
	h, sseMock := newTestHandler()
	sm := addMonitoredChannel(h, "#announce", "")
	ch, _ := h.channels.Get("#announce")
	ch.SetConnectionError("boom")

	sm.Reset()

	require.Equal(t, ChannelStateIdle, sm.CurrentState(), "Reset() from Monitoring")
	assert.Zero(t, sm.authAttempts, "Reset() left authAttempts")
	assert.False(t, sm.joinAfterInvite, "Reset() left joinAfterInvite set")
	assert.Truef(t, sseMock.hasStateEvent("#announce", "Idle"), "Reset() did not broadcast STATE=Idle, got %+v", sseMock.stateEvents())
}

// refusingNetwork returns a network pointing at a local port nothing listens on,
// so every connect attempt is refused, like a tracker IRC server that is down.
func refusingNetwork(t *testing.T) *domain.IrcNetwork {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	return &domain.IrcNetwork{ID: 1, Name: "TestNet", Server: "127.0.0.1", Port: port, Nick: "autobrr", Enabled: true}
}

func runAsync(h *Handler) <-chan error {
	done := make(chan error, 1)
	go func() { done <- h.Run() }()

	return done
}

// TestStopResetsConnectionState covers https://github.com/autobrr/autobrr/issues/2760:
// Stop must leave the network state machine Disconnected so the next Run can climb
// through Connected again, without waiting for the old client's disconnect callback.
func TestStopResetsConnectionState(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	h.Stop()

	assert.Equal(t, StateDisconnected, h.stateMachine.GetState())
}

// TestStopInterruptsConnectBackoff verifies a network waiting out the connect
// backoff, which keeps retrying for as long as the server is down, stops right away.
func TestStopInterruptsConnectBackoff(t *testing.T) {
	h, _ := newTestHandler()
	h.network = refusingNetwork(t)
	h.clientState = ircStopped

	done := runAsync(h)

	// the refusal is immediate, so this is well inside the first backoff delay
	time.Sleep(200 * time.Millisecond)
	h.Stop()

	select {
	case err := <-done:
		require.ErrorIs(t, err, clientManuallyDisconnected)
	case <-time.After(2 * time.Second):
		t.Fatal("Run kept waiting out the connect backoff after Stop")
	}

	assert.Equal(t, StateDisconnected, h.stateMachine.GetState())
}

// TestRestartDuringConnectBackoffKeepsNewRun verifies the Run interrupted by a
// restart does not reset the state of the Run that replaced it when it unwinds.
func TestRestartDuringConnectBackoffKeepsNewRun(t *testing.T) {
	h, _ := newTestHandler()
	h.network = refusingNetwork(t)
	h.clientState = ircStopped

	first := runAsync(h)
	time.Sleep(200 * time.Millisecond)

	// Stop and the claim of a new Run, made before the first Run is cancelled. A real
	// Stop cancels first, and the first Run often unwinds before the new Run claims the
	// handler, which would pass without the generation check.
	h.m.Lock()
	cancel := h.cancelConnect
	h.cancelConnect = nil
	h.connectGen++
	h.clientState = ircConnecting
	h.m.Unlock()
	h.stateMachine.OnDisconnected()
	h.stateMachine.OnConnecting()

	require.NotNil(t, cancel)
	cancel()

	select {
	case err := <-first:
		require.ErrorIs(t, err, clientManuallyDisconnected)
	case <-time.After(2 * time.Second):
		t.Fatal("first Run did not return after Stop")
	}

	assert.False(t, h.Stopped(), "the first Run unwinding stopped the second one")
	assert.Equal(t, StateConnecting, h.stateMachine.GetState())
}

// registrationServer is an IRC server that completes a client's registration only
// when the test says so, holding a connect attempt in flight for as long as needed.
type registrationServer struct {
	port  int
	conns chan net.Conn
}

func newRegistrationServer(t *testing.T) *registrationServer {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	s := &registrationServer{port: ln.Addr().(*net.TCPAddr).Port, conns: make(chan net.Conn, 4)}

	var accepted []net.Conn
	var m sync.Mutex

	t.Cleanup(func() {
		ln.Close()

		m.Lock()
		defer m.Unlock()
		for _, conn := range accepted {
			conn.Close()
		}
	})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			m.Lock()
			accepted = append(accepted, conn)
			m.Unlock()

			s.conns <- conn
		}
	}()

	return s
}

func (s *registrationServer) network() *domain.IrcNetwork {
	return &domain.IrcNetwork{ID: 1, Name: "TestNet", Server: "127.0.0.1", Port: s.port, Nick: "autobrr", Enabled: true}
}

func (s *registrationServer) accept(t *testing.T) net.Conn {
	t.Helper()

	select {
	case conn := <-s.conns:
		return conn
	case <-time.After(2 * time.Second):
		t.Fatal("no connect attempt reached the server")
		return nil
	}
}

func (s *registrationServer) register(t *testing.T, conn net.Conn) {
	t.Helper()

	_, err := conn.Write([]byte(":irc.test 001 autobrr :Welcome\r\n:irc.test 422 autobrr :MOTD File is missing\r\n"))
	require.NoError(t, err)
}

// TestStopDuringConnectAttempt verifies a client that finishes registering after
// Stop, which cannot abort a Connect in progress, is dropped and does not drive
// the network state machine.
func TestStopDuringConnectAttempt(t *testing.T) {
	srv := newRegistrationServer(t)

	h, _ := newTestHandler()
	h.network = srv.network()
	h.clientState = ircStopped

	done := runAsync(h)
	conn := srv.accept(t)

	h.Stop()
	srv.register(t, conn)

	select {
	case err := <-done:
		require.ErrorIs(t, err, clientManuallyDisconnected)
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after Stop")
	}

	assert.True(t, h.Stopped())
	assert.Nil(t, h.getClient())

	// onConnect waits a second before it advances the state machine
	assert.Never(t, func() bool { return h.stateMachine.GetState() != StateDisconnected }, 1500*time.Millisecond, 50*time.Millisecond)
}

// TestStaleRunDoesNotTakeOverRestart verifies a Run whose Connect was in flight
// across a restart, and registers before the new Run does, neither goes live nor
// drives the state machine, leaving the network to the new Run.
func TestStaleRunDoesNotTakeOverRestart(t *testing.T) {
	srv := newRegistrationServer(t)

	h, _ := newTestHandler()
	h.network = srv.network()
	h.clientState = ircStopped

	first := runAsync(h)
	staleConn := srv.accept(t)

	h.Stop()

	second := runAsync(h)
	conn := srv.accept(t)

	srv.register(t, staleConn)

	select {
	case err := <-first:
		require.ErrorIs(t, err, clientManuallyDisconnected)
	case <-time.After(2 * time.Second):
		t.Fatal("stale Run did not return")
	}

	assert.False(t, h.Stopped(), "the stale Run stopped the new one")
	assert.Never(t, func() bool { return h.stateMachine.GetState() != StateConnecting }, 1500*time.Millisecond, 50*time.Millisecond)

	srv.register(t, conn)

	select {
	case err := <-second:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("new Run did not return")
	}

	require.Eventually(t, func() bool { return h.stateMachine.GetState() == StateJoiningChannels }, 3*time.Second, 50*time.Millisecond)

	h.Stop()
}
