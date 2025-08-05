// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var (
	/*
		replaceRegexp replaces various character classes/categories such as
		\p{P} Unicode punctuation category characters
		\p{S} Unicode symbol category characters
		\p{Z) Unicode seperator category characters
		\x{0080}-\x{017F} Unicode block "Latin-1 Supplement" and "Latin Extended-A" characters
		https://www.unicode.org/reports/tr44/#General_Category_Values
		https://www.regular-expressions.info/unicode.html#category
		https://www.compart.com/en/unicode/block/U+0080
		https://www.compart.com/en/unicode/block/U+0100
	*/
	replaceRegexp      = regexp.MustCompile(`[\p{P}\p{S}\p{Z}\x{0080}-\x{017F}]`)
	questionmarkRegexp = regexp.MustCompile(`[?]{2,}`)
	// cleans titles from years and region codes in parentheses, for example (2024) or (US)
	parentheticalRegexp  = regexp.MustCompile(`\(\S+\)`)
	parenthesesEndRegexp = regexp.MustCompile(`\)$`)

	apostropheReplacer = strings.NewReplacer("'", "", "´", "", "`", "", "‘", "", "’", "")
)

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

// yearRegexp = regexp.MustCompile(`\(\d{4}\)$`)
func processTitle(title string, matchRelease bool) []string {
	// Checking if the title is empty.
	if strings.TrimSpace(title) == "" {
		return nil
	}

	// cleans year like (2020) from arr title
	// var re = regexp.MustCompile(`(?m)\s(\(\d+\))`)
	// title = re.ReplaceAllString(title, "")

	t := NewTitleSet()

	if replaceRegexp.ReplaceAllString(title, "") == "" {
		t.Add(title, matchRelease)
	} else {
		titles := slices.Concat(
			// don't remove apostrophes and info in parentheses
			generateVariations(title, false, false),
			// remove apostrophes but don't remove info in parentheses
			generateVariations(title, true, false),
			// don't remove apostrophes but remove info in parentheses
			generateVariations(title, false, true),
			// remove apostrophes and info in parentheses
			generateVariations(title, true, true),
		)

		for _, title := range titles {
			t.Add(title, matchRelease)
		}
	}

	return t.Titles()
}

type Titles struct {
	tm map[string]struct{}
}

func NewTitleSet() *Titles {
	ts := Titles{
		tm: map[string]struct{}{},
	}
	return &ts
}

func (ts *Titles) Add(title string, matchRelease bool) {
	if title == "" || title == "*" {
		return
	}

	if matchRelease {
		title = strings.Trim(title, "?* ")
		title = fmt.Sprintf("*%v*", title)
	}

	_, ok := ts.tm[title]
	if !ok {
		ts.tm[title] = struct{}{}
	}
}

func (ts *Titles) Titles() []string {
	titles := []string{}
	for key := range ts.tm {
		titles = append(titles, key)
	}
	return titles
}
