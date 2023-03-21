package http

import (
	"bytes"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"testing"
)

func TestSanitizeLogFile(t *testing.T) {
	testCases := []struct {
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
			input:    "Voyager autobot us3rn4me 1RCK3Y",
			expected: "Voyager autobot us3rn4me REDACTED",
		},
		{
			input:    "Satsuki enter #announce us3rn4me 1RCK3Y",
			expected: "Satsuki enter #announce us3rn4me REDACTED",
		},
		{
			input:    "Millie announce 1RCK3Y",
			expected: "Millie announce REDACTED",
		},
		{
			input:    "DBBot announce 1RCK3Y",
			expected: "DBBot announce REDACTED",
		},
		{
			input:    "ENDOR !invite us3rnøme 1RCK3Y",
			expected: "ENDOR !invite us3rnøme REDACTED",
		},
		{
			input:    "Vertigo ENTER #GGn-Announce us3rn4me 1RCK3Y",
			expected: "Vertigo ENTER #GGn-Announce us3rn4me REDACTED",
		},
		{
			input:    "midgards announce 1RCK3Y",
			expected: "midgards announce REDACTED",
		},
		{
			input:    "HeBoT !invite 1RCK3Y",
			expected: "HeBoT !invite REDACTED",
		},
		{
			input:    "NBOT !invite 1RCK3Y",
			expected: "NBOT !invite REDACTED",
		},
		{
			input:    "Muffit bot #nbl-announce us3rn4me 1RCK3Y",
			expected: "Muffit bot #nbl-announce us3rn4me REDACTED",
		},
		{
			input:    "hermes enter #announce us3rn4me 1RCK3Y",
			expected: "hermes enter #announce us3rn4me REDACTED",
		},
		{
			input:    "LiMEY_ !invite 1RCK3Y us3rn4me",
			expected: "LiMEY_ !invite REDACTED us3rn4me",
		},
		{
			input:    "PS-Info pass 1RCK3Y",
			expected: "PS-Info pass REDACTED",
		},
		{
			input:    "PT-BOT invite 1RCK3Y",
			expected: "PT-BOT invite REDACTED",
		},
		{
			input:    "Hummingbird ENTER us3rn4me 1RCK3Y #ptp-announce-dev",
			expected: "Hummingbird ENTER us3rn4me REDACTED #ptp-announce-dev",
		},
		{
			input:    "Drone enter #red-announce us3rn4me 1RCK3Y",
			expected: "Drone enter #red-announce us3rn4me REDACTED",
		},
		{
			input:    "SceneHD .invite 1RCK3Y #announce",
			expected: "SceneHD .invite REDACTED #announce",
		},
		{
			input:    "erica letmeinannounce us3rn4me 1RCK3Y",
			expected: "erica letmeinannounce us3rn4me REDACTED",
		},
		{
			input:    "Synd1c4t3 invite 1RCK3Y",
			expected: "Synd1c4t3 invite REDACTED",
		},
		{
			input:    "UHDBot invite 1RCK3Y",
			expected: "UHDBot invite REDACTED",
		},
		{
			input:    "Sauron bot #ant-announce us3rn4me 1RCK3Y",
			expected: "Sauron bot #ant-announce us3rn4me REDACTED",
		},
		{
			input:    "RevoTT !invite us3rn4me P4SSK3Y",
			expected: "RevoTT !invite us3rn4me REDACTED",
		},
		{
			input:    "Cerberus identify us3rn4me P1D",
			expected: "Cerberus identify us3rn4me REDACTED",
		},
		{
			input:    "NickServ IDENTIFY dasøl13sa#!",
			expected: "NickServ IDENTIFY REDACTED",
		},
		{
			input:    "--> AUTHENTICATE poasd!232kljøasdj!%",
			expected: "--> AUTHENTICATE REDACTED",
		},
	}

	// Create a temporary file with sample log data
	tmpFile, err := ioutil.TempFile("", "test-log-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	for _, testCase := range testCases {
		// Write sample log data to the temporary file
		_, err = tmpFile.WriteString(testCase.input + "\n")
		if err != nil {
			tmpFile.Close()
			t.Fatal(err)
		}
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

	// Combine the expected sanitized lines
	expectedSanitizedData := ""
	for _, testCase := range testCases {
		expectedSanitizedData += testCase.expected + "\n"
	}

	// Split and sort the sanitized data and expected data
	sanitizedLines := strings.Split(sanitizedData, "\n")
	expectedLines := strings.Split(expectedSanitizedData, "\n")

	sort.Strings(sanitizedLines)
	sort.Strings(expectedLines)

	// Join the sorted lines back together
	sortedSanitizedData := strings.Join(sanitizedLines, "\n")
	sortedExpectedData := strings.Join(expectedLines, "\n")

	// Check if the sanitized data matches the expected content
	if sortedSanitizedData != sortedExpectedData {
		t.Errorf("Sanitized data does not match expected data\nExpected:\n%s\nActual:\n%s", sortedExpectedData, sortedSanitizedData)
	}
}
