package indexer

import (
	"fmt"
	"regexp"

	"github.com/rs/zerolog"
)

type Logger interface {
	Debug() *zerolog.Event
}

func removeElement(s []string, i int) ([]string, error) {
	// s is [1,2,3,4,5,6], i is 2

	// perform bounds checking first to prevent a panic!
	if i >= len(s) || i < 0 {
		return nil, fmt.Errorf("Index is out of range. Index is %d with slice length %d", i, len(s))
	}

	// This creates a new slice by creating 2 slices from the original:
	// s[:i] -> [1, 2]
	// s[i+1:] -> [4, 5, 6]
	// and joining them together using `append`
	return append(s[:i], s[i+1:]...), nil
}

func regExMatch(pattern string, value string) ([]string, error) {
	rxp, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matches := rxp.FindStringSubmatch(value)
	if matches == nil {
		return nil, nil
	}

	res := make([]string, 0)
	if matches != nil {
		res, err = removeElement(matches, 0)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func ParseLine(logger Logger, pattern string, vars []string, tmpVars map[string]string, line string, ignore bool) (bool, error) {
	if len(vars) > 0 {
		return parseExtract(logger, pattern, vars, tmpVars, line)
	}

	return parseMatchRegexp(pattern, tmpVars, line, ignore)
}

func parseExtract(logger Logger, pattern string, vars []string, tmpVars map[string]string, line string) (bool, error) {
	rxp, err := regExMatch(pattern, line)
	if err != nil {
		logger.Debug().Msgf("did not match expected line: %v", line)
	}

	if rxp == nil {
		return false, nil
	}

	// extract matched
	for i, v := range vars {
		value := ""

		if rxp[i] != "" {
			value = rxp[i]
		}

		tmpVars[v] = value
	}
	return true, nil
}

func parseMatchRegexp(pattern string, tmpVars map[string]string, line string, ignore bool) (bool, error) {
	var re = regexp.MustCompile(`(?mi)` + pattern)

	groupNames := re.SubexpNames()
	for _, match := range re.FindAllStringSubmatch(line, -1) {
		for groupIdx, group := range match {
			// if line should be ignored then lets return
			if ignore {
				return true, nil
			}

			name := groupNames[groupIdx]
			if name == "" {
				name = "raw"
			}
			tmpVars[name] = group
		}
	}

	return true, nil
}
