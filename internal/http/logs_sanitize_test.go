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
			input:    "\"module\":\"irc\",\"network\":\"irc.alpharatio.cc\",\"time\":\"2023-03-21T15:52:02Z\",\"message\":\"--> PRIVMSG Voyager autobot flubbah ad90a8sd09asd8",
			expected: "\"module\":\"irc\",\"network\":\"irc.alpharatio.cc\",\"time\":\"2023-03-21T15:52:02Z\",\"message\":\"--> PRIVMSG Voyager autobot flubbah REDACTED",
		},
		{
			input:    "\"module\":\"irc\" something Satsuki enter #announce us3rn4me 1RCK3Y",
			expected: "\"module\":\"irc\" something Satsuki enter #announce us3rn4me REDACTED",
		},
		{
			input:    "\"module\":\"irc\" something NickServ IDENTIFY dasøl13sa#!",
			expected: "\"module\":\"irc\" something NickServ IDENTIFY REDACTED",
		},
		{
			input:    "\"module\":\"irc\" something --> AUTHENTICATE poasd!232kljøasdj!%",
			expected: "\"module\":\"irc\" something --> AUTHENTICATE REDACTED",
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
	var expectedSanitizedData strings.Builder
	for i, testCase := range testCases {
		if i > 0 {
			expectedSanitizedData.WriteByte('\n')
		}
		expectedSanitizedData.WriteString(testCase.expected)
	}

	// Split the sanitized data and expected data into lines
	sanitizedLines := strings.Split(sanitizedData, "\n")
	expectedLines := strings.Split(expectedSanitizedData.String(), "\n")

	// Sort the lines
	sort.Strings(sanitizedLines)
	sort.Strings(expectedLines)

}
