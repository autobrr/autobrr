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
	if pattern == "" {
		return name == pattern
	}
	if pattern == "*" {
		return true
	}
	// Does only wildcard '*' match.
	return deepMatchRune(name, pattern, true)
}

// Match -  finds whether the text matches/satisfies the pattern string.
// supports  '*' and '?' wildcards in the pattern string.
// unlike path.Match(), considers a path as a flat name space while matching the pattern.
// The difference is illustrated in the example here https://play.golang.org/p/Ega9qgD4Qz .
func Match(pattern, name string) (matched bool) {
	if pattern == "" {
		return name == pattern
	}
	if pattern == "*" {
		return true
	}
	// Does extended wildcard '*' and '?' match.
	return deepMatchRune(name, pattern, false)
}

var convSimple = regexp.MustCompile(regexp.QuoteMeta(`\*`))
var convWildChar = regexp.MustCompile(regexp.QuoteMeta(`\?`))

func deepMatchRune(str, pattern string, simple bool) bool {
	pattern = regexp.QuoteMeta(pattern)
	if strings.Contains(pattern, "*") {
		pattern = convSimple.ReplaceAllLiteralString(pattern, ".*")
	}

	if !simple && strings.Contains(pattern, "?") {
		pattern = convWildChar.ReplaceAllLiteralString(pattern, ".")
	}

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
