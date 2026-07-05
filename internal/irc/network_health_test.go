// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
)

// healthEvents returns the decoded payloads of all published HEALTH events.
func (m *mockSSEServer) healthEvents() []map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()

	var out []map[string]any
	for _, e := range m.published {
		if string(e.Event) != "HEALTH" {
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

// healthEventHasError reports whether a HEALTH event for networkID carried a
// connection_errors entry containing substr. Numeric JSON decodes to float64.
func healthEventHasError(m *mockSSEServer, networkID int64, substr string) bool {
	for _, ev := range m.healthEvents() {
		if nid, ok := ev["network"].(float64); !ok || int64(nid) != networkID {
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

// healthEventHealthy returns the healthy flag of the most recent HEALTH event for
// networkID.
func healthEventHealthy(m *mockSSEServer, networkID int64) (healthy bool, found bool) {
	for _, ev := range m.healthEvents() {
		if nid, ok := ev["network"].(float64); !ok || int64(nid) != networkID {
			continue
		}
		if hv, ok := ev["healthy"].(bool); ok {
			healthy, found = hv, true
		}
	}
	return healthy, found
}

// TestAddConnectErrorDedups verifies repeated identical network-level errors do
// not stack - e.g. NickServ resending the same "registered and protected" NOTICE
// several times, or a reconnect re-hitting the same failure.
func TestAddConnectErrorDedups(t *testing.T) {
	h, _ := newTestHandler()
	const reason = "authentication failed: nick in use and not authenticated"

	h.addConnectError(reason)
	h.addConnectError(reason)
	h.addConnectError(reason)

	h.m.RLock()
	n := len(h.connectionErrors)
	h.m.RUnlock()

	if n != 1 {
		t.Fatalf("duplicate network errors should be deduped, got %d", n)
	}
}

// TestReportStatusSurfacesErrorsWhenStopped is a regression test for the invisible
// network failure: a NickServ auth failure drives the network to Error and Stop()s
// it (client nilled, connectedSince cleared). ReportStatus must still surface the
// stored connection error so the UI shows WHY the network is unhealthy, not just
// that it is.
func TestReportStatusSurfacesErrorsWhenStopped(t *testing.T) {
	h, _ := newTestHandler()

	h.m.Lock()
	h.network.Enabled = true
	h.connectionErrors = []string{"authentication failed: nick in use and not authenticated"}
	h.client = nil // Stop() nils the client
	h.m.Unlock()

	var netw domain.IrcNetworkWithHealth
	h.ReportStatus(&netw)

	if len(netw.ConnectionErrors) == 0 || !strings.Contains(netw.ConnectionErrors[0], "nick in use") {
		t.Fatalf("a stopped network must still surface its connection error, got %v", netw.ConnectionErrors)
	}
}

// TestNetworkErrorBroadcastsHealthWithReason verifies the network-level failure
// reason is pushed to the UI in real time (a HEALTH event) when the connection
// state machine enters Error, rather than only appearing on the next poll.
func TestNetworkErrorBroadcastsHealthWithReason(t *testing.T) {
	h, sse := newTestHandler()

	// a state from which Error is a valid transition
	h.stateMachine.m.Lock()
	h.stateMachine.currentState = StateFullyOperational
	h.stateMachine.m.Unlock()

	// the reason is recorded before OnError drives the transition (as in handleNickServ)
	h.addConnectError("authentication failed: nick in use and not authenticated")
	h.stateMachine.OnError("nickserv authentication failed: nick in use")

	if !waitFor(func() bool { return healthEventHasError(sse, h.network.ID, "nick in use") }, time.Second) {
		t.Fatal("entering network Error should broadcast a HEALTH event carrying the connection error reason")
	}
}

// TestNetworkOperationalBroadcastsHealthy verifies that reaching an operational
// state broadcasts a healthy HEALTH event (so a previously-shown error clears in
// the UI in real time).
func TestNetworkOperationalBroadcastsHealthy(t *testing.T) {
	h, sse := newTestHandler()

	h.stateMachine.m.Lock()
	h.stateMachine.currentState = StateJoiningChannels
	h.stateMachine.m.Unlock()

	// no enabled default channels are unhealthy, so computeHealthy() is true here
	h.stateMachine.transition(StateFullyOperational)

	if !waitFor(func() bool {
		healthy, found := healthEventHealthy(sse, h.network.ID)
		return found && healthy
	}, time.Second) {
		t.Fatal("reaching FullyOperational should broadcast a healthy HEALTH event")
	}
}
