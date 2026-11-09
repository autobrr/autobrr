// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

var nameKeyExtensions = []string{".mkv", ".mp4", ".avi"}

// NameKey returns the normalized release name that name based duplicate checks compare:
// case, accents, punctuation and separators are ignored, '+' is kept.
func NameKey(releaseName string) string {
	// MustNormalize stops at the first invalid byte or U+FFFD, and IRC lines are not UTF-8 validated
	name := strings.ReplaceAll(strings.ToValidUTF8(releaseName, ""), string(utf8.RuneError), "")
	name = strings.TrimSpace(name)

	for _, ext := range nameKeyExtensions {
		if len(name) > len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext) {
			name = name[:len(name)-len(ext)]
			break
		}
	}

	var b strings.Builder
	for _, r := range MustNormalize(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '+' {
			b.WriteRune(r)
		}
	}

	// UNIT3D style names spell DDP as DD+
	return strings.ReplaceAll(b.String(), "dd+", "ddp")
}

var titleKeyFolds = map[rune]string{'æ': "ae", 'œ': "oe", 'ø': "o", 'ß': "ss", 'đ': "d", 'ð': "d", 'ł': "l", 'þ': "th"}

// TitleKey returns the comparison key for a title: lower case letters and digits without
// accents, plus any trailing run of '!', '?' or symbols.
func TitleKey(title string) string {
	var b strings.Builder
	for _, r := range norm.NFKD.String(title) {
		switch {
		case unicode.Is(unicode.Mn, r):
		case r == '&':
			b.WriteString("and")
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			r = unicode.ToLower(r)
			if folded, ok := titleKeyFolds[r]; ok {
				b.WriteString(folded)
				continue
			}

			b.WriteRune(r)
		}
	}

	// anime marks sequels this way: 'K-On!' and 'K-On!!' are different seasons
	title = strings.TrimSpace(title)
	end := len(title)
	for end > 0 {
		r, size := utf8.DecodeLastRuneInString(title[:end])
		if r != '!' && r != '?' && !unicode.IsSymbol(r) {
			break
		}

		end -= size
	}

	if b.Len() == 0 {
		return ""
	}

	return b.String() + title[end:]
}
