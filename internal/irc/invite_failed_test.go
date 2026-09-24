// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testInviteGrace is a short invite-response grace so parking tests do not wait
// the production default. A bot response that is not followed by a JOIN within
// this window is treated as a rejection.
const testInviteGrace = 20 * time.Millisecond

// addAwaitingInviteChannel registers a channel that has sent its invite command
// and is waiting for the bot to respond. The channel's invite command is set
// directly (rather than via SetInviteCommand) so the helper does not trigger the
// Joining transition that a config change would. The invite-response grace is
// shortened so tests resolve quickly.
func addAwaitingInviteChannel(h *Handler, name, inviteCommand string) *ChannelStateMachine {
	ch := NewChannel(zerolog.Nop(), h.network.ID, name, true, false, nil)
	ch.inviteCommand = inviteCommand
	sm := NewChannelStateMachine(ch, h, inviteCommand)
	sm.inviteResponseGrace = testInviteGrace
	ch.SetStateMachine(sm)
	h.channels.Set(ch.Name, ch)

	sm.m.Lock()
	sm.state = ChannelStateAwaitingInvite
	sm.m.Unlock()

	return sm
}

// TestInviteBotResponseParksAfterGrace verifies that a bot response NOT followed
// by a JOIN is, after the grace window, treated as a rejection: the bot's reason
// is surfaced on the channel and broadcast as an InviteFailed state.
func TestInviteBotResponseParksAfterGrace(t *testing.T) {
	h, sse := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	const reason = "invite rejected by Voyager: invalid IRC key"
	sm.OnInviteBotResponse(reason)

	require.Truef(t, waitForState(sm, ChannelStateInviteFailed, time.Second), "a bot response with no join should park in InviteFailed, got %s", sm.CurrentState())
	require.True(t, waitFor(func() bool { return sm.channel.HasConnectionErrors() }, time.Second), "parking should surface a connection error")
	errs := sm.channel.ConnectionErrorsCopy()
	require.NotEmpty(t, errs)
	require.Contains(t, errs[0], "invalid IRC key")
	require.True(t, waitFor(func() bool { return stateEventHasError(sse, "#chan", "InviteFailed", "invalid IRC key") }, time.Second), "parking should broadcast InviteFailed carrying the reason")
}

// TestInviteBotPositiveAckThenJoinMonitors is the regression test for the PTP /
// Hummingbird flow: the invite bot answers positively ("attempting to join you")
// and the server then force-joins us. The positive NOTICE must NOT park the
// channel; the following JOIN moves it straight to Monitoring, and the grace timer
// must not later flip a monitoring channel to InviteFailed.
func TestInviteBotPositiveAckThenJoinMonitors(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "hummingbird enter user key #chan")

	// the bot acknowledges (this used to immediately park the channel)
	sm.OnInviteBotResponse("invite rejected by Hummingbird: Attempting to join you to #chan")

	// ...and the server force-joins us before the grace elapses
	sm.OnJoinSuccess()

	require.Truef(t, waitForState(sm, ChannelStateMonitoring, time.Second), "a positive ack followed by a join should monitor, got %s", sm.CurrentState())

	// let the grace timer fire; it must find the channel monitoring and no-op
	time.Sleep(3 * testInviteGrace)

	require.Equal(t, ChannelStateMonitoring, sm.CurrentState(), "grace timer flipped a monitoring channel")
	require.Falsef(t, sm.channel.HasConnectionErrors(), "a successfully joined channel must not carry an invite error: %v", sm.channel.ConnectionErrorsCopy())
}

// TestInviteBotResponseIgnoredWhenNotAwaiting verifies OnInviteBotResponse only
// acts while the channel is actively awaiting an invite, so a stray bot message
// cannot error a channel that is idle or already monitoring.
func TestInviteBotResponseIgnoredWhenNotAwaiting(t *testing.T) {
	h, _ := newTestHandler()
	sm := newIdleSM(h, "#chan", "voyager autobot user key") // Idle
	sm.inviteResponseGrace = testInviteGrace

	sm.OnInviteBotResponse("stray message")

	// give any (erroneously armed) grace timer time to fire
	time.Sleep(3 * testInviteGrace)

	require.False(t, sm.channel.HasConnectionErrors(), "a bot response from Idle should not surface an error")
	require.Equal(t, ChannelStateIdle, sm.CurrentState(), "a bot response from Idle changed state")
}

// TestInviteFailedParksWithoutRetry verifies a rejected invite stops instead of
// spinning: the channel settles in InviteFailed, does NOT fall into the invite
// backoff loop, and keeps its surfaced error visible.
func TestInviteFailedParksWithoutRetry(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.OnInviteBotResponse("invite rejected by Voyager: invalid IRC key")

	require.Truef(t, waitForState(sm, ChannelStateInviteFailed, time.Second), "expected channel to park in InviteFailed, got %s", sm.CurrentState())
	// a rejection must NOT enter the retry backoff (that path is only for an absent
	// bot); the channel stays parked until an invite or a config change
	require.False(t, waitForState(sm, ChannelStateAwaitingInviteBot, 150*time.Millisecond), "rejected invite must not auto-retry via AwaitingInviteBot")
	require.Equal(t, ChannelStateInviteFailed, sm.CurrentState(), "channel should remain parked in InviteFailed")
	require.True(t, sm.channel.HasConnectionErrors(), "parked InviteFailed channel should keep its surfaced error")
}

// TestInviteFailedRecoversOnInvite verifies a late INVITE still joins the channel
// after a rejection parked it (the failure is not terminal).
func TestInviteFailedRecoversOnInvite(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.OnInviteBotResponse("invite rejected by Voyager: invalid IRC key")
	require.Truef(t, waitForState(sm, ChannelStateInviteFailed, time.Second), "expected InviteFailed before recovery, got %s", sm.CurrentState())

	// the bot invites us after all - the parked channel must move toward Joining
	sm.OnInvite("Voyager")

	require.Truef(t, waitForState(sm, ChannelStateJoining, time.Second), "late INVITE after a rejection should join; state = %s", sm.CurrentState())
}

// TestInviteFailedRecoversOnLateForceJoin verifies that a JOIN arriving after the
// channel already parked in InviteFailed (a slow force-join bot) still recovers to
// Monitoring instead of being dropped as an invalid transition.
func TestInviteFailedRecoversOnLateForceJoin(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "hummingbird enter user key #chan")

	sm.OnInviteBotResponse("invite rejected by Hummingbird: Attempting to join you to #chan")
	require.Truef(t, waitForState(sm, ChannelStateInviteFailed, time.Second), "expected InviteFailed before the late join, got %s", sm.CurrentState())

	sm.OnJoinSuccess()

	require.Truef(t, waitForState(sm, ChannelStateMonitoring, time.Second), "a late force-join should recover to Monitoring, got %s", sm.CurrentState())
	// handleMonitoring (which clears the error) runs in the onStateEntry goroutine
	require.Truef(t, waitFor(func() bool { return !sm.channel.HasConnectionErrors() }, time.Second), "recovery to Monitoring should clear the invite error: %v", sm.channel.ConnectionErrorsCopy())
}

// TestNoSuchNickKeepsRetrying is the counterpart to the parking behaviour: an
// absent invite bot (no-such-nick) keeps retrying via backoff instead of parking,
// because the bot may simply not be connected yet.
func TestNoSuchNickKeepsRetrying(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.OnNoSuchNick("voyager")

	require.Truef(t, waitForState(sm, ChannelStateAwaitingInviteBot, time.Second), "no-such-nick should route into the retry backoff, got %s", sm.CurrentState())
}

// TestHandleInviteResponseRoutesBotDM verifies a direct NOTICE from the invite bot
// is routed to the awaiting channel and, absent a join, surfaces as a failure
// carrying the bot's text after the grace window.
func TestHandleInviteResponseRoutesBotDM(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	msg := ircmsg.Message{
		Source:  "Voyager!bot@irc.example.test",
		Command: "NOTICE",
		Params:  []string{"autobrr", "invalid IRC key"}, // addressed to us, not a channel
	}
	h.handleInviteResponse(msg)

	require.True(t, waitFor(func() bool { return sm.channel.HasConnectionErrors() }, time.Second), "bot DM with no follow-up join should surface an invite failure on the awaiting channel")
	errs := sm.channel.ConnectionErrorsCopy()
	require.NotEmpty(t, errs)
	require.Contains(t, errs[0], "invalid IRC key")
}

// TestHandleInviteResponseIgnoresChannelMessage is the false-positive guard: a bot
// that is both the announcer and the invite bot sends a channel message; it must
// not be mistaken for an invite reply and error a sibling channel.
func TestHandleInviteResponseIgnoresChannelMessage(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	msg := ircmsg.Message{
		Source:  "Voyager!bot@irc.example.test",
		Command: "NOTICE",
		Params:  []string{"#announce", "some announce line"}, // to a channel, not us
	}
	h.handleInviteResponse(msg)

	time.Sleep(3 * testInviteGrace)

	require.Falsef(t, sm.channel.HasConnectionErrors(), "channel message from the bot must not raise an invite failure: %v", sm.channel.ConnectionErrorsCopy())
}

// TestHandleInviteResponseIgnoresUnrelatedNick verifies a DM from a nick that is
// not the channel's invite bot is left alone.
func TestHandleInviteResponseIgnoresUnrelatedNick(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	msg := ircmsg.Message{
		Source:  "SomeUser!u@host",
		Command: "PRIVMSG",
		Params:  []string{"autobrr", "hey there"},
	}
	h.handleInviteResponse(msg)

	time.Sleep(3 * testInviteGrace)

	require.Falsef(t, sm.channel.HasConnectionErrors(), "DM from an unrelated nick must not raise an invite failure: %v", sm.channel.ConnectionErrorsCopy())
}

// TestHandleInviteResponseIgnoresNonAwaitingChannel verifies a channel that is not
// awaiting an invite (e.g. already monitoring) is not touched by a bot DM.
func TestHandleInviteResponseIgnoresNonAwaitingChannel(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")
	sm.m.Lock()
	sm.state = ChannelStateMonitoring
	sm.m.Unlock()

	msg := ircmsg.Message{
		Source:  "Voyager!bot@irc.example.test",
		Command: "NOTICE",
		Params:  []string{"autobrr", "invalid IRC key"},
	}
	h.handleInviteResponse(msg)

	time.Sleep(3 * testInviteGrace)

	require.Falsef(t, sm.channel.HasConnectionErrors(), "a monitoring channel must not be errored by a bot DM: %v", sm.channel.ConnectionErrorsCopy())
}

// TestHandleInviteResponseIgnoresPrefixOverlapNick guards the exact-nick match:
// a message from a different bot whose nick merely prefixes the real invite bot's
// nick must not park the channel (the real bot is absent and should keep retrying).
func TestHandleInviteResponseIgnoresPrefixOverlapNick(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	msg := ircmsg.Message{
		Source:  "voy!bot@host", // "voy" is a prefix of "voyager" but a different nick
		Command: "NOTICE",
		Params:  []string{"autobrr", "some unrelated message"},
	}
	h.handleInviteResponse(msg)

	time.Sleep(3 * testInviteGrace)

	require.Falsef(t, sm.channel.HasConnectionErrors(), "a prefix-overlapping unrelated nick must not park the channel: %v", sm.channel.ConnectionErrorsCopy())
	require.Equal(t, ChannelStateAwaitingInvite, sm.CurrentState(), "channel should still be AwaitingInvite")
}

// TestInviteFailedNeutralizesStaleTimeout verifies the parking is atomic w.r.t.
// the silent invite timeout: once parked (via the grace path), the invite timeout
// armed for that attempt cannot drag the channel back into the retry loop
// (parking bumps inviteGen).
func TestInviteFailedNeutralizesStaleTimeout(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.m.Lock()
	gen := sm.inviteGen
	sm.m.Unlock()

	sm.OnInviteBotResponse("invite rejected by Voyager: invalid IRC key")
	require.Truef(t, waitForState(sm, ChannelStateInviteFailed, time.Second), "expected InviteFailed, got %s", sm.CurrentState())

	// the silent invite timeout that was armed for the now-parked attempt must no-op
	sm.onInviteTimeout(gen)

	require.Equal(t, ChannelStateInviteFailed, sm.CurrentState(), "a stale invite timeout un-parked the channel")
}

// TestAddChannelUnparksInviteFailedChannel verifies a live reconcile that re-adds
// an already-tracked channel parked in InviteFailed re-drives it through the join
// workflow (AddChannel must Reset before Start, else Start's transition from the
// sticky InviteFailed state is dropped as invalid and the channel stays stuck).
func TestAddChannelUnparksInviteFailedChannel(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.OnInviteBotResponse("invite rejected by Voyager: invalid IRC key")
	require.Truef(t, waitForState(sm, ChannelStateInviteFailed, time.Second), "expected InviteFailed before re-add, got %s", sm.CurrentState())

	h.AddChannel(domain.IrcChannel{Name: "#chan", Enabled: true})

	require.Truef(t, waitFor(func() bool { return sm.CurrentState() != ChannelStateInviteFailed }, time.Second), "AddChannel should un-park an InviteFailed channel; still %s", sm.CurrentState())
}

// TestInviteTimeoutDefersToGraceAfterBotResponse verifies the grace-vs-timeout
// window fix: if the bot has answered (inviteResponded) but the silent invite
// timeout for the attempt fires first - which happens when the bot replies in the
// last inviteResponseGrace seconds of the timeout window - the timeout must DEFER
// to the grace timer. A present, rejecting bot must not be routed into the
// absent-bot retry loop (which would re-send the invite and drop its reason).
func TestInviteTimeoutDefersToGraceAfterBotResponse(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	// OnInviteBotResponse records the response (inviteResponded) and arms the grace;
	// set the flag directly so no real grace timer races this deterministic check.
	sm.m.Lock()
	gen := sm.inviteGen
	sm.inviteResponded = true
	sm.m.Unlock()

	// the silent invite timeout for this attempt fires - it must defer to the grace
	sm.onInviteTimeout(gen)

	require.Equal(t, ChannelStateAwaitingInvite, sm.CurrentState(), "a responded-to invite must not enter the absent-bot retry loop")
}

// TestNetworkRecoversFromErrorWhenChannelMonitors verifies the connection state
// machine can climb out of StateError WITHOUT a reconnect once a channel that had
// failed reaches Monitoring (e.g. a late invite-failure recovery: a force-join
// arriving after the grace already parked the channel). This exercises both the
// StateError->operational transitions and the re-notification from handleMonitoring
// that lets the network re-evaluate on the settled (post-monitoring) snapshot.
func TestNetworkRecoversFromErrorWhenChannelMonitors(t *testing.T) {
	h, _ := newTestHandler()

	ch := NewChannel(zerolog.Nop(), h.network.ID, "#chan", true, false, nil)
	sm := NewChannelStateMachine(ch, h, "")
	ch.SetStateMachine(sm)
	ch.SetConnectionError("could not join")
	h.channels.Set(ch.Name, ch)

	// network latched into Error (all enabled channels had failed)
	h.stateMachine.m.Lock()
	h.stateMachine.currentState = StateError
	h.stateMachine.m.Unlock()

	// the channel recovers: Joining -> Monitoring drives handleMonitoring, which
	// clears the error, sets the flag, and re-notifies the connection SM
	sm.m.Lock()
	sm.state = ChannelStateJoining
	sm.m.Unlock()
	sm.OnJoinSuccess()

	require.Truef(t, waitFor(func() bool { return h.stateMachine.GetState() == StateFullyOperational }, time.Second), "network should recover from Error once the channel monitors; got %s", h.stateMachine.GetState())
}

// TestIsChannelTarget covers the channel-vs-DM classifier used to keep channel
// traffic out of the invite-failure path.
func TestIsChannelTarget(t *testing.T) {
	cases := []struct {
		target string
		want   bool
	}{
		{"#announce", true},
		{"&local", true},
		{"!12345chan", true},
		{"+modeless", true},
		{"autobrr", false},
		{"Voyager", false},
		{"", false},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, isChannelTarget(c.target), "isChannelTarget(%q)", c.target)
	}
}
