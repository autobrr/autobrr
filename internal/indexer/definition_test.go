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
			if parseLine.Expectations == nil {
				continue
			}

			es := *parseLine.Expectations
			assert.Equal(t, len(es), len(parseLine.Test), "if expectations are present there must be one for each test line")

			for i, testLine := range parseLine.Test {

				expectation := es[i]

				tmpVars := map[string]string{}
				ParseLine(nil, parseLine.Pattern, parseLine.Vars, tmpVars, testLine, parseLine.Ignore)

				assert.Equal(t, expectation, tmpVars, "error in expectation %d", i)
			}
		}
	}
}
