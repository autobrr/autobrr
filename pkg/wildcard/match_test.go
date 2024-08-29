// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package wildcard

import "testing"

// TestMatch - Tests validate the logic of wild card matching.
// `Match` supports '*' and '?' wildcards.
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
	}
	// Iterating over the test cases, call the function under test and asert the output.
	for i, testCase := range testCases {
		actualResult := Match(testCase.pattern, testCase.text)
		if testCase.matched != actualResult {
			t.Errorf("Test %d: Expected the result to be `%v`, but instead found it to be `%v`", i+1, testCase.matched, actualResult)
		}
	}
}
