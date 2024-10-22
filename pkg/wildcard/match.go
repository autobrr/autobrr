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
	if pattern == "" { //
		return name == ""

	} else if pattern == "*" { // *
		return true

	} else if !simple && pattern == "?" { // ?
		return len(name) == 1

	} else if idx := strings.IndexAny(pattern, "*?"); idx == -1 || (simple && pattern[idx] == '?' && !strings.Contains(pattern, "*")) { // egg
		return name == pattern

	} else if idx == len(pattern)-1 && pattern[idx] == '*' { // egg*
		return strings.HasPrefix(name, pattern[:idx-1])

		// egg?bert*
	} else if wildEnd := pattern[len(pattern)-1] == '*'; !simple && ((wildEnd && strings.Count(pattern, "*") == 1) || (len(pattern) == len(name) && !strings.Contains(pattern, "*"))) { // egg?bert?
		return matchComplex(name, pattern, wildEnd)

		// *egg*
		// simple is fine, if not we need to check for ? and skip if so.
	} else if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") && (simple || (!simple && !strings.Contains(pattern, "?"))) && strings.Count(pattern, "*") == 2 { // make sure that we have no other wildcards.
		return strings.Contains(name, pattern[1:len(pattern)-1])
	}

	return deepMatchRune(name, pattern, simple, pattern, false)
}

func matchComplex(name, pattern string, wildEnd bool) bool {
	base := 0
	consumedPattern := 0

	for base < len(name) && consumedPattern < len(pattern) {
		i := strings.IndexRune(pattern[base:], '?')
		if i == -1 {
			if (wildEnd && !strings.HasPrefix(name[base:], pattern[base:len(pattern)-1])) || // egg*
				(!wildEnd && name[base:] != pattern[base:]) { // egg
				break
			}

			base = len(name)
			consumedPattern = len(pattern)
			continue
		}

		offset := base + i
		if len(name) < offset || name[base:offset] != pattern[base:offset] {
			break
		}

		base = offset + 1
		consumedPattern = base
	}

	return base == len(name) && consumedPattern == len(pattern)
}

func MatchSliceSimple(pattern []string, name string) (matched bool) {
	return matchSlice(pattern, name, true)
}

func MatchSlice(pattern []string, name string) (matched bool) {
	return matchSlice(pattern, name, false)
}

func matchSlice(pattern []string, name string, simple bool) (matched bool) {
	for i := 0; i < len(pattern); i++ {
		if match(pattern[i], name, simple) {
			return true
		}
	}

	return false
}

// go 1.23 seems to still be too slow for regex.
// the single case now skips almost all allocations.
/* func matchSlice(pattern []string, name string, simple bool) (matched bool) {
	var build strings.Builder
	{
		grow := 0
		for i := 0; i < len(pattern); i++ {
			grow += len(pattern[i]) + 6 // ^\?\*$
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

	return deepMatchRune(name, build.String(), simple, build.String(), true)
} */

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
	return `(?m)^` + regexp.QuoteMeta(pattern) + `$`
}

func deepMatchRune(str, pattern string, simple bool, original string, bulk bool) bool {
	salt := ""
	if simple {
		salt = "//" // invalid regex.
	}

	user, ok := regexcache.FindOriginal(original + salt)
	if !ok {
		if !bulk {
			pattern = prepForRegex(pattern)
		}

		pattern = cleanForRegex(pattern, simple)
		{
			var err error
			user, err = regexcache.Compile(pattern)
			if err != nil {
				log.Error().Err(err).Msgf("deepMatchRune: unable to parse %q", pattern)
				return false
			}
		}

		regexcache.SubmitOriginal(original+salt, user)
	}

	user, err := regexcache.Compile(pattern)
	if err != nil {
		log.Error().Err(err).Msgf("deepMatchRune: unable to parse %q", pattern)
		return false
	}

	return user.MatchString(str)
}
