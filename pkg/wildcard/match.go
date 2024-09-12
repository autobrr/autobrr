// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package wildcard

import (
	"regexp"
	"strings"

	"github.com/autobrr/autobrr/pkg/regexcache"
	"github.com/rs/zerolog/log"
)

// MatchSimple - finds whether the text matches/satisfies the pattern string.
// supports only '*' wildcard in the pattern.
// considers a file system path as a flat name space.
func MatchSimple(pattern, name string) bool {
	return match(pattern, name, true)
}

// Match -  finds whether the text matches/satisfies the pattern string.
// supports  '*' and '?' wildcards in the pattern string.
// unlike path.Match(), considers a path as a flat name space while matching the pattern.
// The difference is illustrated in the example here https://play.golang.org/p/Ega9qgD4Qz .
func Match(pattern, name string) (matched bool) {
	return match(pattern, name, false)
}

func match(pattern, name string, simple bool) (matched bool) {
	if pattern == "" {
		return name == ""
	} else if pattern == "*" {
		return true
	}

	return deepMatchRune(name, cleanForRegex(prepForRegex(pattern), simple), simple)
}

func MatchSliceSimple(pattern []string, name string) (matched bool) {
	return matchSlice(pattern, name, true)
}

func MatchSlice(pattern []string, name string) (matched bool) {
	return matchSlice(pattern, name, false)
}

func matchSlice(pattern []string, name string, simple bool) (matched bool) {
	var build strings.Builder

	if len(pattern) > 32 {
		grow := 0
		for i := 0; i < len(pattern); i++ {
			grow += len(pattern[i]) + 6 /* ^\?\*$ */
		}

		build.Grow(grow)
	}

	for i := 0; i < len(pattern); i++ {
		if pattern[i] == "" {
			continue
		}

		if build.Len() != 0 {
			build.WriteString("|")
		}

		build.WriteString(prepForRegex(pattern[i]))
	}

	if build.Len() == 0 {
		return name == ""
	}

	return deepMatchRune(name, cleanForRegex(build.String(), simple), simple)
}

var convSimple = regexp.QuoteMeta("*")
var convWildChar = regexp.QuoteMeta("?")

func cleanForRegex(pattern string, simple bool) string {
	if strings.Contains(pattern, convSimple) {
		pattern = strings.ReplaceAll(pattern, convSimple, ".*")
	}

	if !simple && strings.Contains(pattern, convWildChar) {
		pattern = strings.ReplaceAll(pattern, convWildChar, ".")
	}

	return pattern
}

func prepForRegex(pattern string) string {
	return `^` + regexp.QuoteMeta(pattern) + `$`
}

func deepMatchRune(str, pattern string, simple bool) bool {
	user, err := regexcache.Compile(pattern)
	if err != nil {
		log.Error().Err(err).Msgf("deepMatchRune: unable to parse %q", pattern)
		return false
	}

	idx := user.FindStringIndex(str)
	if idx == nil {
		return false
	}

	return idx[1] == len(str)
}
