// Copyright (c) 2021-2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"errors"

	"github.com/autobrr/autobrr/pkg/regexcache"
	"github.com/rs/zerolog"
)

type Logger interface {
	Debug() *zerolog.Event
}

func regExMatch(pattern string, value string) ([]string, error) {
	rxp, err := regexcache.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matches := rxp.FindStringSubmatch(value)
	if matches == nil {
		return nil, nil
	}

	return matches[1:], nil
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

	for i, v := range vars {
		if i+1 > len(rxp) {
			return false, errors.New("too few matches returned for rxp")
		}

		tmpVars[v] = rxp[i]
	}
	return true, nil
}

func parseMatchRegexp(pattern string, tmpVars map[string]string, line string, ignore bool) (bool, error) {
	var re = regexcache.MustCompile(`(?mi)` + pattern)

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
