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
)

var validIRCAuthMechanisms = []domain.IRCAuthMechanism{
	domain.IRCAuthMechanismNone,
	domain.IRCAuthMechanismSASLPlain,
	domain.IRCAuthMechanismNickServ,
}

// Secret-typed settings are redacted per field type in definition responses, while stored
// indexer settings are redacted per key name via domain.IsSecretIndexerSetting; the two must
// agree or a saved credential leaks unredacted through one of the API paths.
func TestIndexerYamlSecretSettings(t *testing.T) {
	t.Parallel()
	s := &Service{definitions: map[string]domain.IndexerDefinition{}}
	err := s.LoadIndexerDefinitions()
	assert.NoError(t, err)

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
	assert.NoError(t, err)

	for _, d := range s.definitions {
		if d.IRC == nil {
			continue
		}

		t.Run(d.Name, func(t *testing.T) {
			if d.IRC.Auth != nil {
				assert.Contains(t, validIRCAuthMechanisms, d.IRC.Auth.Mechanism, "invalid irc auth mechanism %q", d.IRC.Auth.Mechanism)
			}

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

// TestDefinitionIRCAuthMechanism verifies a definition-declared auth mechanism
// survives YAML decoding in both formats, including the v1 compatibility
// conversion, since the wizard reads it from the served definition.
func TestDefinitionIRCAuthMechanism(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		yaml string
		want domain.IRCAuthMechanism
	}{
		{
			name: "v1 declared",
			yaml: `---
name: TestTracker
identifier: testtracker
implementation: irc
irc:
  network: Test.Net
  server: irc.test.net
  port: 6697
  tls: true
  auth:
    mechanism: NICKSERV
`,
			want: domain.IRCAuthMechanismNickServ,
		},
		{
			name: "v2 declared",
			yaml: `---
version: 2
name: TestTracker
identifier: testtracker
implementation: irc
irc:
  network: Test.Net
  server: irc.test.net
  port: 6697
  tls: true
  auth:
    mechanism: NONE
`,
			want: domain.IRCAuthMechanismNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "testtracker.yaml")
			if err := os.WriteFile(file, []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}

			d, err := OpenAndProcessDefinition(file)
			assert.NoError(t, err)
			assert.NotNil(t, d.IRC)
			assert.NotNil(t, d.IRC.Auth)
			assert.Equal(t, tt.want, d.IRC.Auth.Mechanism)
		})
	}
}

// TestDefinitionIRCAuthAbsent pins the default: no auth block decodes to a nil
// Auth, which is what keeps the wizard on its historical SASL_PLAIN fallback.
func TestDefinitionIRCAuthAbsent(t *testing.T) {
	t.Parallel()

	yaml := `---
name: TestTracker
identifier: testtracker
implementation: irc
irc:
  network: Test.Net
  server: irc.test.net
  port: 6697
  tls: true
`
	file := filepath.Join(t.TempDir(), "testtracker.yaml")
	if err := os.WriteFile(file, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	d, err := OpenAndProcessDefinition(file)
	assert.NoError(t, err)
	assert.NotNil(t, d.IRC)
	assert.Nil(t, d.IRC.Auth)
}
