// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package wildcard

import (
	"testing"
)

// TestMatch - Tests validate the logic of wild card matching.
// `Match` supports '*' (zero or more characters) and '?' (one character) wildcards in typical glob style filtering.
// A '*' in a provided string will not result in matching the strings before and after the '*' of the string provided.
// Sample usage: In resource matching for bucket policy validation.
func TestMatch(t *testing.T) {
	testCases := []struct {
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
			pattern: "*EPUB*",
			text:    "Translated (Group) / EPUB",
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
	}
	// Iterating over the test cases, call the function under test and assert the output.
	for i, testCase := range testCases {
		actualResult := Match(testCase.pattern, testCase.text)
		if testCase.matched != actualResult {
			t.Errorf("Test %d: Expected the result to be `%v`, but instead found it to be `%v`", i+1, testCase.matched, actualResult)
		}
	}
}

func TestMatchSimple(t *testing.T) {
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
	tests := []struct {
		patterns []string
		name     string
		want     bool
	}{
		{[]string{"*", "test"}, "test", true},
		{[]string{"te?t", "tost", "random"}, "tost", true},
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
	tests := []struct {
		patterns []string
		name     string
		want     bool
	}{
		{[]string{"*", "test", "t?st"}, "test", true},
		{[]string{"te?t", "t?st", "random"}, "tost", true},
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
