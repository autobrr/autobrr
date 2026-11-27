// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var (
	// replaceRegexp replaces Unicode punctuation, symbols, and separators with wildcards.
	replaceRegexp      = regexp.MustCompile(`[\p{P}\p{S}\p{Z}]`)
	questionmarkRegexp = regexp.MustCompile(`[?]{2,}`)
	// parentheticalRegexp strips parenthesized tokens such as (2024) or (US).
	parentheticalRegexp  = regexp.MustCompile(`\(\S+\)`)
	parenthesesEndRegexp = regexp.MustCompile(`\)$`)

	apostropheReplacer = strings.NewReplacer("'", "", "´", "", "`", "", "‘", "", "’", "")
)

// foldLatin returns s with Latin diacritics stripped and non-decomposing Latin letters folded to ASCII.
// Non-Latin scripts (such as Japanese kana with dakuten) are left untouched.
func foldLatin(s string) string {
	ascii := true
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			ascii = false
			break
		}
	}
	if ascii {
		return s
	}

	normalized := s
	if !norm.NFC.IsNormalString(s) {
		normalized = norm.NFC.String(s)
	}
	var b strings.Builder
	b.Grow(len(normalized))

	prevLatin := false
	for _, r := range normalized {
		if unicode.Is(unicode.Mn, r) {
			if !prevLatin {
				b.WriteRune(r)
			}
			continue
		}

		prevLatin = unicode.Is(unicode.Latin, r)
		if r < 0x80 || !prevLatin {
			b.WriteRune(r)
			continue
		}

		for _, dr := range norm.NFD.String(string(r)) {
			if unicode.Is(unicode.Mn, dr) {
				continue
			}

			switch dr {
			case 'Ø':
				b.WriteByte('O')
			case 'ø':
				b.WriteByte('o')
			case 'Ł':
				b.WriteByte('L')
			case 'ł':
				b.WriteByte('l')
			case 'Đ', 'Ð':
				b.WriteByte('D')
			case 'đ', 'ð':
				b.WriteByte('d')
			case 'ß':
				b.WriteString("ss")
			case 'ẞ':
				b.WriteString("SS")
			case 'Æ':
				b.WriteString("AE")
			case 'æ':
				b.WriteString("ae")
			case 'Œ':
				b.WriteString("OE")
			case 'œ':
				b.WriteString("oe")
			case 'Þ':
				b.WriteString("TH")
			case 'þ':
				b.WriteString("th")
			case 'ı':
				b.WriteByte('i')
			case 'ª':
				b.WriteByte('a')
			case 'º':
				b.WriteByte('o')
			default:
				b.WriteRune(dr)
			}
		}
	}

	return b.String()
}

// generateVariations returns variations of the title with optionally removing apostrophes and info in parentheses.
func generateVariations(title string, removeApostrophes, removeParenthetical bool) []string {
	var variation string

	if removeParenthetical {
		variation = parentheticalRegexp.ReplaceAllString(title, "")
		variation = strings.TrimRight(variation, " ")
	} else {
		variation = parenthesesEndRegexp.ReplaceAllString(title, "?")
	}

	if removeApostrophes {
		variation = apostropheReplacer.Replace(variation)
	}
	variation = replaceRegexp.ReplaceAllString(variation, "?")
	variation = questionmarkRegexp.ReplaceAllString(variation, "*")

	return []string{
		variation,
		strings.TrimRight(variation, "?* "),
	}
}

func titleVariations(title string) []string {
	if replaceRegexp.ReplaceAllString(title, "") == "" {
		return []string{title}
	}

	variants := []string{title}
	if folded := foldLatin(title); folded != title {
		variants = append(variants, folded)
	}

	var result []string
	for _, v := range variants {
		result = slices.Concat(
			result,
			generateVariations(v, false, false),
			generateVariations(v, true, false),
			generateVariations(v, false, true),
			generateVariations(v, true, true),
		)
	}

	return result
}

func processTitle(title string, matchRelease bool) []string {
	if strings.TrimSpace(title) == "" {
		return nil
	}

	t := NewTitleSet()
	for _, titleVar := range titleVariations(title) {
		t.Add(titleVar, matchRelease)
	}

	return t.Titles()
}

type Titles struct {
	tm map[string]struct{}

	matchReleases bool
}

func NewTitleSet() *Titles {
	return &Titles{
		tm: map[string]struct{}{},
	}
}

func (ts *Titles) Len() int {
	return len(ts.tm)
}

func isOverbroadWildcard(title string) bool {
	if !strings.ContainsRune(title, '*') {
		return false
	}

	literals := 0
	for _, r := range title {
		if r != '*' && r != '?' && r != ' ' {
			literals++
			if literals >= 2 {
				return false
			}
		}
	}

	return true
}

func (ts *Titles) Add(title string, matchRelease bool) {
	if strings.Trim(title, "?* ") == "" || isOverbroadWildcard(title) {
		return
	}

	if matchRelease {
		title = strings.Trim(title, "?* ")
		title = fmt.Sprintf("*%v*", title)
		if isOverbroadWildcard(title) {
			return
		}
	}

	ts.tm[title] = struct{}{}
}

func (ts *Titles) add(title string) {
	if strings.Trim(title, "?* ") == "" || isOverbroadWildcard(title) {
		return
	}

	if ts.matchReleases {
		title = strings.Trim(title, "?* ")
		if isOverbroadWildcard("*" + title + "*") {
			return
		}
	}

	ts.tm[title] = struct{}{}
}

// AddTitle adds normalized matching variations of title to the set.
func (ts *Titles) AddTitle(title string) bool {
	if strings.TrimSpace(title) == "" {
		return false
	}

	for _, titleVar := range titleVariations(title) {
		ts.add(titleVar)
	}

	return true
}

func (ts *Titles) FilterString() string {
	keys := slices.Sorted(maps.Keys(ts.tm))
	if len(keys) == 0 {
		return ""
	}

	if ts.matchReleases {
		for i, key := range keys {
			keys[i] = "*" + key + "*"
		}

		return strings.Join(keys, ",")
	}

	return strings.Join(keys, ",")
}

func (ts *Titles) Titles() []string {
	return slices.Collect(maps.Keys(ts.tm))
}
