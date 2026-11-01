// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build e2e

package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/autobrr/autobrr/test/e2e/harness"
	"github.com/autobrr/autobrr/test/irc/ircd"
)

// sisterDefinition is a v2 indexer definition announcing in one channel on the
// shared test server. An empty inviteBot leaves out the invite command setting.
const sisterDefinition = `---
version: 2
name: %[1]s
identifier: %[2]s
description: %[1]s is a sister site on a shared IRC network.
language: en-us
urls:
  - https://%[2]s.test/
privacy: private
protocol: torrent
supports:
  - irc
source: custom
settings:
  - name: passkey
    type: secret
    required: true
    label: PassKey
    help: PassKey

irc:
  network: Shared
  server: %[3]s
  port: %[4]d
  tls: false
  settings:
    - name: nick
      type: text
      required: true
      label: Nick
      help: Bot nick

    - name: auth.account
      type: text
      required: false
      label: NickServ Account
      help: NickServ account

    - name: auth.password
      type: secret
      required: false
      label: NickServ Password
      help: NickServ password
%[6]s
  channels:
    - name: "%[5]s"
      announcers:
        - %[7]s
      parse:
        type: single
        lines:
          - pattern: 'NEW TORRENT :: (?P<releaseName>.+) :: (?P<baseUrl>https:\/\/.+\/)download\/(?P<torrentId>\d+)'
            tests:
              - line: 'NEW TORRENT :: That.Show.S01E01.1080p.WEB.h264-GRP :: https://%[2]s.test/download/1'
                expect:
                  releaseName: That.Show.S01E01.1080p.WEB.h264-GRP
                  baseUrl: https://%[2]s.test/
                  torrentId: "1"

        match:
          torrenturl: "/download/{{ .torrentId }}?passkey={{ .passkey }}"
`

const inviteSetting = `
    - name: invite_command
      type: secret
      default: "%s !invite USERNAME IRCKEY"
      required: true
      label: Invite command
      help: Invite command
`

type sisterSite struct {
	name       string
	identifier string
	channel    string
	inviteBot  string
}

func writeSisterDefinition(t *testing.T, dir string, srv *ircd.Server, site sisterSite) {
	t.Helper()

	invite := ""
	announcer := site.name + "Bot"
	if site.inviteBot != "" {
		invite = fmt.Sprintf(inviteSetting, site.inviteBot)
		announcer = site.inviteBot
	}

	definition := fmt.Sprintf(sisterDefinition, site.name, site.identifier, srv.Host(), srv.Port(), site.channel, invite, announcer)

	if err := os.WriteFile(filepath.Join(dir, site.identifier+".yaml"), []byte(definition), 0o644); err != nil {
		t.Fatalf("could not write definition for %s: %v", site.name, err)
	}
}

// Sister sites share one IRC network, so the second one added merges its
// channel into the first one's network. With per-site invite commands that
// merge restarts the running network, and the restart used to keep the old
// channel set: the new channel was stored but never joined, with no error,
// until autobrr itself restarted.
func TestAddSisterIndexerJoinsSharedNetwork(t *testing.T) {
	tests := []struct {
		name   string
		first  sisterSite
		second sisterSite
	}{
		{
			name:   "invite commands",
			first:  sisterSite{name: "SisterAlpha", identifier: "sisteralpha", channel: "#announce", inviteBot: "ALPHA"},
			second: sisterSite{name: "SisterBravo", identifier: "sisterbravo", channel: "#bravo-announce", inviteBot: "BRAVO"},
		},
		{
			name:   "no invite commands",
			first:  sisterSite{name: "SisterAlpha", identifier: "sisteralpha", channel: "#announce"},
			second: sisterSite{name: "SisterBravo", identifier: "sisterbravo", channel: "#bravo-announce"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := ircd.New(t, ircd.WithAccount("user", "nickserv-password"))

			definitions := t.TempDir()

			for _, site := range []sisterSite{tt.first, tt.second} {
				writeSisterDefinition(t, definitions, srv, site)

				if site.inviteBot == "" {
					continue
				}

				srv.AddChannel(site.channel, ircd.InviteOnly())
				srv.AddBot(site.inviteBot, func(b *ircd.Bot, from, text string) {
					if strings.HasPrefix(text, "!invite user") {
						b.Invite(from, site.channel)
					}
				})
			}

			app := harness.StartWithDefinitions(t, definitions)
			page := app.NewAuthedPage()

			addSister := func(site sisterSite) {
				page.Open("/settings/indexers")
				page.AddNew()
				page.Select(site.name)
				page.Fill("settings.passkey", site.identifier+"-passkey")
				page.Fill("irc.nick", "user_bot")
				page.Fill("irc.auth.account", "user")
				page.Fill("irc.auth.password", "nickserv-password")
				if site.inviteBot != "" {
					page.Fill("irc.invite_command", site.inviteBot+" !invite user "+site.identifier+"-irckey")
				}
				page.Submit()
				page.ExpectRow(site.name)
			}

			addSister(tt.first)

			page.Open("/settings/irc")
			page.ExpectRow("Shared")
			page.Toggle("enabled")

			app.WaitForChannelMonitoring(tt.first.channel, 30*time.Second)

			// the network is running when the sister site is added, which is
			// what sends the merge down the live update path
			addSister(tt.second)

			app.WaitForChannelMonitoring(tt.second.channel, 30*time.Second)
			app.WaitForChannelMonitoring(tt.first.channel, 30*time.Second)
		})
	}
}
