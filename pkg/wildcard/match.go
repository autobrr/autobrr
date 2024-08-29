// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package wildcard

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
	return deepMatchRune([]rune(name), []rune(pattern), true)
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
	return deepMatchRune([]rune(name), []rune(pattern), false)
}

func deepMatchRune(str, pattern []rune, simple bool) bool {
	k, i := 0, 0
	for ; i < len(pattern) && k < len(str); i++ {
		switch pattern[i] {
		case '*':
			if i == len(pattern)-1 {
				return true
			}

			var val rune
			for i++; i < len(pattern); i++ {
				val = pattern[i]
				if !simple && val == '?' || val == '*' {
					continue
				}

				i--
				break
			}

			if i >= len(pattern)-1 && val == '*' {
				return true
			}

			for ; k < len(str); k++ {
				if str[k] != val {
					continue
				}

				break
			}

			if k == len(str) {
				return false
			}
		case '?':
			if simple && pattern[i] != str[k] {
				return false
			}

			k++
			continue
		case str[k]:
			k++
			continue
		default:
			return false
		}
	}

	return k == len(str) && (i == len(pattern) || i == len(pattern)-1 && pattern[len(pattern)-1] == '*')
}
