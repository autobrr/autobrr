// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import "strings"

type ircCaseMapping uint8

const (
	ircCaseMappingRFC1459 ircCaseMapping = iota
	ircCaseMappingStrictRFC1459
	ircCaseMappingASCII
)

func parseIRCCaseMapping(value string) (ircCaseMapping, bool) {
	switch strings.ToLower(value) {
	case "rfc1459":
		return ircCaseMappingRFC1459, true
	case "rfc1459-strict", "strict-rfc1459":
		return ircCaseMappingStrictRFC1459, true
	case "ascii":
		return ircCaseMappingASCII, true
	default:
		return ircCaseMappingRFC1459, false
	}
}

func (m ircCaseMapping) fold(identifier string) string {
	folded := []byte(identifier)
	for i, char := range folded {
		switch {
		case char >= 'A' && char <= 'Z':
			folded[i] = char + ('a' - 'A')
		case m != ircCaseMappingASCII && char == '[':
			folded[i] = '{'
		case m != ircCaseMappingASCII && char == ']':
			folded[i] = '}'
		case m != ircCaseMappingASCII && char == '\\':
			folded[i] = '|'
		case m == ircCaseMappingRFC1459 && char == '^':
			folded[i] = '~'
		}
	}

	return string(folded)
}

func (m ircCaseMapping) equal(first, second string) bool {
	return m.fold(first) == m.fold(second)
}

// channelMapKey provides a stable storage key independent of a server's
// CASEMAPPING. Unicode lowercasing preserves the handler's historical behavior;
// server-specific IRC equivalents are resolved by Handler.getChannel.
func channelMapKey(channel string) string {
	return strings.ToLower(channel)
}
