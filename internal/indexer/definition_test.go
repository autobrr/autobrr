package indexer

import (
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestYamlExpectations(t *testing.T) {
	s := &service{definitions: map[string]domain.IndexerDefinition{}}
	err := s.LoadIndexerDefinitions()

	assert.Nil(t, err)

	for _, d := range s.definitions {
		if d.IRC == nil {
			continue
		}
		if d.IRC.Parse == nil {
			continue
		}
		if d.IRC.Parse.Lines == nil {
			continue
		}

		for _, parseLine := range d.IRC.Parse.Lines {
			for _, test := range parseLine.Tests {
				parseOutput := map[string]string{}
				ParseLine(nil, parseLine.Pattern, parseLine.Vars, parseOutput, test.Line, parseLine.Ignore)
				assert.Equal(t, test.Expect, parseOutput, "error parsing %s", test.Line)
			}
		}
	}
}
