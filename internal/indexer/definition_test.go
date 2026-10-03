// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Secret-typed settings are redacted per field type in definition responses, while stored
// indexer settings are redacted per key name via domain.IsSecretIndexerSetting; the two must
// agree or a saved credential leaks unredacted through one of the API paths.
func TestIndexerYamlSecretSettings(t *testing.T) {
	t.Parallel()
	s := &Service{definitions: map[string]domain.IndexerDefinition{}}
	err := s.LoadIndexerDefinitions()
	require.NoError(t, err)

	for _, d := range s.definitions {
		for _, setting := range d.Settings {
			secretType := strings.EqualFold(string(setting.Type), "secret")
			assert.Equal(t, secretType, domain.IsSecretIndexerSetting(setting.Name),
				"definition %s setting %s: type '%s' and domain.IsSecretIndexerSetting disagree", d.Identifier, setting.Name, setting.Type)
		}
	}
}

func TestIndexerYamlExpectations(t *testing.T) {
	t.Parallel()
	s := &Service{definitions: map[string]domain.IndexerDefinition{}}
	err := s.LoadIndexerDefinitions()
	require.NoError(t, err)

	for _, d := range s.definitions {
		if d.IRC == nil {
			continue
		}

		t.Run(d.Name, func(t *testing.T) {
			for channelName, channel := range d.IRC.ChannelsMap {
				for _, parseLine := range channel.Parse.Lines {
					for _, test := range parseLine.Tests {
						parseOutput := map[string]string{}
						ok, parseErr := parseLine.ParseLine(parseOutput, test.Line, parseLine.Ignore)
						assert.True(t, ok, "error parsing %s - %s: %s", channelName, test.Line, parseErr)
						assert.NoError(t, parseErr, "error parsing %s - %s", channelName, test.Line)
						assert.Equal(t, test.Expect, parseOutput, "error parsing %s - %s", channelName, test.Line)
					}
				}
			}
		})
	}
}

func TestDefinitionIRCAuth(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		mechanism string
		want      domain.IRCAuthMechanism
		wantErr   bool
	}{
		{name: "v1 nickserv", mechanism: "NICKSERV", want: domain.IRCAuthMechanismNickServ},
		{name: "v2 none", version: "version: 2\n", mechanism: "NONE", want: domain.IRCAuthMechanismNone},
		{name: "unknown", version: "version: 2\n", mechanism: "TYPO", wantErr: true},
		{name: "empty", version: "version: 2\n", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "definition.yaml")
			data := "---\n" + tt.version + `name: Test
identifier: test
implementation: irc
irc:
  network: Test
  server: irc.test
  port: 6697
  tls: true
  auth:
    mechanism: ` + tt.mechanism + "\n"
			require.NoError(t, os.WriteFile(file, []byte(data), 0o644))

			definition, err := OpenAndProcessDefinition(file)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, definition.IRC)
			require.NotNil(t, definition.IRC.Auth)
			assert.Equal(t, tt.want, definition.IRC.Auth.Mechanism)
		})
	}
}

func TestDefinitionIRCSizeUnits(t *testing.T) {
	v1 := `name: Test
identifier: test
implementation: irc
irc:
  network: Test
  server: irc.test
  port: 6697
  channels:
    - "#announce"
  parse:
    type: single
    sizeunits: `
	v2 := `version: 2
name: Test
identifier: test
implementation: irc
irc:
  network: Test
  server: irc.test
  port: 6697
  channels:
    - name: "#announce"
      parse:
        type: single
        sizeunits: `

	tests := []struct {
		name      string
		data      string
		sizeUnits string
		wantErr   bool
	}{
		{name: "v1 binary", data: v1, sizeUnits: "binary"},
		{name: "v1 unknown", data: v1, sizeUnits: "binray", wantErr: true},
		{name: "v2 default", data: v2, sizeUnits: `""`},
		{name: "v2 binary", data: v2, sizeUnits: "binary"},
		{name: "v2 wrong case", data: v2, sizeUnits: "Binary", wantErr: true},
		{name: "v2 decimal", data: v2, sizeUnits: "decimal", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "definition.yaml")
			require.NoError(t, os.WriteFile(file, []byte(tt.data+tt.sizeUnits+"\n"), 0o644))

			definition, err := OpenAndProcessDefinition(file)
			if tt.wantErr {
				require.ErrorContains(t, err, "invalid size units")
				return
			}

			require.NoError(t, err)
			require.NotNil(t, definition.IRC)
			require.Len(t, definition.IRC.Channels, 1)
			assert.Equal(t, strings.Trim(tt.sizeUnits, `"`), definition.IRC.Channels[0].Parse.SizeUnits.String())
		})
	}
}

func TestBundledDefinitionIRCAuth(t *testing.T) {
	s := &Service{definitions: map[string]domain.IndexerDefinition{}}
	require.NoError(t, s.LoadIndexerDefinitions())

	for _, identifier := range []string{
		"aither", "animebytes", "brks", "docspedia", "funfile", "sharewood", "superbits", "torrentleech", "zenith",
	} {
		definition := s.definitions[identifier]
		require.NotNil(t, definition.IRC, identifier)
		require.NotNil(t, definition.IRC.Auth, identifier)
		assert.Equal(t, domain.IRCAuthMechanismNickServ, definition.IRC.Auth.Mechanism, identifier)
	}

	for _, identifier := range []string{"alpharatio"} {
		definition := s.definitions[identifier]
		require.NotNil(t, definition.IRC, identifier)
		assert.Nil(t, definition.IRC.Auth, identifier)
	}
}
