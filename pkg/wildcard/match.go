// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package wildcard

import (
	"fmt"
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
	if len(pattern) == 0 {
		return pattern == name
	} else if pattern == "*" {
		return true
	}

	return deepMatchRune(name, cleanForRegex(pattern, simple), simple)
}

func MatchSliceSimple(pattern []string, name string) (matched bool) {
	return matchSlice(pattern, name, true)
}

func MatchSlice(pattern []string, name string) (matched bool) {
	return matchSlice(pattern, name, false)
}

func matchSlice(pattern []string, name string, simple bool) (matched bool) {
	var build strings.Builder
	for i := 0; i < len(pattern); i++ {
		if len(pattern[i]) == 0 {
			continue
		}

		if build.Len() != 0 {
			build.WriteString("|")
		}

		build.WriteString(cleanForRegex(pattern[i], simple))
	}

	if build.Len() == 0 {
		return len(name) == 0
	}

	return deepMatchRune(name, build.String(), simple)
}

var convSimple = regexp.MustCompile(regexp.QuoteMeta(`\*`))
var convWildChar = regexp.MustCompile(regexp.QuoteMeta(`\?`))

func cleanForRegex(pattern string, simple bool) string {
	pattern = regexp.QuoteMeta(pattern)
	if strings.Contains(pattern, "*") {
		pattern = convSimple.ReplaceAllLiteralString(pattern, ".*")
	}

	if !simple && strings.Contains(pattern, "?") {
		pattern = convWildChar.ReplaceAllLiteralString(pattern, ".")
	}

	return `^` + pattern + `$`
}

func deepMatchRune(str, pattern string, simple bool) bool {
	fmt.Printf("")
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
