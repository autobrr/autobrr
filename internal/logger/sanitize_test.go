// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestSanitizeLogFile(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "BHD_URL",
			input:    "\"module\":\"filter\" https://beyond-hd.me/torrent/download/auto.t0rrent1d.rssk3y",
			expected: "\"module\":\"filter\" https://beyond-hd.me/torrent/download/auto.t0rrent1d.REDACTED",
		},
		{
			name:     "Standard_UNIT3D_URL",
			input:    "\"module\":\"filter\" https://aither.cc/torrent/download/t0rrent1d.rssk3y",
			expected: "\"module\":\"filter\" https://aither.cc/torrent/download/t0rrent1d.REDACTED",
		},
		{
			name:     "TL_URL",
			input:    "\"module\":\"filter\" https://www.torrentleech.org/rss/download/t0rrent1d/rssk3y/Dark+Places+1974+1080p+BluRay+x264-GAZER.torrent",
			expected: "\"module\":\"filter\" https://www.torrentleech.org/rss/download/t0rrent1d/REDACTED/Dark+Places+1974+1080p+BluRay+x264-GAZER.torrent",
		},
		{
			name:     "auth_key_torrent_pass",
			input:    "\"module\":\"filter\" https://alpharatio.cc/torrents.php?action=download&id=t0rrent1d&authkey=4uthk3y&torrent_pass=t0rrentp4ss",
			expected: "\"module\":\"filter\" https://alpharatio.cc/torrents.php?action=download&id=t0rrent1d&authkey=REDACTED&torrent_pass=REDACTED",
		},
		{
			input:    "\"module\":\"irc\" LiMEY_ !invite 1irck3y us3rn4me",
			expected: "\"module\":\"irc\" LiMEY_ !invite REDACTED us3rn4me",
		},
		{
			input:    "\"module\":\"irc\" Voyager autobot us3rn4me 1irck3y",
			expected: "\"module\":\"irc\" Voyager autobot us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" Satsuki enter #announce us3rn4me 1irck3y",
			expected: "\"module\":\"irc\" Satsuki enter #announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" Sauron bot #ant-announce us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" Sauron bot #ant-announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" Millie announce IRCKEY",
			expected: "\"module\":\"irc\" Millie announce REDACTED",
		},
		{
			input:    "\"module\":\"irc\" DBBot announce IRCKEY",
			expected: "\"module\":\"irc\" DBBot announce REDACTED",
		},
		{
			input:    "\"module\":\"irc\" PT-BOT invite IRCKEY",
			expected: "\"module\":\"irc\" PT-BOT invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" midgards announce IRCKEY",
			expected: "\"module\":\"irc\" midgards announce REDACTED",
		},
		{
			input:    "\"module\":\"irc\" HeBoT !invite IRCKEY",
			expected: "\"module\":\"irc\" HeBoT !invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" NBOT !invite IRCKEY",
			expected: "\"module\":\"irc\" NBOT !invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" PS-Info pass IRCKEY",
			expected: "\"module\":\"irc\" PS-Info pass REDACTED",
		},
		{
			input:    "\"module\":\"irc\" Synd1c4t3 invite IRCKEY",
			expected: "\"module\":\"irc\" Synd1c4t3 invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" UHDBot invite IRCKEY",
			expected: "\"module\":\"irc\" UHDBot invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" ENDOR !invite us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" ENDOR !invite us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" Vertigo ENTER #GGn-Announce us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" Vertigo ENTER #GGn-Announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" immortal invite us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" immortal invite us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" Muffit bot #nbl-announce us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" Muffit bot #nbl-announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" hermes enter #announce us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" hermes enter #announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" Hummingbird ENTER us3rn4me IRCKEY #ptp-announce-dev",
			expected: "\"module\":\"irc\" Hummingbird ENTER us3rn4me REDACTED #ptp-announce-dev",
		},
		{
			input:    "\"module\":\"irc\" Drone enter #red-announce us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" Drone enter #red-announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" RevoTT !invite us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" RevoTT !invite us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" SceneHD .invite IRCKEY #announce",
			expected: "\"module\":\"irc\" SceneHD .invite REDACTED #announce",
		},
		{
			input:    "\"module\":\"irc\" erica letmeinannounce us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" erica letmeinannounce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" Cerberus identify us3rn4me IRCKEY",
			expected: "\"module\":\"irc\" Cerberus identify us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" NickServ IDENTIFY Nvbkødn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" NickServ IDENTIFY REDACTED",
		},
		{
			input:    "\"module\":\"irc\" PRIVMSG NickServ IDENTIFY zAPEJEA8ryYnpj3AiE3KJ",
			expected: "\"module\":\"irc\" PRIVMSG NickServ IDENTIFY REDACTED",
		},
		{
			input:    "\"module\":\"action\" \\\"host\\\":\\\"subdomain.domain.com:42069/subfolder\\\", \\n   \\\"user\\\":\\\"AUserName\\\", \\n   \\\"password\\\":\\\"p4ssw0!rd\\\", \\n",
			expected: "\"module\":\"action\" \\\"host\\\":\\\"REDACTED\\\", \\n   \\\"user\\\":\\\"REDACTED\\\", \\n   \\\"password\\\":\\\"REDACTED\\\", \\n",
		},
		{
			input:    "\"module\":\"action\" ExternalWebhookHost:http://127.0.0.1:6940/api/upgrade ExternalWebhookData:",
			expected: "\"module\":\"action\" ExternalWebhookHost:REDACTED ExternalWebhookData:",
		},
		{
			input:    "\"module\":\"filter\" \\\"id\\\": 3855,\\n  \\\"apikey\\\": \\\"ad789a9s8d.asdpoiasdpojads09sad809\\\",\\n  \\\"minratio\\\": 10.0\\n",
			expected: "\"module\":\"filter\" \\\"id\\\": 3855,\\n  \\\"apikey\\\": \\\"REDACTED\\\",\\n  \\\"minratio\\\": 10.0\\n",
		},
		{
			input:    "\"module\":\"filter\" request: https://username:password@111.server.name.here/qbittorrent/api/v2/torrents/info: error making request",
			expected: "\"module\":\"filter\" request: https://REDACTED_USER:REDACTED_PW@111.server.name.here/qbittorrent/api/v2/torrents/info: error making request",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Create a temporary file with sample log data
			tmpFile, err := os.CreateTemp("", "test-log-*.log")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())

			// Write the test case input to the temporary file
			_, err = tmpFile.WriteString(testCase.input + "\n")
			if err != nil {
				tmpFile.Close()
				t.Fatal(err)
			}
			err = tmpFile.Close()
			if err != nil {
				t.Fatal(err)
			}

			// Create a bytes.Buffer to store the sanitized content
			sanitizedContent := &bytes.Buffer{}

			// Call SanitizeLogFile on the temporary file
			err = SanitizeLogFile(tmpFile.Name(), sanitizedContent)
			if err != nil {
				t.Fatal(err)
			}

			// Read the content of the sanitized content
			sanitizedData := sanitizedContent.String()

			// Check if the sanitized data matches the expected content
			if !strings.Contains(sanitizedData, testCase.expected+"\n") {
				t.Errorf("Sanitized data does not match expected data\nExpected:\n%s\nActual:\n%s", testCase.expected, sanitizedData)
			}
		})
	}
}
