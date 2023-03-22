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
			input:    "\"module\":\"irc\" bla bla Voyager autobot us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Voyager autobot us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla Satsuki enter #announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Satsuki enter #announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla Millie announce Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Millie announce REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla DBBot announce Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla DBBot announce REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla ENDOR !invite us3rnøme Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla ENDOR !invite us3rnøme REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla Vertigo ENTER #GGn-Announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Vertigo ENTER #GGn-Announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla midgards announce Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla midgards announce REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla HeBoT !invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla HeBoT !invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla NBOT !invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla NBOT !invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla Muffit bot #nbl-announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Muffit bot #nbl-announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla hermes enter #announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla hermes enter #announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla LiMEY_ !invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg us3rn4me",
			expected: "\"module\":\"irc\" bla bla LiMEY_ !invite REDACTED us3rn4me",
		},
		{
			input:    "\"module\":\"irc\" bla bla PS-Info pass Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla PS-Info pass REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla PT-BOT invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla PT-BOT invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla Hummingbird ENTER us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg #ptp-announce-dev",
			expected: "\"module\":\"irc\" bla bla Hummingbird ENTER us3rn4me REDACTED #ptp-announce-dev",
		},
		{
			input:    "\"module\":\"irc\" bla bla Drone enter #red-announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Drone enter #red-announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla SceneHD .invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg #announce",
			expected: "\"module\":\"irc\" bla bla SceneHD .invite REDACTED #announce",
		},
		{
			input:    "\"module\":\"irc\" bla bla erica letmeinannounce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla erica letmeinannounce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla Synd1c4t3 invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Synd1c4t3 invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla UHDBot invite Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla UHDBot invite REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla Sauron bot #ant-announce us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Sauron bot #ant-announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla RevoTT !invite us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla RevoTT !invite us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla Cerberus identify us3rn4me Nvbkddn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla Cerberus identify us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla NickServ IDENTIFY Nvbkødn~vzjHkPEimnJ6PmJw8ayiE#wg",
			expected: "\"module\":\"irc\" bla bla NickServ IDENTIFY REDACTED",
		},
		{
			input:    "\"module\":\"irc\" bla bla --> AUTHENTICATE poasd!232kljøasdj!%",
			expected: "\"module\":\"irc\" bla bla --> AUTHENTICATE REDACTED",
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
