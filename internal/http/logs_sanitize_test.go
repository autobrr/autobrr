package http

import (
	"io/ioutil"
	"os"
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
	}

	for _, testCase := range testCases {
		// Create a temporary file with sample log data
		tmpFile, err := ioutil.TempFile("", "test-log-*.log")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(tmpFile.Name())

		// Write sample log data to the temporary file
		_, err = tmpFile.WriteString(testCase.input)
		if err != nil {
			tmpFile.Close()
			t.Fatal(err)
		}
		err = tmpFile.Close()
		if err != nil {
			t.Fatal(err)
		}

		// Call SanitizeLogFile on the temporary file
		sanitizedTmpFilePath, err := SanitizeLogFile(tmpFile.Name())
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(sanitizedTmpFilePath)

		// Read the content of the sanitized temporary file
		sanitizedData, err := ioutil.ReadFile(sanitizedTmpFilePath)
		if err != nil {
			t.Fatal(err)
		}

		// Check if the sanitized data matches the expected content
		if string(sanitizedData) != testCase.expected {
			t.Errorf("Sanitized data does not match expected data for input: %s\nExpected:\n%s\nActual:\n%s", testCase.input, testCase.expected, sanitizedData)
		}
	}
}
