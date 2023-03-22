package http

import (
	"bytes"
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestSanitizeLogFile(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			input:    "https://beyond-hd.me/torrent/download/auto.t0rrent1d.rssk3y",
			expected: "https://beyond-hd.me/torrent/download/auto.t0rrent1d.REDACTED",
		},
		{
			input:    "https://aither.cc/torrent/download/t0rrent1d.rssk3y",
			expected: "https://aither.cc/torrent/download/t0rrent1d.REDACTED",
		},
		{
			input:    "https://www.torrentleech.org/rss/download/t0rrent1d/rssk3y/Dark+Places+1974+1080p+BluRay+x264-GAZER.torrent",
			expected: "https://www.torrentleech.org/rss/download/t0rrent1d/REDACTED/Dark+Places+1974+1080p+BluRay+x264-GAZER.torrent",
		},
		{
			input:    "https://alpharatio.cc/torrents.php?action=download&id=t0rrent1d&authkey=4uthk3y&torrent_pass=t0rrentp4ss",
			expected: "https://alpharatio.cc/torrents.php?action=download&id=t0rrent1d&authkey=REDACTED&torrent_pass=REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Voyager autobot us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Voyager autobot us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Satsuki enter #announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Satsuki enter #announce us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Millie announce Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Millie announce REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla DBBot announce Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla DBBot announce REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla ENDOR !invite us3rnøme Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla ENDOR !invite us3rnøme REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Vertigo ENTER #GGn-Announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Vertigo ENTER #GGn-Announce us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla midgards announce Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla midgards announce REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla HeBoT !invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla HeBoT !invite REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla NBOT !invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla NBOT !invite REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Muffit bot #nbl-announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Muffit bot #nbl-announce us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla hermes enter #announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla hermes enter #announce us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla LiMEY_ !invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg us3rn4me",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla LiMEY_ !invite REDACTED us3rn4me",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla PS-Info pass Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla PS-Info pass REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla PT-BOT invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla PT-BOT invite REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Hummingbird ENTER us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg #ptp-announce-dev",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Hummingbird ENTER us3rn4me REDACTED #ptp-announce-dev",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Drone enter #red-announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Drone enter #red-announce us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla SceneHD .invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg #announce",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla SceneHD .invite REDACTED #announce",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla erica letmeinannounce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla erica letmeinannounce us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Synd1c4t3 invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Synd1c4t3 invite REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla UHDBot invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla UHDBot invite REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Sauron bot #ant-announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Sauron bot #ant-announce us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla RevoTT !invite us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla RevoTT !invite us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Cerberus identify us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" bla bla Cerberus identify us3rn4me REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" --> AUTHENTICATE poasd!232kljøasdj!%",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" --> AUTHENTICATE REDACTED",
		},
		{
			input:    "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" NickServ IDENTIFY Nvbkødn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "{\"level\":\"trace\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T06:51:02Z\",\"message\":\" NickServ IDENTIFY REDACTED",
		},
		{
			input:    "{\"level\":\"debug\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T10:40:08Z\",\"message\":\"--> PRIVMSG NickServ IDENTIFY zAPEJEA8ryYnpj3AiE3KJ",
			expected: "{\"level\":\"debug\",\"module\":\"irc\",\"network\":\"irc.digitalirc.org\",\"time\":\"2023-03-22T10:40:08Z\",\"message\":\"--> PRIVMSG NickServ IDENTIFY REDACTED",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Create a temporary file with sample log data
			tmpFile, err := ioutil.TempFile("", "test-log-*.log")
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

			// Call SanitizeLogFile on the temporary file
			sanitizedContent, err := SanitizeLogFile(tmpFile.Name())
			if err != nil {
				t.Fatal(err)
			}

			// Read the content of the sanitized content
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(sanitizedContent)
			if err != nil {
				t.Fatal(err)
			}

			sanitizedData := buf.String()

			// Check if the sanitized data matches the expected content
			if !strings.Contains(sanitizedData, testCase.expected+"\n") {
				t.Errorf("Sanitized data does not match expected data\nExpected:\n%s\nActual:\n%s", testCase.expected, sanitizedData)
			}
		})
	}
}
