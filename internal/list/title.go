// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"fmt"
	"regexp"
	"strings"
)

// Regex patterns
// https://www.regular-expressions.info/unicode.html#category
// https://www.ncbi.nlm.nih.gov/staff/beck/charents/hex.html
var (
	replaceRegexp        = regexp.MustCompile(`[\p{P}\p{Z}\x{00C0}-\x{017E}\x{00AE}]`)
	questionmarkRegexp   = regexp.MustCompile(`[?]{2,}`)
	regionCodeRegexp     = regexp.MustCompile(`\(.{2,3}\)$`) // TODO should this only be 2 or 3 letters
	parenthesesEndRegexp = regexp.MustCompile(`\)$`)
)

func processTitle(title string, matchRelease bool) []string {
	// Checking if the title is empty.
	if strings.TrimSpace(title) == "" {
		return nil
	}

	// cleans year like (2020) from arr title
	//var re = regexp.MustCompile(`(?m)\s(\(\d+\))`)
	//title = re.ReplaceAllString(title, "")

	t := NewTitleSlice()

	if replaceRegexp.ReplaceAllString(title, "") == "" {
		t.Add(title, matchRelease)
	} else {
		// title with all non-alphanumeric characters replaced by "?"
		apostropheTitle := parenthesesEndRegexp.ReplaceAllString(title, "?")
		apostropheTitle = replaceRegexp.ReplaceAllString(apostropheTitle, "?")
		apostropheTitle = questionmarkRegexp.ReplaceAllString(apostropheTitle, "*")

		t.Add(apostropheTitle, matchRelease)
		t.Add(strings.TrimRight(apostropheTitle, "?* "), matchRelease)

		// title with apostrophes removed and all non-alphanumeric characters replaced by "?"
		noApostropheTitle := parenthesesEndRegexp.ReplaceAllString(title, "?")
		noApostropheTitle = strings.ReplaceAll(noApostropheTitle, "'", "")
		noApostropheTitle = replaceRegexp.ReplaceAllString(noApostropheTitle, "?")
		noApostropheTitle = questionmarkRegexp.ReplaceAllString(noApostropheTitle, "*")

		t.Add(noApostropheTitle, matchRelease)
		t.Add(strings.TrimRight(noApostropheTitle, "?* "), matchRelease)

		// title with regions in parentheses removed and all non-alphanumeric characters replaced by "?"
		removedRegionCodeApostrophe := regionCodeRegexp.ReplaceAllString(title, "")
		removedRegionCodeApostrophe = strings.TrimRight(removedRegionCodeApostrophe, " ")
		removedRegionCodeApostrophe = replaceRegexp.ReplaceAllString(removedRegionCodeApostrophe, "?")
		removedRegionCodeApostrophe = questionmarkRegexp.ReplaceAllString(removedRegionCodeApostrophe, "*")

		t.Add(removedRegionCodeApostrophe, matchRelease)
		t.Add(strings.TrimRight(removedRegionCodeApostrophe, "?* "), matchRelease)

		// title with regions in parentheses and apostrophes removed and all non-alphanumeric characters replaced by "?"
		removedRegionCodeNoApostrophe := regionCodeRegexp.ReplaceAllString(title, "")
		removedRegionCodeNoApostrophe = strings.TrimRight(removedRegionCodeNoApostrophe, " ")
		removedRegionCodeNoApostrophe = strings.ReplaceAll(removedRegionCodeNoApostrophe, "'", "")
		removedRegionCodeNoApostrophe = replaceRegexp.ReplaceAllString(removedRegionCodeNoApostrophe, "?")
		removedRegionCodeNoApostrophe = questionmarkRegexp.ReplaceAllString(removedRegionCodeNoApostrophe, "*")

		t.Add(removedRegionCodeNoApostrophe, matchRelease)
		t.Add(strings.TrimRight(removedRegionCodeNoApostrophe, "?* "), matchRelease)
	}

	return t.Titles()
}

type Titles struct {
	tm map[string]struct{}
}

func NewTitleSlice() *Titles {
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
		title = strings.Trim(title, "?")
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
