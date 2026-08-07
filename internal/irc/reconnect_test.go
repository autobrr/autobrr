// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/alphadose/haxmap"
	"github.com/ergochat/irc-go/ircmsg"
	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

type noopNotificationSender struct{}

func (noopNotificationSender) Send(_ domain.NotificationEvent, _ domain.NotificationPayload) {}

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
		log:                 zerolog.Nop(),
		sse:                 sseMock,
		network:             &domain.IrcNetwork{ID: 1, Name: "TestNet", Server: "irc.example.test"},
		notificationService: noopNotificationSender{},
		definitions:         map[string]*domain.IndexerDefinition{},
		channels:            haxmap.New[string, *Channel](),
		clientState:         ircLive,
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
			if sm.CurrentState() != ChannelStateMonitoring {
				t.Fatalf("precondition: state = %s, want Monitoring", sm.CurrentState())
			}

			// an unexpected disconnect (ircevent will auto-reconnect in-process)
			h.onDisconnect(ircmsg.Message{})

			// the channel must no longer be considered monitored
			if ch.Monitoring {
				t.Errorf("channel still Monitoring after disconnect")
			}

			// and the state machine must be back at Idle so a rejoin can run
			if got := sm.CurrentState(); got != ChannelStateIdle {
				t.Fatalf("state after disconnect = %s, want Idle (reconnect trap: channel would never rejoin)", got)
			}

			// the exact transition Start() performs on reconnect must be valid
			// from the post-disconnect state; if it isn't, the channel is stuck
			// and no JOIN/invite is ever re-sent.
			from := sm.CurrentState()
			if !sm.isValidTransition(from, tt.restartTo) {
				t.Fatalf("reconnect trap: cannot restart join workflow (%s -> %s is invalid)", from, tt.restartTo)
			}

			// the UI must be told the channel left Monitoring, otherwise the pill
			// stays stale until the channel rejoins.
			if !sseMock.hasStateEvent(tt.channel, "Idle") {
				t.Errorf("expected a STATE=Idle SSE event for %s on disconnect, got %+v", tt.channel, sseMock.stateEvents())
			}
		})
	}
}

// disconnectAfter simulates an unexpected disconnect ending a session that
// lived for the given duration. Zero models a session that died before the
// connect time was recorded.
func disconnectAfter(h *Handler, lifetime time.Duration) {
	h.m.Lock()
	if lifetime > 0 {
		h.connectedSince = time.Now().Add(-lifetime)
	} else {
		h.connectedSince = time.Time{}
	}
	h.m.Unlock()

	h.onDisconnect(ircmsg.Message{})
}

// TestFlappingConnectionsTripBreaker covers the circuit breaker: the irc-go
// client reconnects every 15s forever, so a server that keeps killing us right
// after registration must stop the network at the threshold instead of being
// hammered indefinitely (the TorrentLeech 2026-08 ban-wave failure mode).
func TestFlappingConnectionsTripBreaker(t *testing.T) {
	tests := []struct {
		name     string
		lifetime time.Duration
	}{
		{"short sessions", 2 * time.Second},
		{"connect time never recorded", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, _ := newTestHandler()

			for i := 0; i < flappingStopThreshold; i++ {
				if h.Stopped() {
					t.Fatalf("network stopped after %d short sessions, want %d", i, flappingStopThreshold)
				}
				disconnectAfter(h, tt.lifetime)
			}

			if !h.Stopped() {
				t.Fatal("expected the network to stop once the flapping threshold was reached")
			}

			if !hasConnectError(h, "connection flapping") {
				t.Errorf("expected a flapping connection error, got %v", h.connectionErrors)
			}
		})
	}
}

// TestLongSessionResetsFlappingCount verifies the consecutive requirement: a
// session that lives past the threshold proves the connection can hold, so the
// count starts over and ordinary occasional disconnects never accumulate.
func TestLongSessionResetsFlappingCount(t *testing.T) {
	h, _ := newTestHandler()

	for range flappingStopThreshold - 1 {
		disconnectAfter(h, time.Second)
	}
	disconnectAfter(h, 10*time.Minute)
	for range flappingStopThreshold - 1 {
		disconnectAfter(h, time.Second)
	}

	if h.Stopped() {
		t.Fatal("breaker tripped although a long session broke the streak")
	}

	disconnectAfter(h, time.Second)

	if !h.Stopped() {
		t.Fatal("expected the network to stop on the next full streak of short sessions")
	}
}

// serverError delivers an ERROR line from the server, the way one arrives when
// the link is refused before registration completes.
func serverError(h *Handler, reason string) {
	h.handleServerError(ircmsg.Message{Command: "ERROR", Params: []string{reason}})
}

// TestPreRegistrationErrorsTripBreaker covers the blind spot the disconnect
// callback cannot see: irc-go runs it only for connections that finished
// registering, so a server that throttles us before 001 ("Trying to reconnect
// too fast") produces no disconnect event at all while Loop() keeps retrying
// every 15s forever. The ERROR line has to feed the breaker for the network to
// ever stop, and the server's own explanation is the most useful reason to show.
func TestPreRegistrationErrorsTripBreaker(t *testing.T) {
	const reason = "Trying to reconnect too fast."

	h, _ := newTestHandler()

	for range flappingStopThreshold - 1 {
		serverError(h, reason)
		if h.Stopped() {
			t.Fatal("breaker tripped before reaching the threshold")
		}
	}

	serverError(h, reason)

	if !h.Stopped() {
		t.Fatal("expected repeated pre-registration refusals to stop the network")
	}

	if !hasConnectError(h, reason) {
		t.Errorf("expected the server's own reason to be surfaced, got %v", h.connectionErrors)
	}
}

// TestServerErrorAfterRegistrationLeftToDisconnect prevents double counting: a
// server closing an established link commonly sends ERROR *and* triggers the
// disconnect callback, which would otherwise advance the streak twice for one
// session.
func TestServerErrorAfterRegistrationLeftToDisconnect(t *testing.T) {
	h, _ := newTestHandler()

	h.m.Lock()
	h.connectedSince = time.Now().Add(-time.Second)
	h.m.Unlock()

	serverError(h, "Closing link: (ping timeout)")

	h.m.RLock()
	count := h.consecutiveShortSessions
	h.m.RUnlock()

	if count != 0 {
		t.Errorf("registered session must be accounted by onDisconnect only, got count %d", count)
	}
}

// TestServerErrorOnManualStopIgnored keeps our own QUIT out of the breaker:
// most servers answer it with an ERROR line.
func TestServerErrorOnManualStopIgnored(t *testing.T) {
	h, _ := newTestHandler()
	h.m.Lock()
	h.clientState = ircStopped
	h.m.Unlock()

	for range flappingStopThreshold + 1 {
		serverError(h, "Closing link: (Quit: bye from autobrr)")
	}

	if hasConnectError(h, "connection flapping") {
		t.Errorf("our own quit must not trip the breaker, got %v", h.connectionErrors)
	}
}

// TestMixedFailureModesShareOneStreak verifies the two feeds are one breaker: a
// server that alternates between refusing us early and dropping us just after
// registration must still be counted as flapping.
func TestMixedFailureModesShareOneStreak(t *testing.T) {
	h, _ := newTestHandler()

	serverError(h, "Trying to reconnect too fast.")
	disconnectAfter(h, time.Second)
	serverError(h, "Trying to reconnect too fast.")
	disconnectAfter(h, time.Second)

	if h.Stopped() {
		t.Fatal("breaker tripped before reaching the threshold")
	}

	serverError(h, "Trying to reconnect too fast.")

	if !h.Stopped() {
		t.Fatal("expected mixed early refusals and short sessions to share one streak")
	}
}

// TestManualStopDoesNotCountTowardFlapping keeps user-initiated stops and
// restarts out of the breaker: only disconnects the server forced on us count.
func TestManualStopDoesNotCountTowardFlapping(t *testing.T) {
	h, _ := newTestHandler()
	h.m.Lock()
	h.clientState = ircStopped
	h.m.Unlock()

	for range flappingStopThreshold + 1 {
		disconnectAfter(h, time.Second)
	}

	if hasConnectError(h, "connection flapping") {
		t.Errorf("manual disconnects must not trip the breaker, got %v", h.connectionErrors)
	}

	h.m.RLock()
	defer h.m.RUnlock()
	if h.consecutiveShortSessions != 0 {
		t.Errorf("consecutiveShortSessions = %d, want 0 for manual disconnects", h.consecutiveShortSessions)
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

	if got := sm.CurrentState(); got != ChannelStateIdle {
		t.Fatalf("Reset() from Monitoring left state = %s, want Idle", got)
	}
	if sm.authAttempts != 0 {
		t.Errorf("Reset() left authAttempts = %d, want 0", sm.authAttempts)
	}
	if sm.joinAfterInvite {
		t.Errorf("Reset() left joinAfterInvite = true, want false")
	}
	if !sseMock.hasStateEvent("#announce", "Idle") {
		t.Errorf("Reset() did not broadcast STATE=Idle, got %+v", sseMock.stateEvents())
	}
}
