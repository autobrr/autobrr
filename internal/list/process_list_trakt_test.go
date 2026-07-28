// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockFilterService struct {
	updatedFilter domain.FilterUpdate
}

func (m *mockFilterService) UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error {
	m.updatedFilter = filter
	return nil
}

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
			name:   "smart list direct api url without items",
			input:  "https://api.trakt.tv/smart-lists/series-ece60ab98f5dd862",
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

func Test_trakt_smartList(t *testing.T) {
	smartListResponse := `[
  {
    "type": "show",
    "show": {
      "ids": {
        "imdb": "tt11198330",
        "plex": {
          "guid": "5e160ed3e68804001e87a7b5",
          "slug": "house-of-the-dragon"
        },
        "slug": "house-of-the-dragon",
        "tmdb": 94997,
        "tvdb": 371572,
        "trakt": 154574
      },
      "year": 2022,
      "title": "House of the Dragon",
      "aired_episodes": 24
    },
    "rank": 1
  }
]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2", r.Header.Get("trakt-api-version"))
		assert.Equal(t, "test-api-key", r.Header.Get("trakt-api-key"))
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(smartListResponse))
	}))
	defer ts.Close()

	mockFilter := &mockFilterService{}
	svc := &Service{
		log:        zerolog.Nop(),
		httpClient: ts.Client(),
		filterSvc:  mockFilter,
	}

	list := &domain.List{
		Name:   "Smart List Test",
		Type:   domain.ListTypeTrakt,
		URL:    ts.URL,
		APIKey: "test-api-key",
		Filters: []domain.ListFilter{
			{ID: 1, Name: "Filter 1"},
		},
	}

	err := svc.trakt(context.Background(), list)
	require.NoError(t, err)

	require.NotNil(t, mockFilter.updatedFilter.Shows)
	assert.Contains(t, *mockFilter.updatedFilter.Shows, "House?of?the?Dragon")
}
