// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package logger

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sanitize(t *testing.T, line string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "autobrr.log")
	require.NoError(t, os.WriteFile(path, []byte(line+"\n"), 0o600))

	var out bytes.Buffer
	require.NoError(t, SanitizeLogFile(path, &out))

	return out.String()
}

// These are the shapes zerolog actually writes to the log file. The redaction
// used to be gated on a handful of module names, so anything logged by another
// module went out untouched.
func TestSanitizeLogFile_JSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "secret field is redacted regardless of module",
			input:    `{"level":"trace","module":"api","api_key":"s3cr3t","message":"cache hit"}`,
			expected: `{"level":"trace","module":"api","api_key":"REDACTED","message":"cache hit"}`,
		},
		{
			name:     "secret nested in an object dump",
			input:    `{"level":"debug","module":"filter","filter_data":{"name":"tv","api_key":"s3cr3t"},"message":"m"}`,
			expected: `{"level":"debug","module":"filter","filter_data":{"name":"tv","api_key":"REDACTED"},"message":"m"}`,
		},
		{
			name:     "secret nested in an array of objects",
			input:    `{"level":"debug","module":"http","clients":[{"name":"qbit","password":"s3cr3t"}],"message":"m"}`,
			expected: `{"level":"debug","module":"http","clients":[{"name":"qbit","password":"REDACTED"}],"message":"m"}`,
		},
		{
			name:     "secret key holding an object is dropped whole",
			input:    `{"level":"debug","module":"irc","auth":{"account":"bob","password":"s3cr3t"},"cookie":{"a":"b"},"message":"m"}`,
			expected: `{"level":"debug","module":"irc","auth":{"account":"bob","password":"REDACTED"},"cookie":"REDACTED","message":"m"}`,
		},
		{
			name:     "passkey in a url from a module that was never covered",
			input:    `{"level":"debug","module":"list","url":"https://x.org/rss?passkey=s3cr3t&i=1","message":"m"}`,
			expected: `{"level":"debug","module":"list","url":"https://x.org/rss?passkey=REDACTED&i=1","message":"m"}`,
		},
		{
			name:     "basic auth in a url",
			input:    `{"level":"error","module":"notification","host":"https://bob:s3cr3t@x.org","message":"m"}`,
			expected: `{"level":"error","module":"notification","host":"https://REDACTED_USER:REDACTED_PW@x.org","message":"m"}`,
		},
		{
			name:     "escaped json payload inside a value",
			input:    `{"level":"trace","module":"indexer","payload":"{\"apikey\":\"s3cr3t\"}","message":"m"}`,
			expected: `{"level":"trace","module":"indexer","payload":"{\"apikey\":\"REDACTED\"}","message":"m"}`,
		},
		{
			name:     "numbers booleans and null survive unchanged",
			input:    `{"level":"debug","module":"feed","count":42,"ratio":10.5,"ok":true,"err":null,"message":"m"}`,
			expected: `{"level":"debug","module":"feed","count":42,"ratio":10.5,"ok":true,"err":null,"message":"m"}`,
		},
		{
			name:     "non secret values are left alone",
			input:    `{"level":"info","module":"irc","network":"irc.example.org","channel":"#announce","message":"joined channel"}`,
			expected: `{"level":"info","module":"irc","network":"irc.example.org","channel":"#announce","message":"joined channel"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected+"\n", sanitize(t, tt.input))
		})
	}
}

// Every invite_command default in internal/indexer/definitions has to redact
// the credential, wherever it sits in the command.
func TestSanitizeLogFile_IRCInviteCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		command  string
		expected string
	}{
		{"bithumen !invite s3cr3t", "bithumen !invite REDACTED"},
		{"BJAnnounce invite s3cr3t", "BJAnnounce invite REDACTED"},
		{"Drone enter #red-announce bob s3cr3t", "Drone enter #red-announce bob REDACTED"},
		{"ENDOR !invite bob s3cr3t", "ENDOR !invite bob REDACTED"},
		{"erica letmeinannounce bob s3cr3t", "erica letmeinannounce bob REDACTED"},
		{"Fapster !auth s3cr3t", "Fapster !auth REDACTED"},
		{"HD-Bot enter #announce bob s3cr3t", "HD-Bot enter #announce bob REDACTED"},
		{"HeBoT !invite s3cr3t", "HeBoT !invite REDACTED"},
		{"hermes enter #announce bob s3cr3t", "hermes enter #announce bob REDACTED"},
		{"Hummingbird ENTER bob s3cr3t #ptp-announce", "Hummingbird ENTER bob REDACTED #ptp-announce"},
		{"immortal invite bob s3cr3t", "immortal invite bob REDACTED"},
		{"LiMEY_ !invite s3cr3t bob", "LiMEY_ !invite REDACTED bob"},
		{"midgards announce s3cr3t", "midgards announce REDACTED"},
		{"Millie announce s3cr3t", "Millie announce REDACTED"},
		{"mmt !invite s3cr3t", "mmt !invite REDACTED"},
		{"Muffit bot #nbl-announce bob s3cr3t", "Muffit bot #nbl-announce bob REDACTED"},
		{"NBOT !invite s3cr3t", "NBOT !invite REDACTED"},
		{"OldSchool invite bob s3cr3t", "OldSchool invite bob REDACTED"},
		{"PT-BOT invite s3cr3t", "PT-BOT invite REDACTED"},
		{"RevoTT !invite bob s3cr3t", "RevoTT !invite bob REDACTED"},
		{"Satsuki enter #announce bob s3cr3t", "Satsuki enter #announce bob REDACTED"},
		{"Sauron bot #ant-announce bob s3cr3t", "Sauron bot #ant-announce bob REDACTED"},
		{"SceneHD .invite s3cr3t #announce", "SceneHD .invite REDACTED #announce"},
		{"stonekeeper enter bob s3cr3t #announce", "stonekeeper enter bob REDACTED #announce"},
		{"Synd1c4t3 invite s3cr3t", "Synd1c4t3 invite REDACTED"},
		{"THR key s3cr3t", "THR key REDACTED"},
		{"Vertigo ENTER #GGn-Announce bob s3cr3t", "Vertigo ENTER #GGn-Announce bob REDACTED"},
		{"Voyager autobot bob s3cr3t", "Voyager autobot bob REDACTED"},
		{"vulcan enter #announce bob s3cr3t", "vulcan enter #announce bob REDACTED"},
		{"Yuki enter #sugoi-announce bob s3cr3t", "Yuki enter #sugoi-announce bob REDACTED"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()

			line := `{"level":"trace","module":"irc","message":"` + tt.command + `"}`
			want := `{"level":"trace","module":"irc","message":"` + tt.expected + `"}` + "\n"

			assert.Equal(t, want, sanitize(t, line))
		})
	}
}

func TestSanitizeLogFile_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("final line without a trailing newline is not dropped", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "autobrr.log")
		content := `{"level":"info","module":"irc","message":"first"}` + "\n" +
			`{"level":"info","module":"api","api_key":"s3cr3t","message":"last"}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

		var out bytes.Buffer
		require.NoError(t, SanitizeLogFile(path, &out))

		assert.Contains(t, out.String(), `"message":"first"`)
		assert.Contains(t, out.String(), `"api_key":"REDACTED"`)
		assert.NotContains(t, out.String(), "s3cr3t")
	})

	t.Run("line that is not json still gets scrubbed", func(t *testing.T) {
		t.Parallel()

		out := sanitize(t, `plain text spilled here https://x.org/rss?passkey=s3cr3t`)

		assert.Equal(t, "plain text spilled here https://x.org/rss?passkey=REDACTED\n", out)
	})

	t.Run("truncated json falls back instead of erroring", func(t *testing.T) {
		t.Parallel()

		out := sanitize(t, `{"level":"info","module":"irc","message":"Millie announce s3cr3t`)

		assert.NotContains(t, out, "s3cr3t")
	})

	t.Run("empty file", func(t *testing.T) {
		t.Parallel()

		path := filepath.Join(t.TempDir(), "autobrr.log")
		require.NoError(t, os.WriteFile(path, nil, 0o600))

		var out bytes.Buffer
		require.NoError(t, SanitizeLogFile(path, &out))
		assert.Empty(t, out.String())
	})
}

// Anything that is not exactly one well formed json object has to fall back to
// the text rules rather than be rewritten to whatever happened to parse.
func TestSanitizeLogFile_MalformedLines(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trailing junk after the object is kept and scrubbed",
			input: `{"level":"info","module":"irc","message":"hi"} oops passkey=s3cr3t`,
			want:  `{"level":"info","module":"irc","message":"hi"} oops passkey=REDACTED`,
		},
		{
			name:  "two objects on one line",
			input: `{"level":"info","module":"irc","message":"hi"}{"level":"info","api_key":"s3cr3t"}`,
			want:  `{"level":"info","module":"irc","message":"hi"}{"level":"info","api_key":"REDACTED"}`,
		},
		{
			name:  "json array rather than an object",
			input: `["passkey=s3cr3t"]`,
			want:  `["passkey=REDACTED"]`,
		},
		{
			name:  "binary garbage",
			input: "\xff\xfe passkey=s3cr3t",
			want:  "\xff\xfe passkey=REDACTED",
		},
		{
			name:  "blank line is preserved",
			input: "   ",
			want:  "   ",
		},
		{
			name:  "empty object",
			input: `{}`,
			want:  `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want+"\n", sanitize(t, tt.input))
		})
	}
}

// A corrupt line must not be able to drive the rewriter into unbounded
// recursion; encoding/json rejects the line first and the text rules take over.
func TestSanitizeLogFile_DeeplyNested(t *testing.T) {
	t.Parallel()

	for _, depth := range []int{100, 5_000, 200_000} {
		line := `{"a":` + strings.Repeat("[", depth) + strings.Repeat("]", depth) + `,"api_key":"s3cr3t"}`

		out := sanitize(t, line)

		assert.NotContains(t, out, "s3cr3t", "depth %d", depth)
	}
}
