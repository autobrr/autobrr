package trakt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_transformTraktURL(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			name:   "smart list app.trakt.tv with query",
			input:  "https://app.trakt.tv/lists/smart/view/series-ece60ab98f5dd862?mode=media",
			expect: "https://api.trakt.tv/smart-lists/series-ece60ab98f5dd862/items",
		},
		{
			name:   "smart list app.trakt.tv without query",
			input:  "https://app.trakt.tv/lists/smart/view/series-ece60ab98f5dd862",
			expect: "https://api.trakt.tv/smart-lists/series-ece60ab98f5dd862/items",
		},
		{
			name:   "smart list trakt.tv",
			input:  "https://trakt.tv/lists/smart/view/series-ece60ab98f5dd862?mode=media",
			expect: "https://api.trakt.tv/smart-lists/series-ece60ab98f5dd862/items",
		},
		{
			name:   "smart list direct api url",
			input:  "https://api.trakt.tv/smart-lists/series-ece60ab98f5dd862/items",
			expect: "https://api.trakt.tv/smart-lists/series-ece60ab98f5dd862/items",
		},
		{
			name:   "user custom list web url",
			input:  "https://trakt.tv/users/johndoe/lists/my-cool-list",
			expect: "https://api.trakt.tv/users/johndoe/lists/my-cool-list/items",
		},
		{
			name:   "user watchlist web url",
			input:  "https://trakt.tv/users/johndoe/watchlist",
			expect: "https://api.trakt.tv/users/johndoe/watchlist/items",
		},
		{
			name:   "autobrr hosted list url",
			input:  "https://api.autobrr.com/lists/trakt/anticipated-tv",
			expect: "https://api.autobrr.com/lists/trakt/anticipated-tv",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := transformTraktURL(tt.input)
			assert.Equal(t, tt.expect, got)
		})
	}
}
