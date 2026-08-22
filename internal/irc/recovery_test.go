// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
)

// newIdleSM builds a fresh Idle channel state machine (not yet monitoring).
func newIdleSM(h *Handler, name, invite string) *ChannelStateMachine {
	ch := NewChannel(zerolog.Nop(), h.network.ID, name, true, false, nil)
	sm := NewChannelStateMachine(ch, h, invite)
	ch.SetStateMachine(sm)
	h.channels.Set(ch.Name, ch)
	return sm
}

func waitForState(sm *ChannelStateMachine, want ChannelState, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if sm.CurrentState() == want {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return sm.CurrentState() == want
}

func waitFor(cond func() bool, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(2 * time.Millisecond)
	}
	return cond()
}

// TestChannelStateTransitions_Recoverable pins down that the Error and Kicked
// states are no longer terminal: a retry or a successful join can leave them.
func TestChannelStateTransitions_Recoverable(t *testing.T) {
	cases := []struct{ from, to ChannelState }{
		{ChannelStateIdle, ChannelStateMonitoring},
		{ChannelStateError, ChannelStateJoining},
		{ChannelStateError, ChannelStateAwaitingInvite},
		{ChannelStateError, ChannelStateMonitoring},
		{ChannelStateKicked, ChannelStateJoining},
		{ChannelStateKicked, ChannelStateAwaitingInvite},
		{ChannelStateKicked, ChannelStateMonitoring},
	}
	sm := &ChannelStateMachine{}
	for _, c := range cases {
		if !sm.isValidTransition(c.from, c.to) {
			t.Errorf("expected %s -> %s to be valid (recoverable), but it is not", c.from, c.to)
		}
	}
}

// TestOnInviteRecoversFromError verifies an invite received while a channel is
// stuck in Error pulls it back into the join workflow.
func TestOnInviteRecoversFromError(t *testing.T) {
	h, sse := newTestHandler()
	sm := newIdleSM(h, "#announce", "invitebot !invite u p")
	sm.state = ChannelStateError

	sm.OnInvite("invitebot")

	if !waitFor(func() bool { return sse.hasStateEvent("#announce", "Joining") }, time.Second) {
		t.Fatalf("OnInvite from Error did not restart the join workflow, state=%s", sm.CurrentState())
	}
}

// TestForceJoinBeforeStart covers servers (e.g. InspIRCd trackers) that
// force-join the bot on connect: the join confirms while the channel is still
// Idle, and the later Start() must not re-send a JOIN the server would
// silently drop.
func TestForceJoinBeforeStart(t *testing.T) {
	h, _ := newTestHandler()
	sm := newIdleSM(h, "#announce", "")

	sm.OnJoinSuccess()

	if !waitForState(sm, ChannelStateMonitoring, time.Second) {
		t.Fatalf("force-join while Idle did not confirm the channel, state=%s", sm.CurrentState())
	}

	sm.Start()

	if got := sm.CurrentState(); got != ChannelStateMonitoring {
		t.Fatalf("Start() after a force-join disturbed the channel, state=%s", got)
	}
}

// TestOnInviteIgnoredWhenMonitoring ensures a stray invite for a channel we are
// already monitoring does not disturb it.
func TestOnInviteIgnoredWhenMonitoring(t *testing.T) {
	h, sse := newTestHandler()
	sm := addMonitoredChannel(h, "#announce", "invitebot !invite u p")

	sm.OnInvite("invitebot")

	time.Sleep(50 * time.Millisecond)
	if sse.hasStateEvent("#announce", "Joining") {
		t.Fatal("OnInvite while Monitoring should be ignored")
	}
	if got := sm.CurrentState(); got != ChannelStateMonitoring {
		t.Fatalf("state changed to %s on stray invite while Monitoring", got)
	}
}

// TestOnInviteTimeout covers the AwaitingInvite timeout that rescues a channel
// from a silent invite bot.
func TestOnInviteTimeout(t *testing.T) {
	t.Run("retries when still awaiting the same attempt", func(t *testing.T) {
		h, _ := newTestHandler()
		sm := newIdleSM(h, "#c", "invitebot !invite u p")
		sm.state = ChannelStateAwaitingInvite
		sm.inviteGen = 7

		sm.onInviteTimeout(7)

		if !waitForState(sm, ChannelStateAwaitingInviteBot, time.Second) {
			t.Fatalf("timeout did not route into backoff, state=%s", sm.CurrentState())
		}
	})

	t.Run("no-op on a stale attempt", func(t *testing.T) {
		h, _ := newTestHandler()
		sm := newIdleSM(h, "#c", "invitebot !invite u p")
		sm.state = ChannelStateAwaitingInvite
		sm.inviteGen = 7

		sm.onInviteTimeout(6) // a newer attempt already superseded this one

		if got := sm.CurrentState(); got != ChannelStateAwaitingInvite {
			t.Fatalf("stale timeout changed state to %s", got)
		}
	})

	t.Run("no-op when no longer awaiting", func(t *testing.T) {
		h, _ := newTestHandler()
		sm := newIdleSM(h, "#c", "invitebot !invite u p")
		sm.state = ChannelStateMonitoring
		sm.inviteGen = 7

		sm.onInviteTimeout(7)

		if got := sm.CurrentState(); got != ChannelStateMonitoring {
			t.Fatalf("timeout fired while Monitoring, state=%s", got)
		}
	})
}

// TestOnJoinTimeout covers the Joining timeout that rescues a channel whose
// JOIN is silently rejected (banned/full/bad key) or never confirmed.
func TestOnJoinTimeout(t *testing.T) {
	t.Run("errors when join is not confirmed", func(t *testing.T) {
		h, sse := newTestHandler()
		sm := newIdleSM(h, "#c", "")
		sm.state = ChannelStateJoining
		sm.joinGen = 3

		sm.onJoinTimeout(3)

		if !waitFor(func() bool { return sse.hasStateEvent("#c", "Error") }, time.Second) {
			t.Fatalf("join timeout did not fail the channel, state=%s", sm.CurrentState())
		}
	})

	t.Run("no-op on a stale attempt", func(t *testing.T) {
		h, sse := newTestHandler()
		sm := newIdleSM(h, "#c", "")
		sm.state = ChannelStateJoining
		sm.joinGen = 3

		sm.onJoinTimeout(2) // a newer JOIN superseded this one

		time.Sleep(20 * time.Millisecond)
		if sse.hasStateEvent("#c", "Error") {
			t.Fatal("a stale join timeout should not fail the channel")
		}
	})

	t.Run("no-op after a successful join", func(t *testing.T) {
		h, sse := newTestHandler()
		sm := newIdleSM(h, "#c", "")
		sm.state = ChannelStateMonitoring
		sm.joinGen = 3

		sm.onJoinTimeout(3)

		time.Sleep(20 * time.Millisecond)
		if sse.hasStateEvent("#c", "Error") {
			t.Fatal("join timeout fired after the join was confirmed")
		}
	})
}

// TestKickDoesNotAutoRejoin verifies a kicked channel stays Kicked and never
// re-joins on its own — proactively re-joining a channel that kicked us risks a
// ban. The channel only recovers on reconnect, restart, or an explicit invite.
func TestKickDoesNotAutoRejoin(t *testing.T) {
	h, sse := newTestHandler()
	sm := addMonitoredChannel(h, "#announce", "") // direct-join channel

	sm.OnKicked("me", "operator", "spam")

	if !sse.hasStateEvent("#announce", "Kicked") {
		t.Fatal("kick did not broadcast the Kicked state")
	}
	if sm.channel.Monitoring {
		t.Error("channel should not be monitoring after a kick")
	}

	// give any (unwanted) rejoin machinery ample time to fire, then assert we
	// are still parked in Kicked and never attempted a JOIN.
	time.Sleep(100 * time.Millisecond)
	if got := sm.CurrentState(); got != ChannelStateKicked {
		t.Fatalf("kicked channel left Kicked on its own (state=%s) — it must not auto-rejoin", got)
	}
	if sse.hasStateEvent("#announce", "Joining") || sse.hasStateEvent("#announce", "AwaitingInvite") {
		t.Fatal("kicked channel attempted to rejoin on its own")
	}
}

// TestKickRecoversOnExplicitInvite verifies that being invited back after a kick
// (an explicit welcome, not our own initiative) does resume the join workflow.
func TestKickRecoversOnExplicitInvite(t *testing.T) {
	h, sse := newTestHandler()
	sm := addMonitoredChannel(h, "#announce", "")

	sm.OnKicked("me", "operator", "spam")
	if !waitForState(sm, ChannelStateKicked, time.Second) {
		t.Fatalf("expected Kicked, got %s", sm.CurrentState())
	}

	sm.OnInvite("someop")

	if !waitFor(func() bool { return sse.hasStateEvent("#announce", "Joining") }, time.Second) {
		t.Fatalf("an explicit invite after a kick should resume joining, state=%s", sm.CurrentState())
	}
}

// TestErrorAutoRetries verifies a channel in Error schedules a recovery attempt.
func TestErrorAutoRetries(t *testing.T) {
	h, sse := newTestHandler()
	sm := newIdleSM(h, "#announce", "") // direct-join channel
	sm.errorBaseDelay = 15 * time.Millisecond

	sm.enterError("boom")

	if !sse.hasStateEvent("#announce", "Error") {
		t.Fatal("enterError did not broadcast the Error state")
	}
	if !waitFor(func() bool { return sse.hasStateEvent("#announce", "Joining") }, time.Second) {
		t.Fatal("Error state did not auto-retry the join workflow")
	}
}

// TestErrorRetryGivesUp verifies recovery stops after maxErrorRetries.
func TestErrorRetryGivesUp(t *testing.T) {
	h, sse := newTestHandler()
	sm := newIdleSM(h, "#announce", "")
	sm.state = ChannelStateError

	sm.scheduleErrorRetry(maxErrorRetries + 1)

	time.Sleep(20 * time.Millisecond)
	if sse.hasStateEvent("#announce", "Joining") {
		t.Fatal("error recovery should have given up, but it retried")
	}
}

// TestErrorRetryStaleGeneration verifies only the most recent error drives recovery.
func TestErrorRetryStaleGeneration(t *testing.T) {
	h, sse := newTestHandler()
	sm := newIdleSM(h, "#announce", "")
	sm.state = ChannelStateError
	sm.errorAttempts = 2 // a newer error has occurred since attempt 1 was scheduled
	sm.errorBaseDelay = 5 * time.Millisecond

	sm.scheduleErrorRetry(1) // the stale attempt

	if sse.hasStateEvent("#announce", "Joining") {
		t.Fatal("a stale error retry attempted recovery")
	}
}

// TestHandleMonitoringResetsCounters verifies a successful join clears the
// retry/backoff bookkeeping so a later failure starts from a clean slate.
func TestHandleMonitoringResetsCounters(t *testing.T) {
	h, _ := newTestHandler()
	sm := addMonitoredChannel(h, "#announce", "")
	sm.authAttempts = 5
	sm.errorAttempts = 2

	sm.handleMonitoring()

	sm.m.RLock()
	defer sm.m.RUnlock()
	if sm.authAttempts != 0 || sm.errorAttempts != 0 {
		t.Fatalf("handleMonitoring did not reset counters: auth=%d err=%d",
			sm.authAttempts, sm.errorAttempts)
	}
}

// TestResetClearsRecoveryCounters verifies a disconnect wipes the recovery
// bookkeeping and invalidates any pending invite timeout.
func TestResetClearsRecoveryCounters(t *testing.T) {
	h, _ := newTestHandler()
	sm := addMonitoredChannel(h, "#announce", "")
	sm.errorAttempts = 2
	sm.inviteGen = 4
	sm.joinGen = 6

	sm.Reset()

	sm.m.RLock()
	defer sm.m.RUnlock()
	if sm.errorAttempts != 0 {
		t.Fatalf("Reset did not clear errorAttempts: err=%d", sm.errorAttempts)
	}
	if sm.inviteGen != 5 {
		t.Errorf("Reset should bump inviteGen (4 -> 5) to void pending timeouts, got %d", sm.inviteGen)
	}
	if sm.joinGen != 7 {
		t.Errorf("Reset should bump joinGen (6 -> 7) to void pending timeouts, got %d", sm.joinGen)
	}
}

// TestRetryBackoffSchedule pins the documented invite backoff, including the
// second phase that previously (incorrectly) waited 15s instead of 30s.
func TestRetryBackoffSchedule(t *testing.T) {
	cases := []struct {
		attempt int
		want    time.Duration
	}{
		{1, 15 * time.Second},
		{8, 15 * time.Second},
		{9, 30 * time.Second},  // phase 2 starts here
		{68, 30 * time.Second}, // phase 2 ends
		{69, time.Minute},      // phase 3
		{128, time.Minute},
		{129, time.Hour}, // phase 4
	}
	for _, c := range cases {
		if d, ok := retryBackoff(c.attempt); !ok || d != c.want {
			t.Errorf("retryBackoff(%d) = %s,%v; want %s,true", c.attempt, d, ok, c.want)
		}
	}
	if _, ok := retryBackoff(1_000_000); ok {
		t.Error("retryBackoff should give up for very large attempt counts")
	}
}
