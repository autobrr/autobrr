// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"strings"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/ergochat/irc-go/ircmsg"
	"github.com/rs/zerolog"
)

// addAwaitingInviteChannel registers a channel that has sent its invite command
// and is waiting for the bot to respond. The channel's invite command is set
// directly (rather than via SetInviteCommand) so the helper does not trigger the
// Joining transition that a config change would.
func addAwaitingInviteChannel(h *Handler, name, inviteCommand string) *ChannelStateMachine {
	ch := NewChannel(zerolog.Nop(), h.network.ID, name, true, nil)
	ch.inviteCommand = inviteCommand
	sm := NewChannelStateMachine(ch, h, inviteCommand)
	ch.SetStateMachine(sm)
	h.channels.Set(ch.Name, ch)

	sm.m.Lock()
	sm.state = ChannelStateAwaitingInvite
	sm.m.Unlock()

	return sm
}

// TestOnInviteFailedSurfacesReason verifies that a rejected invite surfaces the
// bot's reason on the channel and broadcasts it as an InviteFailed state.
func TestOnInviteFailedSurfacesReason(t *testing.T) {
	h, sse := newTestHandler()
	sm := newIdleSM(h, "#chan", "voyager autobot user key")

	sm.m.Lock()
	sm.state = ChannelStateAwaitingInvite
	sm.m.Unlock()

	const reason = "invite rejected by Voyager: invalid IRC key"
	sm.OnInviteFailed(reason)

	if !waitFor(func() bool { return sm.channel.HasConnectionErrors() }, time.Second) {
		t.Fatal("OnInviteFailed should surface a connection error")
	}
	if errs := sm.channel.ConnectionErrorsCopy(); len(errs) == 0 || !strings.Contains(errs[0], "invalid IRC key") {
		t.Fatalf("connection errors = %v, want one containing the bot reason", errs)
	}
	if !waitFor(func() bool { return stateEventHasError(sse, "#chan", "InviteFailed", "invalid IRC key") }, time.Second) {
		t.Fatal("OnInviteFailed should broadcast InviteFailed carrying the reason")
	}
}

// TestOnInviteFailedIgnoredWhenNotAwaiting verifies OnInviteFailed only acts while
// the channel is actively awaiting an invite, so a stray bot message cannot error
// a channel that is idle or already monitoring.
func TestOnInviteFailedIgnoredWhenNotAwaiting(t *testing.T) {
	h, _ := newTestHandler()
	sm := newIdleSM(h, "#chan", "voyager autobot user key") // Idle

	sm.OnInviteFailed("stray message")

	if sm.channel.HasConnectionErrors() {
		t.Fatal("OnInviteFailed from Idle should not surface an error")
	}
	if got := sm.CurrentState(); got != ChannelStateIdle {
		t.Fatalf("OnInviteFailed from Idle changed state to %s", got)
	}
}

// TestInviteFailedParksWithoutRetry verifies a rejected invite stops instead of
// spinning: the channel settles in InviteFailed, does NOT fall into the invite
// backoff loop, and keeps its surfaced error visible.
func TestInviteFailedParksWithoutRetry(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.OnInviteFailed("invite rejected by Voyager: invalid IRC key")

	if !waitForState(sm, ChannelStateInviteFailed, time.Second) {
		t.Fatalf("expected channel to park in InviteFailed, got %s", sm.CurrentState())
	}
	// a rejection must NOT enter the retry backoff (that path is only for an absent
	// bot); the channel stays parked until an invite or a config change
	if waitForState(sm, ChannelStateAwaitingInviteBot, 150*time.Millisecond) {
		t.Fatal("rejected invite must not auto-retry via AwaitingInviteBot")
	}
	if got := sm.CurrentState(); got != ChannelStateInviteFailed {
		t.Fatalf("channel should remain parked in InviteFailed, got %s", got)
	}
	if !sm.channel.HasConnectionErrors() {
		t.Fatal("parked InviteFailed channel should keep its surfaced error")
	}
}

// TestInviteFailedRecoversOnInvite verifies a late INVITE still joins the channel
// after a rejection parked it (the failure is not terminal).
func TestInviteFailedRecoversOnInvite(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.OnInviteFailed("invite rejected by Voyager: invalid IRC key")
	if !waitForState(sm, ChannelStateInviteFailed, time.Second) {
		t.Fatalf("expected InviteFailed before recovery, got %s", sm.CurrentState())
	}

	// the bot invites us after all - the parked channel must move toward Joining
	sm.OnInvite("Voyager")

	if !waitForState(sm, ChannelStateJoining, time.Second) {
		t.Fatalf("late INVITE after a rejection should join; state = %s", sm.CurrentState())
	}
}

// TestNoSuchNickKeepsRetrying is the counterpart to the parking behaviour: an
// absent invite bot (no-such-nick) keeps retrying via backoff instead of parking,
// because the bot may simply not be connected yet.
func TestNoSuchNickKeepsRetrying(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.OnNoSuchNick("voyager")

	if !waitForState(sm, ChannelStateAwaitingInviteBot, time.Second) {
		t.Fatalf("no-such-nick should route into the retry backoff, got %s", sm.CurrentState())
	}
}

// TestHandleInviteResponseRoutesBotDM verifies a direct NOTICE from the invite bot
// is routed to the awaiting channel as a failure carrying the bot's text.
func TestHandleInviteResponseRoutesBotDM(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	msg := ircmsg.Message{
		Source:  "Voyager!bot@irc.example.test",
		Command: "NOTICE",
		Params:  []string{"autobrr", "invalid IRC key"}, // addressed to us, not a channel
	}
	h.handleInviteResponse(msg)

	if !waitFor(func() bool { return sm.channel.HasConnectionErrors() }, time.Second) {
		t.Fatal("bot DM should surface an invite failure on the awaiting channel")
	}
	if errs := sm.channel.ConnectionErrorsCopy(); len(errs) == 0 || !strings.Contains(errs[0], "invalid IRC key") {
		t.Fatalf("connection errors = %v, want the bot reason", errs)
	}
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

	if sm.channel.HasConnectionErrors() {
		t.Fatalf("channel message from the bot must not raise an invite failure: %v", sm.channel.ConnectionErrorsCopy())
	}
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

	if sm.channel.HasConnectionErrors() {
		t.Fatalf("DM from an unrelated nick must not raise an invite failure: %v", sm.channel.ConnectionErrorsCopy())
	}
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

	if sm.channel.HasConnectionErrors() {
		t.Fatalf("a monitoring channel must not be errored by a bot DM: %v", sm.channel.ConnectionErrorsCopy())
	}
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

	if sm.channel.HasConnectionErrors() {
		t.Fatalf("a prefix-overlapping unrelated nick must not park the channel: %v", sm.channel.ConnectionErrorsCopy())
	}
	if got := sm.CurrentState(); got != ChannelStateAwaitingInvite {
		t.Fatalf("channel should still be AwaitingInvite, got %s", got)
	}
}

// TestInviteFailedNeutralizesStaleTimeout verifies the parking is atomic w.r.t.
// the invite timeout: once parked, the timeout armed for that attempt cannot drag
// the channel back into the retry loop (OnInviteFailed bumps inviteGen).
func TestInviteFailedNeutralizesStaleTimeout(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.m.Lock()
	gen := sm.inviteGen
	sm.m.Unlock()

	sm.OnInviteFailed("invite rejected by Voyager: invalid IRC key")
	if !waitForState(sm, ChannelStateInviteFailed, time.Second) {
		t.Fatalf("expected InviteFailed, got %s", sm.CurrentState())
	}

	// the invite timeout that was armed for the now-parked attempt must no-op
	sm.onInviteTimeout(gen)

	if got := sm.CurrentState(); got != ChannelStateInviteFailed {
		t.Fatalf("a stale invite timeout un-parked the channel to %s", got)
	}
}

// TestAddChannelUnparksInviteFailedChannel verifies a live reconcile that re-adds
// an already-tracked channel parked in InviteFailed re-drives it through the join
// workflow (AddChannel must Reset before Start, else Start's transition from the
// sticky InviteFailed state is dropped as invalid and the channel stays stuck).
func TestAddChannelUnparksInviteFailedChannel(t *testing.T) {
	h, _ := newTestHandler()
	sm := addAwaitingInviteChannel(h, "#chan", "voyager autobot user key")

	sm.OnInviteFailed("invite rejected by Voyager: invalid IRC key")
	if !waitForState(sm, ChannelStateInviteFailed, time.Second) {
		t.Fatalf("expected InviteFailed before re-add, got %s", sm.CurrentState())
	}

	h.AddChannel(domain.IrcChannel{Name: "#chan", Enabled: true})

	if !waitFor(func() bool { return sm.CurrentState() != ChannelStateInviteFailed }, time.Second) {
		t.Fatalf("AddChannel should un-park an InviteFailed channel; still %s", sm.CurrentState())
	}
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
		if got := isChannelTarget(c.target); got != c.want {
			t.Errorf("isChannelTarget(%q) = %v, want %v", c.target, got, c.want)
		}
	}
}
