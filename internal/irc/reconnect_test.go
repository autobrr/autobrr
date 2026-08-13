// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"crypto/tls"
	"encoding/json"
	stdErr "errors"
	"io"
	"log"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/alphadose/haxmap"
	"github.com/ergochat/irc-go/ircevent"
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

func (m *mockSSEServer) eventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.published)
}

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

// TestFlappingConnectionsTripBreaker covers the circuit breaker: a server that
// keeps killing us right after registration must stop the network at the
// threshold instead of being reconnected to indefinitely (the TorrentLeech
// 2026-08 ban-wave failure mode).
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

// TestSupersededSessionEndIsIgnored is the regression test for orphaned
// connections. An attempt that completed after the network was stopped, or that
// was overtaken by a restart, is not the network's connection any more - and
// because handler state is shared, letting its eventual disconnect run the
// normal bookkeeping tears down the session that IS live: channels drop out of
// monitoring with nothing to rejoin them, the state machine lands in
// Disconnected with no route back to operational, and the failure is charged to
// the live session's flapping streak.
func TestSupersededSessionEndIsIgnored(t *testing.T) {
	h, sseMock := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	sm := addMonitoredChannel(h, "#announce", "")
	ch, _ := h.channels.Get("#announce")

	// the live session, plus a newer attempt that has taken ownership
	staleGen := uint64(1)
	h.m.Lock()
	h.clientGen = 2
	h.connectedSince = time.Now().Add(-time.Hour)
	h.m.Unlock()

	sessionEnded := make(chan uint64, 1)
	h.onSessionEnded(staleGen, ircmsg.Message{Command: "DISCONNECT"}, sessionEnded)

	if !ch.Monitoring {
		t.Error("a superseded connection ending must not drop the live session's channel out of monitoring")
	}

	if got := sm.CurrentState(); got != ChannelStateMonitoring {
		t.Errorf("channel state = %s, want Monitoring", got)
	}

	if got := h.stateMachine.GetState(); got == StateDisconnected {
		t.Error("a superseded connection ending must not report the live network as disconnected")
	}

	h.m.RLock()
	count := h.consecutiveShortSessions
	connected := h.connectedSince
	h.m.RUnlock()

	if count != 0 {
		t.Errorf("superseded session counted toward the breaker: %d", count)
	}

	if connected.IsZero() {
		t.Error("a superseded connection ending cleared the live session's connect time")
	}

	if len(sessionEnded) != 0 {
		t.Error("a superseded connection must not wake the supervisor to reconnect")
	}

	if sseMock.hasStateEvent("#announce", "Idle") {
		t.Error("a superseded connection ending was broadcast to the UI")
	}
}

// TestCurrentSessionEndIsHandled is the counterpart: the guard must not swallow
// the real thing, including a user-initiated stop, where nothing newer exists.
func TestCurrentSessionEndIsHandled(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	sm := addMonitoredChannel(h, "#announce", "")

	h.m.Lock()
	h.clientGen = 7
	h.m.Unlock()

	sessionEnded := make(chan uint64, 1)
	h.onSessionEnded(7, ircmsg.Message{Command: "DISCONNECT"}, sessionEnded)

	if got := sm.CurrentState(); got != ChannelStateIdle {
		t.Errorf("channel state = %s, want Idle so it can rejoin", got)
	}

	if len(sessionEnded) != 1 {
		t.Error("the supervisor was not woken to reconnect the current session")
	}
}

// TestReleaseConnectingStateRespectsNewerRun is the regression test for the
// restart race: Stop now wakes a backing-off run immediately, and the service
// starts the replacement run in the same breath. The woken run must not clear
// the connecting state its successor has already claimed, or the restart aborts
// mid-connect and the UI reports success for a network that stays down.
func TestReleaseConnectingStateRespectsNewerRun(t *testing.T) {
	h, _ := newTestHandler()

	oldRun := make(chan struct{})
	newRun := make(chan struct{})

	h.m.Lock()
	h.clientState = ircConnecting
	h.stopSig = newRun
	h.m.Unlock()

	h.releaseConnectingState(oldRun)

	h.m.RLock()
	state := h.clientState
	h.m.RUnlock()

	if state != ircConnecting {
		t.Fatal("a superseded run cleared the connecting state claimed by its replacement")
	}

	// the run that still owns the state must of course release it
	h.releaseConnectingState(newRun)

	if !h.Stopped() {
		t.Error("the owning run did not release the connecting state")
	}
}

// claimRun marks stopSig as the handler's current run, the way Run does when it
// claims the connecting state; connectOnce refuses attempts from any other run.
func claimRun(h *Handler, stopSig chan struct{}) {
	h.m.Lock()
	h.stopSig = stopSig
	h.m.Unlock()
}

// closedPort returns a loopback address nothing is listening on, so a connect
// attempt fails immediately instead of waiting out a dial timeout.
func closedPort(t *testing.T) (string, int) {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	addr := ln.Addr().(*net.TCPAddr)
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}

	return addr.IP.String(), addr.Port
}

// TestConnectWithBackoffRetriesUntilStopped exercises the retry loop itself:
// a failed attempt has to be paced and tried again, and the wait has to end the
// moment the network is stopped rather than running out the full delay.
func TestConnectWithBackoffRetriesUntilStopped(t *testing.T) {
	h, _ := newTestHandler()
	h.network.Server, h.network.Port = closedPort(t)

	sessionEnded := make(chan uint64, 1)
	stopSig := make(chan struct{})
	claimRun(h, stopSig)

	go func() {
		time.Sleep(300 * time.Millisecond)
		close(stopSig)
	}()

	start := time.Now()
	client, err := h.connectWithBackoff(sessionEnded, stopSig)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected connecting to a closed port to fail")
	}

	if client != nil {
		t.Error("no client should be returned when connecting failed")
	}

	if elapsed < 250*time.Millisecond {
		t.Errorf("returned after %s: the failed attempt was not paced before retrying", elapsed)
	}

	h.m.RLock()
	attempts := h.connectAttempts
	h.m.RUnlock()

	if attempts == 0 {
		t.Error("a failed attempt did not advance the backoff schedule")
	}

	// the delay has to be honoured, not merely computed: without the wait the
	// loop spins on the failing server as fast as it can dial
	if attempts > 3 {
		t.Errorf("%d attempts in %s: the backoff delay is not being waited out", attempts, elapsed)
	}

	if h.Stopped() {
		t.Error("a network that is merely unreachable must not be stopped")
	}
}

// TestConnectWithBackoffStopsOnUnusableConfig covers the fatal branch: a failure
// that comes from the network's own settings repeats identically on every
// attempt, so it must stop with a reason instead of retrying forever.
func TestConnectWithBackoffStopsOnUnusableConfig(t *testing.T) {
	h, _ := newTestHandler()
	h.network.UseProxy = true
	h.network.Proxy = &domain.Proxy{Enabled: true, Addr: "://not-a-url"}

	sessionEnded := make(chan uint64, 1)
	stopSig := make(chan struct{})

	// bail the loop out if the failure is ever treated as retryable, so that
	// regressing this fails the test instead of hanging it
	go func() {
		time.Sleep(2 * time.Second)
		close(stopSig)
	}()

	if _, err := h.connectWithBackoff(sessionEnded, stopSig); err == nil {
		t.Fatal("expected an unusable proxy configuration to fail")
	}

	if !h.Stopped() {
		t.Error("an unusable configuration must stop the network, not be retried forever")
	}

	if len(h.connectionErrors) == 0 {
		t.Error("the reason the network cannot connect was never surfaced")
	}
}

// TestFatalConnectErrorRecognisesBadCertificate pins the certificate case, which
// now applies to every attempt rather than only the first connect.
func TestFatalConnectErrorRecognisesBadCertificate(t *testing.T) {
	certErr := &tls.CertificateVerificationError{Err: stdErr.New("certificate has expired")}

	reason, fatal := fatalConnectError(errors.Wrap(certErr, "could not connect"))
	if !fatal {
		t.Fatal("an invalid certificate must be fatal: retrying cannot fix it")
	}

	if !strings.Contains(reason, "certificate has expired") {
		t.Errorf("reason %q does not carry the underlying certificate problem", reason)
	}

	if _, fatal := fatalConnectError(stdErr.New("connection refused")); fatal {
		t.Error("an unreachable server must stay retryable")
	}
}

// TestReconnectDelayAfterSession pins the policy that decides how quickly we go
// back: instantly after a session that held, paced after one that did not.
func TestReconnectDelayAfterSession(t *testing.T) {
	h, _ := newTestHandler()

	if got := h.reconnectDelayAfter(time.Hour); got != 0 {
		t.Errorf("delay after a healthy session = %s, want 0 - a netsplit should cost no announces", got)
	}

	if got := h.reconnectDelayAfter(2 * time.Second); got < 15*time.Second {
		t.Errorf("delay after a short session = %s, want at least the 15s floor", got)
	}
}

// TestFlappingStrikesExpire covers the breaker's window: drops spread far apart
// are not flapping, however many of them there are, and stopping the network for
// them would take a working network down.
func TestFlappingStrikesExpire(t *testing.T) {
	h, _ := newTestHandler()

	for range flappingStopThreshold - 1 {
		disconnectAfter(h, time.Second)
	}

	// the strikes so far are old news by the time the next drop happens
	h.m.Lock()
	h.firstShortSession = time.Now().Add(-flappingWindow - time.Minute)
	h.m.Unlock()

	disconnectAfter(h, time.Second)

	if h.Stopped() {
		t.Fatal("drops spread beyond the flapping window must not stop the network")
	}

	h.m.RLock()
	count := h.consecutiveShortSessions
	h.m.RUnlock()

	if count != 1 {
		t.Errorf("expired strikes were not dropped: count = %d, want 1", count)
	}
}

// TestPreRegistrationErrorsCountOncePerAttempt stops one refusal from being
// reported as five: a server may send several ERROR lines before closing, and
// counting each would trip the breaker on a single connection.
func TestPreRegistrationErrorsCountOncePerAttempt(t *testing.T) {
	h, _ := newTestHandler()

	for range flappingStopThreshold * 2 {
		serverError(h, "Trying to reconnect too fast.")
	}

	if h.Stopped() {
		t.Fatal("a single refused attempt must not trip the breaker")
	}

	h.m.RLock()
	count := h.consecutiveShortSessions
	h.m.RUnlock()

	if count != 1 {
		t.Errorf("one attempt counted %d strikes, want 1", count)
	}

	if !hasConnectError(h, "Trying to reconnect too fast") {
		t.Errorf("the server's reason was not surfaced, got %v", h.connectionErrors)
	}
}

// jitterCeiling is the exclusive upper bound nextConnectDelay may return for a
// given base delay.
func jitterCeiling(base time.Duration) time.Duration {
	return base + base/reconnectJitterDivisor
}

// TestConnectBackoffGrowsAndLevelsOff pins the reconnect policy autobrr now owns
// itself: never faster than the floor trackers expect, doubling while a server
// stays unreachable, and levelling off at the cap rather than giving up - a
// network that is merely down must come back on its own once the server does.
func TestConnectBackoffGrowsAndLevelsOff(t *testing.T) {
	h, _ := newTestHandler()

	for attempt, base := range []time.Duration{
		reconnectBaseDelay,
		2 * reconnectBaseDelay,
		4 * reconnectBaseDelay,
		8 * reconnectBaseDelay,
	} {
		got := h.nextConnectDelay()
		if got < base || got >= jitterCeiling(base) {
			t.Errorf("attempt %d: delay %s outside [%s, %s)", attempt+1, got, base, jitterCeiling(base))
		}
	}

	// far past the point where doubling would exceed the cap
	for range 20 {
		h.nextConnectDelay()
	}

	got := h.nextConnectDelay()
	if got < reconnectMaxDelay || got >= jitterCeiling(reconnectMaxDelay) {
		t.Errorf("delay %s did not level off at %s", got, reconnectMaxDelay)
	}
}

// TestConnectBackoffHonoursTheAdvertisedSchedule states the floor and the cap as
// literals rather than in terms of the constants, so that changing a constant
// has to be a deliberate decision about the documented behaviour: 15 seconds is
// the shortest gap trackers tolerate, and 15 minutes keeps an unreachable
// network cheap while still recovering on its own.
func TestConnectBackoffHonoursTheAdvertisedSchedule(t *testing.T) {
	h, _ := newTestHandler()

	first := h.nextConnectDelay()
	if first < 15*time.Second || first >= 18*time.Second {
		t.Errorf("first delay = %s, want 15s plus at most 20%% jitter", first)
	}

	for range 30 {
		h.nextConnectDelay()
	}

	capped := h.nextConnectDelay()
	if capped < 15*time.Minute || capped >= 18*time.Minute {
		t.Errorf("capped delay = %s, want 15m plus at most 20%% jitter", capped)
	}
}

// TestConnectBackoffIsJittered guards the anti-herd property: without jitter
// every autobrr in the world returns to a recovering tracker in the same
// instant, which is invisible in any single instance's behaviour.
func TestConnectBackoffIsJittered(t *testing.T) {
	h, _ := newTestHandler()

	seen := map[time.Duration]bool{}
	for range 40 {
		h.resetConnectBackoff()
		seen[h.nextConnectDelay()] = true
	}

	if len(seen) < 2 {
		t.Errorf("the same delay came back every time (%v); jitter is not being applied", seen)
	}
}

// TestHealthySessionResetsConnectBackoff covers the counterpart: once a
// connection has proven it can hold, a later outage must not inherit the
// previous outage's long delay.
func TestHealthySessionResetsConnectBackoff(t *testing.T) {
	h, _ := newTestHandler()

	for range 5 {
		h.nextConnectDelay()
	}

	h.resetConnectBackoff()

	if got := h.nextConnectDelay(); got >= jitterCeiling(reconnectBaseDelay) {
		t.Errorf("delay after reset = %s, want the base delay %s", got, reconnectBaseDelay)
	}
}

// TestWaitOrStopReturnsWhenStopped makes sure disabling a network takes effect
// immediately instead of after however much backoff was outstanding.
func TestWaitOrStopReturnsWhenStopped(t *testing.T) {
	h, _ := newTestHandler()
	stopSig := make(chan struct{})

	elapsed := make(chan time.Duration, 1)
	go func() {
		start := time.Now()
		h.waitOrStop(time.Hour, stopSig)
		elapsed <- time.Since(start)
	}()

	close(stopSig)

	select {
	case took := <-elapsed:
		if took > 5*time.Second {
			t.Errorf("waitOrStop took %s to notice the stop", took)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("waitOrStop never returned after the network was stopped")
	}
}

// TestStopEndsSupervisor guards against leaking a reconnect supervisor per
// stopped network, and against a stopped network quietly reconnecting itself.
func TestStopEndsSupervisor(t *testing.T) {
	h, _ := newTestHandler()

	sessionEnded := make(chan uint64, 1)
	stopSig := make(chan struct{})

	h.m.Lock()
	h.stopSig = stopSig
	h.m.Unlock()

	done := make(chan struct{})
	go func() {
		h.superviseConnection(sessionEnded, stopSig)
		close(done)
	}()

	h.Stop()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("supervisor still running after Stop")
	}
}

// TestStoppedForIsPerRun covers the Restart race: a supervisor from the previous
// run must give up even though the handler has since been started again, or two
// supervisors would reconnect the same network in parallel.
func TestStoppedForIsPerRun(t *testing.T) {
	h, _ := newTestHandler()

	oldRun := make(chan struct{})
	close(oldRun)

	// the handler itself is live again, as it would be after a Restart
	if h.Stopped() {
		t.Fatal("precondition: handler should be live")
	}

	if !h.stoppedFor(oldRun) {
		t.Error("a supervisor whose run was stopped must not keep reconnecting")
	}

	if h.stoppedFor(make(chan struct{})) {
		t.Error("the current run must not be treated as stopped")
	}
}

// serverError delivers an ERROR line from the server, the way one arrives when
// the link is refused before registration completes.
func serverError(h *Handler, reason string) {
	h.handleServerError(ircmsg.Message{Command: "ERROR", Params: []string{reason}})
}

// refusedAttempt models a whole connection attempt that the server refused
// before registration: a fresh attempt, then its ERROR line. One attempt is one
// strike however many ERROR lines it carries, so the fresh attempt is what makes
// the next line count.
func refusedAttempt(h *Handler, reason string) {
	h.m.Lock()
	h.errorCounted = false
	h.m.Unlock()

	serverError(h, reason)
}

// TestPreRegistrationErrorsTripBreaker covers the blind spot the disconnect
// callback cannot see: irc-go runs it only for connections that finished
// registering, so a server that throttles us before 001 ("Trying to reconnect
// too fast") produces no disconnect event at all. The ERROR line is the only
// signal that the server is refusing us rather than merely being unreachable,
// and its own explanation is the most useful reason to show the user.
func TestPreRegistrationErrorsTripBreaker(t *testing.T) {
	const reason = "Trying to reconnect too fast."

	h, _ := newTestHandler()

	for range flappingStopThreshold - 1 {
		refusedAttempt(h, reason)
		if h.Stopped() {
			t.Fatal("breaker tripped before reaching the threshold")
		}
	}

	refusedAttempt(h, reason)

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
		refusedAttempt(h, "Closing link: (Quit: bye from autobrr)")
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

	refusedAttempt(h, "Trying to reconnect too fast.")
	disconnectAfter(h, time.Second)
	refusedAttempt(h, "Trying to reconnect too fast.")
	disconnectAfter(h, time.Second)

	if h.Stopped() {
		t.Fatal("breaker tripped before reaching the threshold")
	}

	refusedAttempt(h, "Trying to reconnect too fast.")

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

// TestStopDuringConnectDiscardsConnection is the regression test for orphaned
// connections: a Stop that lands while an attempt is mid-Connect cannot quit a
// client it cannot see, so connectOnce itself must notice on the way out and
// discard the session it just registered - otherwise it lingers joined to the
// announce channels, feeding a network the user stopped.
func TestStopDuringConnectDiscardsConnection(t *testing.T) {
	srv := newFakeIRCServer(t)

	h, _ := newTestHandler()
	h.network.Server, h.network.Port = srv.Addr()

	sessionEnded := make(chan uint64, 1)
	stopSig := make(chan struct{})
	claimRun(h, stopSig)

	// the stop lands after the dial succeeded but before registration finishes -
	// the exact window where the client exists but was not yet handed to anyone.
	// The server ignores the stop's QUIT the way a real ircd that has not
	// processed it yet does, so registration still completes underneath the stop.
	srv.ignoreQuit = true
	srv.onRegister = func() { h.Stop() }

	client, err := h.connectOnce(sessionEnded, stopSig)
	if !stdErr.Is(err, clientManuallyDisconnected) {
		t.Fatalf("err = %v, want clientManuallyDisconnected", err)
	}

	if client != nil {
		t.Error("a connection that completed after Stop must not be handed to the supervisor")
	}

	// Two QUITs must reach the server: Stop's own (its client was published
	// pre-Connect, so Stop can see it here) and the discard path's. The second
	// is the one that matters - when Stop instead lands mid-dial its QUIT is
	// dropped by the library (not yet running) and the discard Quit is the only
	// teardown - so waiting for just one would let the discard Quit vanish
	// while Stop's QUIT keeps this test green.
	deadline := time.Now().Add(5 * time.Second)
	for srv.Quits() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("saw %d QUITs, want 2 (Stop's and the discard path's); the discarded connection lingers registered on a stopped network", srv.Quits())
		}
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(100 * time.Millisecond)
	if len(sessionEnded) != 0 {
		t.Error("a discarded connection must not wake the supervisor")
	}
}

// TestConnectOnceRefusesSupersededRun closes the zombie-claim window: an attempt
// from a stopped run that was already past its loop's stop check must not claim
// the generation, or it would supersede the replacement run's live connection
// and gate every one of its callbacks off.
func TestConnectOnceRefusesSupersededRun(t *testing.T) {
	h, _ := newTestHandler()

	newRun := make(chan struct{})
	claimRun(h, newRun)

	h.m.Lock()
	h.clientGen = 41
	h.m.Unlock()

	oldRun := make(chan struct{})
	close(oldRun)

	sessionEnded := make(chan uint64, 1)
	if _, err := h.connectOnce(sessionEnded, oldRun); !stdErr.Is(err, clientManuallyDisconnected) {
		t.Fatalf("err = %v, want clientManuallyDisconnected", err)
	}

	h.m.RLock()
	gen := h.clientGen
	clientSet := h.client != nil
	h.m.RUnlock()

	if gen != 41 {
		t.Errorf("a superseded run claimed generation %d; the replacement's callbacks are now gated off", gen)
	}

	if clientSet {
		t.Error("a superseded run published its client over the replacement's")
	}
}

// TestSessionEndTokenDisplacesStaleToken covers the lost-wakeup case: an attempt
// can complete registration and still fail Connect, parking a token nobody has
// consumed yet. When the live session then ends while that leftover still fills
// the buffer, dropping the new token would strand the supervisor forever - the
// newest token must displace the stale one instead.
func TestSessionEndTokenDisplacesStaleToken(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	h.m.Lock()
	h.clientGen = 7
	h.m.Unlock()

	sessionEnded := make(chan uint64, 1)
	sessionEnded <- 6 // the registered-then-failed attempt's leftover

	h.onSessionEnded(7, ircmsg.Message{Command: "DISCONNECT"}, sessionEnded)

	select {
	case gen := <-sessionEnded:
		if gen != 7 {
			t.Fatalf("token in the buffer = %d, want the live session's 7", gen)
		}
	default:
		t.Fatal("no token buffered at all after the live session ended")
	}
}

// TestStopSupersedesSessionCallbacks pins what makes a stop final: whatever the
// stopped session still delivers - a JOIN confirmation, an auth notice, its own
// disconnect - runs against a generation that no longer exists.
func TestStopSupersedesSessionCallbacks(t *testing.T) {
	h, _ := newTestHandler()

	h.m.Lock()
	h.clientGen = 3
	h.m.Unlock()

	called := 0
	cb := h.gated(3, func(_ ircmsg.Message) { called++ })

	cb(ircmsg.Message{})
	if called != 1 {
		t.Fatal("the current generation's callback must run")
	}

	h.Stop()

	cb(ircmsg.Message{})
	if called != 1 {
		t.Error("a stopped session's callback still ran; its late deliveries poison the next run")
	}
}

// TestSupersededFatalCallbackDoesNotStopNetwork covers the sharpest consequence
// of un-gated callbacks: a superseded attempt still draining can deliver the
// SASL failure of the credentials the user just fixed, and stop the freshly
// restarted network with the old error.
func TestSupersededFatalCallbackDoesNotStopNetwork(t *testing.T) {
	h, _ := newTestHandler()

	h.m.Lock()
	h.clientGen = 2
	h.m.Unlock()

	h.gated(1, func(m ircmsg.Message) { h.handleSASLFail(1, m) })(ircmsg.Message{Command: "904"})

	if h.Stopped() {
		t.Fatal("a superseded attempt's SASL failure stopped the live network")
	}

	if hasConnectError(h, "SASL") {
		t.Error("a superseded attempt's SASL failure was surfaced as the live network's error")
	}

	h.gated(2, func(m ircmsg.Message) { h.handleSASLFail(2, m) })(ircmsg.Message{Command: "904"})

	if !h.Stopped() {
		t.Error("the live session's SASL failure must still stop the network")
	}

	if got := h.stateMachine.GetState(); got != StateError {
		t.Errorf("state after the live session's SASL failure = %s, want Error with the reason on display", got)
	}
}

// TestStaleSessionDeliveriesAfterStopAreDropped pins the wiring end to end: a
// stopped session's socket can stay open for minutes when the server ignores
// our QUIT, and every line it still delivers arrives through the real callback
// chain. A stale SASL failure is the sharpest of them - un-gated it would
// surface the old error and stop whatever the user starts next.
func TestStaleSessionDeliveriesAfterStopAreDropped(t *testing.T) {
	srv := newFakeIRCServer(t)
	srv.ignoreQuit = true

	h, _ := newTestHandler()
	h.network.Server, h.network.Port = srv.Addr()

	sessionEnded := make(chan uint64, 1)
	stopSig := make(chan struct{})
	claimRun(h, stopSig)

	client, err := h.connectOnce(sessionEnded, stopSig)
	if err != nil {
		t.Fatal(err)
	}

	// a sentinel behind the gated handler (callbacks run in registration order)
	// proves the 904 was actually delivered and processed before the assertion,
	// so this cannot pass vacuously on a machine too loaded to deliver in time
	delivered := make(chan struct{})
	client.AddCallback("904", func(_ ircmsg.Message) { close(delivered) })

	h.Stop()

	if srv.Broadcast(":fake.test 904 tester :SASL authentication failed") == 0 {
		t.Fatal("the stopped session's socket was already gone; nothing was tested")
	}

	select {
	case <-delivered:
	case <-time.After(10 * time.Second):
		t.Fatal("the 904 was never delivered to the client")
	}

	if hasConnectError(h, "SASL") {
		t.Error("a stale 904 from the stopped session was surfaced as the network's error")
	}
}

// TestNewClientRefusesMissingProxy pins the no-direct-fallback guard: a network
// set to use a proxy that has none attached must refuse to build rather than
// quietly hand the tracker the user's real IP - and a well-formed proxy must
// still build, or the guard has just bricked every proxied network.
func TestNewClientRefusesMissingProxy(t *testing.T) {
	h, _ := newTestHandler()
	h.network.UseProxy = true
	h.network.Proxy = nil

	if _, err := h.newClient(); err == nil || !strings.Contains(err.Error(), "proxy") {
		t.Fatalf("err = %v, want a refusal naming the missing proxy", err)
	}

	h.network.Proxy = &domain.Proxy{Enabled: true, Addr: "socks5://127.0.0.1:1080"}

	client, err := h.newClient()
	if err != nil {
		t.Fatalf("a well-formed proxy must still build a client: %v", err)
	}

	if client.DialContext == nil {
		t.Error("the proxy dialer was not wired into the client")
	}
}

// TestWiredCallbacksAreAllGated sweeps every registration in wireCallbacks with
// a superseded generation, driving the lines through the client's real dispatch:
// whatever a stale session's socket still delivers, none of it may touch handler
// state. The tests that call gated() directly cannot see the wiring, so without
// this sweep, dropping the gate from a single registration - the announce path,
// a JOIN confirmation - is invisible to both suites.
func TestWiredCallbacksAreAllGated(t *testing.T) {
	h, sseMock := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational
	// the wildcard-nick path (soju) lets the 366 line below match without a live
	// connection tracking our nick
	h.network.UseBouncer = true

	addMonitoredChannel(h, "#announce", "")

	// a channel mid-join, where a JOIN confirmation is a legal transition - the
	// stale 366 below must be dropped by the gate, not by the channel SM
	joining := NewChannel(zerolog.Nop(), h.network.ID, "#joining", true, false, nil)
	joiningSM := NewChannelStateMachine(joining, h, "")
	joining.SetStateMachine(joiningSM)
	joiningSM.state = ChannelStateJoining
	h.channels.Set("#joining", joining)

	h.m.Lock()
	h.clientGen = 7
	h.m.Unlock()

	client := &ircevent.Connection{
		Server:   "unused.test:6667",
		Nick:     "tester",
		User:     "tester",
		RealName: "tester",
		Log:      log.New(io.Discard, "", 0),
	}
	h.wireCallbacks(client, 6) // the superseded attempt's wiring

	baseline := sseMock.eventCount()

	lines := []string{
		":tester!t@h MODE tester :+r",
		":bot!b@h INVITE tester :#invite",
		":tester!t@h PART #announce",
		":ann!a@h PRIVMSG #announce :New Torrent: some.release-GRP",
		":NickServ!s@h NOTICE tester :Password incorrect.",
		":NickServ!s@h NOTICE tester :This nickname is registered and protected",
		":tester!t@h NICK :tester2",
		":op!o@h KICK #announce tester :bye",
		":tester!t@h JOIN #announce",
		":op!o@h TOPIC #announce :new topic",
		":srv 332 tester #announce :topic",
		":srv 366 * #joining :End of /NAMES list",
		":srv 900 tester tester!t@h tester :You are now logged in",
		":srv 903 tester :SASL authentication successful",
		":srv 904 tester :SASL authentication failed",
		":srv 465 tester :You are banned from this server",
		"ERROR :Trying to reconnect too fast.",
		":srv 471 tester #announce :Cannot join channel (+l)",
		":srv 473 tester #announce :Cannot join channel (+i)",
		":srv 474 tester #announce :Cannot join channel (+b)",
		":srv 475 tester #announce :Cannot join channel (+k)",
		":srv 477 tester #announce :You need a registered nick",
		":srv 401 tester NickServ :No such nick",
	}

	for _, line := range lines {
		msg, err := ircmsg.ParseLine(line)
		if err != nil {
			t.Fatalf("bad test line %q: %v", line, err)
		}

		client.HandleMessage(msg)
	}

	if h.Stopped() {
		t.Error("a stale delivery stopped the network")
	}

	if len(h.connectionErrors) != 0 {
		t.Errorf("stale deliveries surfaced connection errors: %v", h.connectionErrors)
	}

	if got := h.stateMachine.GetState(); got != StateFullyOperational {
		t.Errorf("stale deliveries moved the state machine to %s", got)
	}

	if h.authenticated {
		t.Error("a stale login marked the live session authenticated")
	}

	// the SM state, not the Monitoring flag, is what a join confirmation moves
	if ch, _ := h.channels.Get("#joining"); ch.StateMachine().CurrentState() != ChannelStateJoining {
		t.Errorf("a stale JOIN confirmation advanced the channel to %s", ch.StateMachine().CurrentState())
	}

	if got := sseMock.eventCount(); got != baseline {
		t.Errorf("stale deliveries broadcast %d events to the UI", got-baseline)
	}
}

// TestStopDuringOnConnectSleepDoesNotAdvanceStateMachine covers the widest
// TOCTOU hole in the entry-only gate: onConnect sleeps a full second between
// passing the gate and driving the state machine, and a stop or restart fits
// comfortably inside it. Without the post-sleep ownership re-check, the stale
// OnConnected walks the machine to JoiningChannels on a stopped network,
// every channel errors against the quit client, and the next session finds no
// valid transition out - it registers but monitors nothing.
func TestStopDuringOnConnectSleepDoesNotAdvanceStateMachine(t *testing.T) {
	h, _ := newTestHandler()

	h.m.Lock()
	h.clientGen = 5
	h.m.Unlock()

	done := make(chan struct{})
	go func() {
		h.onConnect(5, ircmsg.Message{})
		close(done)
	}()

	// land the stop well inside onConnect's one-second sleep
	time.Sleep(300 * time.Millisecond)
	h.Stop()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("onConnect never returned")
	}

	// OnConnected does not stop at Connected: with no auth configured it chains
	// through Authenticated into JoiningChannels, so pin the only state a
	// stopped network may show
	if got := h.stateMachine.GetState(); got != StateDisconnected {
		t.Errorf("a stop during onConnect's sleep still advanced the state machine to %s", got)
	}
}

// TestManualStopParksStateMachineDisconnected: the disconnect callback used to
// drive the state machine on a stop, and it is superseded before it can run, so
// the stop must do it itself - or a stopped network keeps reporting
// FullyOperational to the UI.
func TestManualStopParksStateMachineDisconnected(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateFullyOperational

	h.m.Lock()
	h.client = &ircevent.Connection{}
	h.m.Unlock()

	h.Stop()

	if got := h.stateMachine.GetState(); got != StateDisconnected {
		t.Errorf("state after a manual stop = %s, want Disconnected", got)
	}
}

// TestStopPreservesErrorState: the fatal paths (a ban, bad credentials, the
// breaker) park the machine in Error with the reason on display and then stop
// the network; the stop must not paper over that with Disconnected.
func TestStopPreservesErrorState(t *testing.T) {
	h, _ := newTestHandler()
	h.stateMachine.currentState = StateError

	h.m.Lock()
	h.client = &ircevent.Connection{}
	h.m.Unlock()

	h.Stop()

	if got := h.stateMachine.GetState(); got != StateError {
		t.Errorf("state after a fatal stop = %s, want Error kept on display", got)
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
