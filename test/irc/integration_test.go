// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build irc_integration_test

package irc_test

import (
	"strings"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/test/irc/harness"
	"github.com/autobrr/autobrr/test/irc/ircd"
)

// TestConnectJoinAnnounce is the vertical-slice smoke test: a None-auth network
// connects to the in-process server, joins a channel, and an announcer's line is
// parsed into a release. It proves the whole pipeline end to end over a socket.
func TestConnectJoinAnnounce(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#test", ircd.Announcer("announcer"))

	def := harness.MinimalDefinition("test", "#test", "announcer")

	network := domain.IrcNetwork{
		ID:       1,
		Name:     "Test",
		Server:   srv.Host(),
		Port:     srv.Port(),
		Nick:     "autobrr",
		Auth:     domain.IRCAuth{Mechanism: domain.IRCAuthMechanismNone},
		Channels: []domain.IrcChannel{{Name: "#test", Enabled: true}},
	}

	inst := harness.Start(t, network, []*domain.IndexerDefinition{def})

	inst.WaitForMonitoring("#test", 10*time.Second)

	srv.Announce("#test", "announcer", "New torrent: Some.Release.2024.1080p.BluRay.x264-GRP in Movies")

	rls, ok := inst.Releases.Wait(5 * time.Second)
	if !ok {
		t.Fatal("no release produced from announce")
	}
	if rls.TorrentName != "Some.Release.2024.1080p.BluRay.x264-GRP" {
		t.Fatalf("unexpected release name: %q", rls.TorrentName)
	}
}

// ---- auth

// TestSASLAuth verifies a SASL PLAIN login: the +r (registered-only) channel can
// only be joined once the client is authenticated, so reaching Monitoring proves
// SASL succeeded.
func TestSASLAuth(t *testing.T) {
	srv := ircd.New(t, ircd.WithAccount("autobrr", "s3cret"))
	srv.AddChannel("#sasl", ircd.RegisteredOnly(), ircd.Announcer("ann"))

	def := harness.MinimalDefinition("sasl", "#sasl", "ann")
	net := harness.Network(srv, "autobrr", harness.SASL("autobrr", "s3cret"), harness.Channel("#sasl"))

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#sasl", 10*time.Second)
}

// TestNickServAuth verifies NickServ IDENTIFY: again the +r channel gates on being
// authenticated, so Monitoring proves the IDENTIFY exchange worked.
func TestNickServAuth(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#ns", ircd.RegisteredOnly(), ircd.Announcer("ann"))

	def := harness.MinimalDefinition("ns", "#ns", "ann")
	net := harness.Network(srv, "autobrr", harness.NickServ("autobrr", "nspass"), harness.Channel("#ns"))

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#ns", 10*time.Second)
}

// TestNoneAuthJoinsDirectly verifies the simplest path: no auth, direct join.
func TestNoneAuthJoinsDirectly(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#none", ircd.Announcer("ann"))

	def := harness.MinimalDefinition("none", "#none", "ann")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#none"))

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#none", 10*time.Second)
}

// TestNoneAuthWithLeftoverPasswordSkipsNickServ is the regression test for the
// 2026-08 TorrentLeech ban wave: mechanism NONE with leftover credentials used
// to send a NickServ IDENTIFY on every connect, because the fallback was gated
// on password presence alone. The network must join and monitor without a
// single message to NickServ.
func TestNoneAuthWithLeftoverPasswordSkipsNickServ(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#none", ircd.Announcer("ann"))

	def := harness.MinimalDefinition("none", "#none", "ann")
	auth := harness.None()
	auth.Account = "autobrr"
	auth.Password = "leftover"
	net := harness.Network(srv, "autobrr", auth, harness.Channel("#none"))

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#none", 10*time.Second)

	if got := srv.NickServMessageCount(); got != 0 {
		t.Fatalf("mechanism NONE must not message NickServ, got %d message(s)", got)
	}
}

// TestNoneAuthIgnoresImpostorNickServ is the end-to-end guard for the stored
// credentials of a mechanism NONE network. On a network without services the
// nick "NickServ" is unregistered and free for anyone to take, so a hostile user
// can send whatever a real service would. Neither the notice that triggers the
// account-qualified IDENTIFY retry (which carries the password) nor a bogus
// rejection may have any effect.
func TestNoneAuthIgnoresImpostorNickServ(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#none", ircd.Announcer("ann"))
	impostor := srv.AddBot("NickServ", nil)

	def := harness.MinimalDefinition("none", "#none", "ann")
	auth := harness.None()
	auth.Account = "autobrr_acct"
	auth.Password = "leftover"
	net := harness.Network(srv, "autobrr", auth, harness.Channel("#none"))

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#none", 10*time.Second)

	// bait the IDENTIFY escalation, then bait the network-stopping branch
	impostor.Notice("autobrr", "Nick autobrr isn't registered.")
	impostor.Notice("autobrr", "Password incorrect.")

	// the announce path must still work, which also gives the notices time to land
	srv.Announce("#none", "ann", "New torrent: Some.Release.2024.1080p.BluRay.x264-GRP in Movies")
	if _, ok := inst.Releases.Wait(5 * time.Second); !ok {
		t.Fatal("no release produced after the impostor notices")
	}

	if got := srv.NickServMessageCount(); got != 0 {
		t.Fatalf("impostor elicited %d message(s) to NickServ; stored credentials must never be sent on a NONE network", got)
	}

	if inst.Handler.Stopped() {
		t.Fatal("an impostor NickServ must not be able to stop the network")
	}
}

// TestBannedStopsAndSurfacesReason verifies end to end that a server ban (465
// ERR_YOUREBANNEDCREEP, e.g. a G-Line) stops the network and surfaces the ban
// reason to the UI via a network-level HEALTH event, rather than reconnecting into
// a ban loop.
func TestBannedStopsAndSurfacesReason(t *testing.T) {
	const reason = "You are not welcome on this network. G-Lined: reconnect loop."
	srv := ircd.New(t, ircd.Banned(reason))
	srv.AddChannel("#none", ircd.Announcer("ann"))

	def := harness.MinimalDefinition("none", "#none", "ann")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#none"))

	inst := harness.Start(t, net, harness.Defs(def))

	// the ban reason is surfaced at the network level...
	inst.WaitForNetworkError("G-Lined", 10*time.Second)
	// ...and the handler stops rather than reconnecting into the ban
	inst.WaitForStopped(10 * time.Second)
}

// TestBannedAtRegistrationAbortsReconnectFast verifies the fatal-error
// short-circuit of the reconnect backoff: a connect-time ban (465 sent before
// registration completes) fails Connect(), but because handleBanned already
// stopped the handler, the retry loop aborts IMMEDIATELY (returns Unrecoverable)
// instead of waiting out the 15s reconnect backoff before noticing. The reason is
// still surfaced and the handler ends stopped.
func TestBannedAtRegistrationAbortsReconnectFast(t *testing.T) {
	const reason = "You are not welcome on this network. G-Lined: reconnect loop."
	srv := ircd.New(t, ircd.BannedAtRegistration(reason))
	srv.AddChannel("#none", ircd.Announcer("ann"))

	def := harness.MinimalDefinition("none", "#none", "ann")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#none"))

	start := time.Now()
	// Run returns an error here (the connect fails fatally); AllowRunError tolerates it
	inst := harness.Start(t, net, harness.Defs(def), harness.Options{AllowRunError: true})
	elapsed := time.Since(start)

	// without the short-circuit, Run would block ~15s on the reconnect backoff
	// before the retry loop noticed the Stop()
	if elapsed > 10*time.Second {
		t.Fatalf("Run blocked %s on a fatal ban; reconnect backoff was not short-circuited", elapsed)
	}

	inst.WaitForNetworkError("G-Lined", 5*time.Second)
	if !inst.Handler.Stopped() {
		t.Fatal("handler should be stopped after a connect-time ban")
	}
}

// TestPreRegistrationErrorsStopNetwork is the end-to-end guard for the reconnect
// hammering the circuit breaker exists to stop, on the path that is easiest to
// miss: a server refusing us BEFORE registration completes. Those connections
// never reach the disconnect callback, so the ERROR line is the only signal, and
// without it the client would keep reconnecting every 15s indefinitely. The
// network must stop and surface the server's own reason.
func TestPreRegistrationErrorsStopNetwork(t *testing.T) {
	const reason = "Trying to reconnect too fast."

	// more refusals than the breaker's threshold, delivered on one connection so
	// the test does not wait out a reconnect cycle per refusal
	srv := ircd.New(t, ircd.ErrorBeforeRegistration(reason, 10))
	srv.AddChannel("#none", ircd.Announcer("ann"))

	def := harness.MinimalDefinition("none", "#none", "ann")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#none"))

	// tripping the breaker stops the handler mid-connect, so Run reports the
	// aborted connection
	inst := harness.Start(t, net, harness.Defs(def), harness.Options{AllowRunError: true})

	inst.WaitForNetworkError(reason, 10*time.Second)
	inst.WaitForStopped(10 * time.Second)
}

// ---- kick

// TestKickDoesNotRejoin verifies the deliberate policy that a kicked channel is
// NOT auto-rejoined (to avoid bans): after a KICK the channel reports Kicked and
// the server never sees a second join.
func TestKickDoesNotRejoin(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#k", ircd.Announcer("ann"))

	def := harness.MinimalDefinition("k", "#k", "ann")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#k"))

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#k", 10*time.Second)

	if got := srv.JoinCount("#k"); got != 1 {
		t.Fatalf("expected 1 join before kick, got %d", got)
	}

	srv.Kick("#k", "autobrr", "chanop", "you were naughty")

	inst.WaitForState("#k", "Kicked", 10*time.Second)

	// give any (unwanted) rejoin attempt time to happen, then assert none did
	time.Sleep(500 * time.Millisecond)
	if got := srv.JoinCount("#k"); got != 1 {
		t.Fatalf("kicked channel must not auto-rejoin; join count = %d", got)
	}
}

// ---- modes

// TestChannelKeyMissingErrors verifies a +k channel joined without a key surfaces
// the specific 475 (bad key) reason on the channel instead of stalling.
func TestChannelKeyMissingErrors(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#locked", ircd.Key("s3cr3t"), ircd.Announcer("ann"))

	def := harness.MinimalDefinition("locked", "#locked", "ann")
	// channel supplies NO password -> JOIN #locked without a key -> 475
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#locked"))

	inst := harness.Start(t, net, harness.Defs(def))

	inst.WaitForState("#locked", "Error", 10*time.Second)
	if reason := inst.LastError("#locked"); !strings.Contains(reason, "+k") {
		t.Fatalf("expected a +k (bad key) reason, got %q", reason)
	}
}

// TestChannelKeyProvidedJoins verifies supplying the channel password lets the
// +k join succeed (proves the password is sent on JOIN).
func TestChannelKeyProvidedJoins(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#locked", ircd.Key("s3cr3t"), ircd.Announcer("ann"))

	def := harness.MinimalDefinition("locked", "#locked", "ann")
	net := harness.Network(srv, "autobrr", harness.None(), harness.ChannelWithPassword("#locked", "s3cr3t"))

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#locked", 10*time.Second)
}

// TestInviteOnlyWithoutInviteErrors verifies a +i channel joined directly (no
// invite command configured) surfaces the 473 reason.
func TestInviteOnlyWithoutInviteErrors(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#invonly", ircd.InviteOnly(), ircd.Announcer("ann"))

	def := harness.MinimalDefinition("invonly", "#invonly", "ann")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#invonly"))

	inst := harness.Start(t, net, harness.Defs(def))

	inst.WaitForState("#invonly", "Error", 10*time.Second)
	if reason := inst.LastError("#invonly"); !strings.Contains(reason, "+i") {
		t.Fatalf("expected a +i (invite-only) reason, got %q", reason)
	}
}

// ---- invite

const inviteCmd = "Gatekeeper enter #inv autobrr IRCKEY"

// TestInviteAccepted verifies the happy invite path: the handler messages the
// gatekeeper, the bot INVITEs it, and it joins the +i channel.
func TestInviteAccepted(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#inv", ircd.InviteOnly(), ircd.Announcer("Gatekeeper"))
	srv.AddBot("Gatekeeper", func(b *ircd.Bot, from, text string) {
		b.Invite(from, "#inv")
	})

	def := harness.InviteDefinition("inv", "#inv", "Gatekeeper", "Gatekeeper")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#inv"))
	net.InviteCommand = inviteCmd

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#inv", 15*time.Second)
}

// TestInviteRejectedParks verifies the fix from this session: a present bot that
// answers the invite command with a message (rejection) parks the channel in
// InviteFailed with the bot's reason surfaced - it does NOT keep retrying.
func TestInviteRejectedParks(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#inv", ircd.InviteOnly(), ircd.Announcer("Gatekeeper"))
	srv.AddBot("Gatekeeper", func(b *ircd.Bot, from, text string) {
		b.Notice(from, "Invalid IRCKEY - request denied")
	})

	def := harness.InviteDefinition("inv", "#inv", "Gatekeeper", "Gatekeeper")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#inv"))
	net.InviteCommand = inviteCmd

	inst := harness.Start(t, net, harness.Defs(def))

	inst.WaitForState("#inv", "InviteFailed", 15*time.Second)
	if reason := inst.LastError("#inv"); !strings.Contains(reason, "Invalid IRCKEY") {
		t.Fatalf("expected the bot's rejection reason to be surfaced, got %q", reason)
	}
}

// TestInviteBotAbsentRetries verifies the counterpart: when the gatekeeper is not
// present (no such nick), the channel keeps retrying via backoff rather than
// parking - the bot may simply not be connected yet.
func TestInviteBotAbsentRetries(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#inv", ircd.InviteOnly(), ircd.Announcer("Gatekeeper"))
	// deliberately no AddBot -> PRIVMSG to Gatekeeper yields 401 no-such-nick

	def := harness.InviteDefinition("inv", "#inv", "Gatekeeper", "Gatekeeper")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#inv"))
	net.InviteCommand = inviteCmd

	inst := harness.Start(t, net, harness.Defs(def))

	// an absent bot routes into the invite backoff loop
	inst.WaitForState("#inv", "AwaitingInviteBot", 15*time.Second)
	// and must NOT have parked in InviteFailed
	if inst.LastError("#inv") != "" && strings.Contains(strings.ToLower(inst.LastError("#inv")), "rejected") {
		t.Fatalf("absent bot must not surface a rejection error")
	}
}

// TestInviteLateAcceptRecovers verifies a bot that first rejects (parking the
// channel) and later INVITEs it anyway still recovers - the park is not terminal.
func TestInviteLateAcceptRecovers(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#inv", ircd.InviteOnly(), ircd.Announcer("Gatekeeper"))

	// the gatekeeper rejects the invite command, parking the channel
	bot := srv.AddBot("Gatekeeper", func(b *ircd.Bot, from, text string) {
		b.Notice(from, "Invalid IRCKEY - request denied")
	})

	def := harness.InviteDefinition("inv", "#inv", "Gatekeeper", "Gatekeeper")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#inv"))
	net.InviteCommand = inviteCmd

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForState("#inv", "InviteFailed", 15*time.Second)

	// the gatekeeper later invites us anyway (e.g. an admin approved the request);
	// the parked channel must accept the INVITE and join
	bot.Invite("autobrr", "#inv")
	inst.WaitForMonitoring("#inv", 15*time.Second)
}

// TestInvitePositiveAckForceJoin verifies the PTP / Hummingbird flow end to end:
// the invite bot answers the invite command with a POSITIVE NOTICE ("attempting
// to join you") and the server then force-joins us. The positive message must not
// be mistaken for a rejection - the channel must reach Monitoring, and no invite
// error may be surfaced. Regression test for a bot reply being treated as a
// failure even though the join succeeds.
func TestInvitePositiveAckForceJoin(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#inv", ircd.InviteOnly(), ircd.Announcer("Gatekeeper"))
	srv.AddBot("Gatekeeper", func(b *ircd.Bot, from, text string) {
		// acknowledge positively, then force-join - like Hummingbird
		b.Notice(from, "Hello! Attempting to join you to #inv")
		b.ForceJoin(from, "#inv")
	})

	def := harness.InviteDefinition("inv", "#inv", "Gatekeeper", "Gatekeeper")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#inv"))
	net.InviteCommand = inviteCmd

	inst := harness.Start(t, net, harness.Defs(def))
	inst.WaitForMonitoring("#inv", 15*time.Second)
	// the whole network must report healthy (reached operational), not just the channel
	inst.WaitForHealthy(15 * time.Second)

	if reason := inst.LastError("#inv"); reason != "" {
		t.Fatalf("a positive-ack force-join must not surface an error, got %q", reason)
	}
}

// TestInviteRejectThenLateForceJoinRecovers exercises the network-recovery fix end
// to end: the bot answers with a NOTICE (parking the channel in InviteFailed once
// the grace elapses with no join) and only LATER force-joins us. The channel must
// recover to Monitoring AND the network-level state machine must climb back out of
// the transient error to healthy, rather than staying stuck until a reconnect.
func TestInviteRejectThenLateForceJoinRecovers(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#inv", ircd.InviteOnly(), ircd.Announcer("Gatekeeper"))
	srv.AddBot("Gatekeeper", func(b *ircd.Bot, from, text string) {
		b.Notice(from, "Please hold, verifying your request...")
		// force-join only AFTER the grace (prod default 5s) has parked the channel
		go func() {
			time.Sleep(7 * time.Second)
			b.ForceJoin(from, "#inv")
		}()
	})

	def := harness.InviteDefinition("inv", "#inv", "Gatekeeper", "Gatekeeper")
	net := harness.Network(srv, "autobrr", harness.None(), harness.Channel("#inv"))
	net.InviteCommand = inviteCmd

	inst := harness.Start(t, net, harness.Defs(def))

	// the bot answered but did not join in time -> parks
	inst.WaitForState("#inv", "InviteFailed", 15*time.Second)
	// the late force-join recovers the channel and the network returns to healthy
	inst.WaitForMonitoring("#inv", 15*time.Second)
	inst.WaitForHealthy(15 * time.Second)
}
