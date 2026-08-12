// Copyright (c) 2021-2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/stretchr/testify/assert"
)

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
