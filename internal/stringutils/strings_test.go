// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package stringutils

import (
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/assert"
)

func TestTruncateStr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		limit int
		want  string
	}{
		{
			name:  "shorter than limit is untouched",
			input: "hello",
			limit: 10,
			want:  "hello",
		},
		{
			name:  "exactly at limit is untouched",
			input: "hello",
			limit: 5,
			want:  "hello",
		},
		{
			name:  "keeps both ends and drops the middle",
			input: "abcdefghijklmnop",
			limit: 13,
			want:  "abc [...] nop",
		},
		{
			name:  "limit equal to the marker just cuts",
			input: "abcdefghij",
			limit: 7,
			want:  "abcdefg",
		},
		{
			name:  "one over the marker keeps a single leading rune",
			input: "abcdefghij",
			limit: 8,
			want:  "a [...] ",
		},
		{
			name:  "limit smaller than the marker just cuts",
			input: "abcdefgh",
			limit: 3,
			want:  "abc",
		},
		{
			name:  "zero limit yields empty",
			input: "abcdefgh",
			limit: 0,
			want:  "",
		},
		{
			name:  "negative limit yields empty",
			input: "abcdefgh",
			limit: -1,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := TruncateStr(tt.input, tt.limit)

			assert.Equal(t, tt.want, got)
			if tt.limit > 0 {
				assert.LessOrEqual(t, utf8.RuneCountInString(got), tt.limit)
			}
		})
	}
}

func TestTruncateStr_CountsRunesNotBytes(t *testing.T) {
	t.Parallel()

	got := TruncateStr(strings.Repeat("日", 500), 100)

	assert.Equal(t, 100, utf8.RuneCountInString(got))
	assert.NotContains(t, got, "�")
}
