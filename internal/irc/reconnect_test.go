// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"encoding/json"
	"sync"
	"testing"

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

// newTestHandler builds a Handler wired up just enough to exercise the
// disconnect/reconnect bookkeeping without a live IRC connection.
func newTestHandler() (*Handler, *mockSSEServer) {
	sseMock := &mockSSEServer{}
	h := &Handler{
		log:                 zerolog.Nop(),
		sse:                 sseMock,
		network:             &domain.IrcNetwork{ID: 1, Name: "TestNet", Server: "irc.example.test"},
		notificationService: noopNotificationSender{},
		channels:            haxmap.New[string, *Channel](),
		bots:                haxmap.New[string, *domain.IrcUser](),
		clientState:         ircLive,
	}
	h.stateMachine = NewConnectionStateMachine(h)
	return h, sseMock
}

// addMonitoredChannel registers a channel that is already joined and being
// monitored, i.e. the steady state right before a connection drop.
func addMonitoredChannel(h *Handler, name, inviteCommand string) *ChannelStateMachine {
	ch := NewChannel(zerolog.Nop(), h.network.ID, name, true, nil)
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
