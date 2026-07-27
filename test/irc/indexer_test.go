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

type authKind int

const (
	authNone authKind = iota
	authSASL
	authNickServ
)

// indexer mirrors one row of irc_test.md. The test derives the server/channel
// setup from these columns rather than reproducing every literal mode letter:
//   - auth            -> network auth mechanism
//   - invite          -> channel is +i and reached via an invite gatekeeper bot
//   - modes           -> a lowercase 'r' gates on registration, but only when the
//     row actually authenticates (a None network with +r still joins in reality,
//     because the announce bot is exempt - modelling it as gated would be wrong)
//   - announcerPresent -> whether the announcer shows up in NAMES; either way its
//     announces must still parse (the invisible-announcer / auditorium case)
type indexer struct {
	indexer          string
	server           string // informational; the tests use the local ircd
	channel          string
	auth             authKind
	invite           bool
	modes            string
	announcerPresent bool
}

var indexers = []indexer{
	{"Aither", "Aither", "#feed", authNickServ, false, "+MNnRrt", true},
	{"AlphaRatio", "AlphaRatio", "#announce", authNone, true, "+ituTPsmCn", false},
	{"BeyondHD", "BeyondHD", "#bhd_announce", authSASL, true, "+iuPrsm", false},
	{"Brokenstones", "BrokenStones", "#announce", authNickServ, true, "+itrn", true},
	{"BTN", "BTN", "#btn-announce", authNickServ, true, "+ituDrRmn", false},
	{"DarkPeers", "DarkPeers", "#announce", authNone, false, "+tuprmn", false},
	{"DigitalCore", "DigitalCore", "#announce", authSASL, true, "+itPrsn", true},
	{"EMP", "DigitalIRC", "#empornium", authSASL, false, "+zZtrsRmCn", true},
	{"GGn", "GGn", "#ggn-announce", authSASL, true, "+ituTPrsRmn", false},
	{"IPTorrents", "IPTorrents", "#ipt.announce", authNone, false, "+trmn", true},
	{"MyAnonamouse", "MAM", "#announce", authNickServ, false, "+tuPrsRmn", false},
	{"MidnightScene", "MidnightScene", "#announce", authSASL, false, "+tPrmn", true},
	{"PhoenixProject", "PhoenixProject", "#announce", authSASL, true, "+itrsmn", true},
	{"PTP", "PTP", "#ptp-announce", authSASL, true, "+mPnTrtui", false},
	{"Redacted", "Scratch-network", "#red-announce", authSASL, true, "+itupRrsmn", false},
	{"HDBits", "P2P-Network", "#hdbits.announce", authSASL, true, "+MitKrsmCn", true},
	{"Milkie", "P2P-Network", "#milkie-announce", authNone, false, "+trsmn", true},
}

// TestAllIndexers reproduces every irc_test.md row: connect with the row's auth,
// join the channel (direct or via invite), and confirm an announce from the
// (possibly invisible) announcer is parsed into a release.
func TestAllIndexers(t *testing.T) {
	for _, tr := range indexers {
		t.Run(tr.indexer, func(t *testing.T) {
			t.Parallel()
			runIndexer(t, tr)
		})
	}
}

func runIndexer(t *testing.T, tr indexer) {
	t.Helper()

	srv := ircd.New(t)

	opts := []ircd.ChannelOption{}
	if tr.announcerPresent {
		opts = append(opts, ircd.Announcer("Announcer"))
	} else {
		opts = append(opts, ircd.HiddenAnnouncer("Announcer"))
	}
	if tr.invite {
		opts = append(opts, ircd.InviteOnly())
	}
	if tr.auth != authNone && strings.ContainsRune(tr.modes, 'r') {
		opts = append(opts, ircd.RegisteredOnly())
	}
	srv.AddChannel(tr.channel, opts...)

	network := harness.Network(srv, "autobrr", authFor(tr.auth), harness.Channel(tr.channel))

	var def *domain.IndexerDefinition
	if tr.invite {
		srv.AddBot("Gatekeeper", func(b *ircd.Bot, from, _ string) { b.Invite(from, tr.channel) })
		def = harness.InviteDefinition(tr.indexer, tr.channel, "Announcer", "Gatekeeper")
		network.InviteCommand = "Gatekeeper enter " + tr.channel + " autobrr IRCKEY"
	} else {
		def = harness.MinimalDefinition(tr.indexer, tr.channel, "Announcer")
	}

	inst := harness.Start(t, network, harness.Defs(def))

	inst.WaitForMonitoring(tr.channel, 20*time.Second)

	// the announcer posts a line; it must parse whether or not the announcer is
	// visible in NAMES (auditorium / hidden announcers still announce)
	name := tr.indexer + ".Release.2024.1080p.BluRay.x264-GRP"
	srv.Announce(tr.channel, "Announcer", "New torrent: "+name+" in Movies")

	rls, ok := inst.Releases.Wait(5 * time.Second)
	if !ok {
		t.Fatalf("%s: no release produced from announce on %s", tr.indexer, tr.channel)
	}
	if rls.TorrentName != name {
		t.Fatalf("%s: release name = %q, want %q", tr.indexer, rls.TorrentName, name)
	}
}

func authFor(kind authKind) domain.IRCAuth {
	switch kind {
	case authSASL:
		return harness.SASL("autobrr", "pass")
	case authNickServ:
		return harness.NickServ("autobrr", "pass")
	default:
		return harness.None()
	}
}
