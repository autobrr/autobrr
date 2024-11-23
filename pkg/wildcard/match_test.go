// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package wildcard

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMatch - Tests validate the logic of wild card matching.
// `Match` supports '*' (zero or more characters) and '?' (one character) wildcards in typical glob style filtering.
// A '*' in a provided string will not result in matching the strings before and after the '*' of the string provided.
// Sample usage: In resource matching for bucket policy validation.
func TestMatch(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pattern string
		text    string
		matched bool
	}{
		{
			pattern: "The?Simpsons*",
			text:    "The Simpsons S12",
			matched: true,
		},
		{
			pattern: "The?Simpsons*",
			text:    "The.Simpsons.S12",
			matched: true,
		},
		{
			pattern: "The?Simpsons*",
			text:    "The.Simps.S12",
			matched: false,
		},
		{
			pattern: "The?Simp",
			text:    "The.Simps.S12",
			matched: false,
		},
		{
			pattern: "The?Simp",
			text:    "The.Simps.S12",
			matched: false,
		},
		{
			pattern: "The*Simp",
			text:    "The.Simp",
			matched: true,
		},
		{
			pattern: "*tv*",
			text:    "tv",
			matched: true,
		},
		{
			pattern: "t?",
			text:    "tv",
			matched: true,
		},
		{
			pattern: "?",
			text:    "z",
			matched: true,
		},
		{
			pattern: "*EPUB*",
			text:    "Translated (Group) / EPUB",
			matched: true,
		},
		{
			pattern: "*EP?B*",
			text:    "ARG THIS IS A STUPID LONG ONG LONG LONG STRING BEFORE AND AFTER \\n ARG THIS IS A STUPID LONG ONG LONG LONG STRING BEFORE AND AFTER \\n ARG THIS IS A STUPID LONG ONG LONG LONG STRING BEFORE AND AFTER \\n ARG THIS IS A STUPID LONG ONG LONG LONG STRING BEFORE AND AFTER \\n Translated (Group) / EPUB WITH OTHER STUFF ON THE OTHER END ARG THIS IS A STUPID LONG ONG LONG LONG STRING BEFORE AND AFTER \\n ARG THIS IS A STUPID LONG ONG LONG LONG STRING BEFORE AND AFTER \\n ARG THIS IS A STUPID LONG ONG LONG LONG STRING BEFORE AND AFTER \\n ",
			matched: true,
		},
		{
			pattern: "*shift*",
			text:    "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP",
			matched: true,
		},
		{
			pattern: "The God of the Brr*The Power of Brr",
			text:    "The Power of Brr",
			matched: false,
		},
		{
			pattern: "The God of the Brr*The Power of Brr",
			text:    "The God of the Brr",
			matched: false,
		},
		{
			pattern: "The God of the Brr*The Power of Brr",
			text:    "The God of the Brr The Power of Brr",
			matched: true,
		},
		{
			pattern: "The God of the Brr*The Power of Brr",
			text:    "The God of the Brr - The Power of Brr",
			matched: true,
		},
		{
			pattern: "The God of the Brr*The Power of Brr",
			text:    "The God of the BrrThe Power of Brr",
			matched: true,
		},
		{
			pattern: "mysteries?of?the?abandoned*",
			text:    "them",
			matched: false,
		},
		{
			pattern: "t?q*",
			text:    "tam e",
			matched: false,
		},
		{
			pattern: "Hard?Quiz*",
			text:    "HardX 24 10 12 Ella Reese XXX 1080p MP4-WRB",
			matched: false,
		},
		{
			pattern: "Hard?Quiz*",
			text:    "HardX",
			matched: false,
		},
		{
			pattern: "T?Q*",
			text:    "T?Q",
			matched: true,
		},
		{
			pattern: "Lee*",
			text:    "Let Go",
			matched: false,
		},
		{
			pattern: "*black?metal*",
			text:    "   ||  Artist......: Vredehammer                                        ||\n   ||  Album.......: Mintaka                                            ||\n   ||  Year........: 2013                                               ||\n   ||                                                                   ||\n   ||  Genre.......: black metal                                        ||\n   ||  Label.......: Indie Recordings                                   ||\n   ||                                                                   ||\n   ||  Source......: FLAC/WEB (16bit)                                   ||\n   ||  Encoder.....: libFLAC                                            ||\n   ||  Bitrate.....: 948 kbps avg.                                      ||\n   ||  F.Rate......: 44.1kHz                                            ||\n   ||                                                                   ||\n   ||  Playtime....: 00:19:27 / 138.70MB                                ||\n   ||  R.Date......: 2024-10-22                                         ||\n   ||  S.Date......: 2013-03-27                                         ||\n   ||                                                                   ||\n   ||                                                                   ||\n   ||  01. The King Has Risen                                  3:53     ||\n   ||  02. H├╕ster av sjeler                                    4:17     ||\n   ||  03. Mintaka                                             4:10     ||\n   ||  04. Ditt siste aandedrag                                7:07     ||\n   ||                                                                   ||\n   ||                                                                   ||\n   ||  Vredehammer combines aggressive guitars and Norse melodies.      ||\n   ",
			matched: true,
		},
	}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("match: %d", idx), func(t *testing.T) {
			actualResult := Match(tt.pattern, tt.text)
			assert.Equal(t, tt.matched, actualResult)

		})
	}
}

func TestMatchSimple(t *testing.T) {
	t.Parallel()
	tests := []struct {
		pattern string
		name    string
		want    bool
	}{
		{"", "", true},
		{"*", "test", true},
		{"t*t", "test", true},
		{"t*t", "tost", true},
		{"t?st", "test", false},
		{"t?st", "tast", false},
		{"test", "test", true},
		{"*te?t*", "test", false},
		{"*test*", "test", true},
		{"test", "toast", false},
		{"", "non-empty", false},
		{"*", "", true},
		{"te*t", "test", true},
		{"te*", "te", true},
		{"te*", "ten", true},
		{"?est", "test", false},
		{"best", "best", true},
	}

	for _, tt := range tests {
		if got := MatchSimple(tt.pattern, tt.name); got != tt.want {
			t.Errorf("MatchSimple(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}

func TestMatchSliceSimple(t *testing.T) {
	t.Parallel()
	tests := []struct {
		patterns []string
		name     string
		want     bool
	}{
		{[]string{"*", "test"}, "test", true},
		{[]string{"te?t", "tost", "random"}, "tost", true},
		{[]string{"te?t", "t?s?", "random"}, "tost", false},
		{[]string{"*st", "n?st", "l*st"}, "list", true},
		{[]string{"?", "?*", "?**"}, "t", false},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "test", false},
		{[]string{"*"}, "any", true},
		{[]string{"abc", "def", "ghi"}, "ghi", true},
		{[]string{"abc", "def", "ghi"}, "xyz", false},
		{[]string{"abc*", "def*", "ghi*"}, "ghi-test", true},
	}

	for _, tt := range tests {
		if got := MatchSliceSimple(tt.patterns, tt.name); got != tt.want {
			t.Errorf("MatchSliceSimple(%v, %q) = %v, want %v", tt.patterns, tt.name, got, tt.want)
		}
	}
}

func TestMatchSlice(t *testing.T) {
	t.Parallel()
	tests := []struct {
		patterns []string
		name     string
		want     bool
	}{
		{[]string{"*", "test", "t?st"}, "test", true},
		{[]string{"te?t", "t?st", "random"}, "tost", true},
		{[]string{"te?t", "t?s?", "random"}, "tost", true},
		{[]string{"te?t", "t??e?", "random"}, "toser", true},
		{[]string{"*st", "n?st", "l*st"}, "list", true},
		{[]string{"?", "??", "???"}, "t", true},
		{[]string{"a", "b", "c"}, "d", false},
		{[]string{}, "test", false},
		{[]string{"*"}, "any", true},
		{[]string{"abc", "def", "ghi"}, "ghi", true},
		{[]string{"abc", "def", "ghi"}, "xyz", false},
		{[]string{"abc*", "def*", "ghi*"}, "ghi-test", true},
		{[]string{"abc?", "def?", "ghi?"}, "ghiz", true},
		{[]string{"abc?", "def?", "ghi?"}, "ghizz", false},
		{[]string{"a*?", "b*?", "c*?"}, "cwhatever", true},
		{[]string{"a*?", "b*?", "c*?"}, "dwhatever", false},
		{[]string{"*"}, "", true},
		{[]string{"abc"}, "abc", true},
		{[]string{"?bc"}, "abc", true},
		{[]string{"abc*"}, "abcd", true},
		{[]string{"guacamole", "The?Simpsons*"}, "The Simpsons S12", true},
		{[]string{"guacamole*", "The?Sompsons*"}, "The Simpsons S12", false},
		{[]string{"guac?mole*", "The?S?mpson"}, "The Simpsons S12", false},
		{[]string{"guac?mole*", "The?S?mpson"}, "guacamole Tornado", true},
		{[]string{"mole*", "The?S?mpson"}, "guacamole Tornado", false},
		{[]string{"??**mole*", "The?S?mpson"}, "guacamole Tornado", true},
	}

	for _, tt := range tests {
		if got := MatchSlice(tt.patterns, tt.name); got != tt.want {
			t.Errorf("MatchSlice(%v, %q) = %v, want %v", tt.patterns, tt.name, got, tt.want)
		}
	}
}

func Benchmark_Regex(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StartTimer()
		TestMatchSlice(nil)
		b.StopTimer()
	}
}
