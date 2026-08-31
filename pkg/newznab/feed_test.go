// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package newznab

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeedItem_parseAttributes(t *testing.T) {
	tests := []struct {
		name       string
		attributes []ItemAttr
		want       string
	}{
		{
			name:       "bare id is prefixed with tt",
			attributes: []ItemAttr{{Name: "imdb", Value: "0133093"}},
			want:       "tt0133093",
		},
		{
			name:       "prefixed id is kept as is",
			attributes: []ItemAttr{{Name: "imdbid", Value: "tt0133093"}},
			want:       "tt0133093",
		},
		{
			name:       "empty value is ignored",
			attributes: []ItemAttr{{Name: "imdb", Value: ""}},
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FeedItem{Attributes: tt.attributes}
			f.parseAttributes()

			assert.Equal(t, tt.want, f.ImdbId)
		})
	}
}
