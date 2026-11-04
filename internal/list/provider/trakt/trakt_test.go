package trakt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func newPagedServer(t *testing.T, total, limit int, paginate bool) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "2", r.Header.Get("trakt-api-version"))
		assert.Equal(t, "secret", r.Header.Get("trakt-api-key"))
		assert.Equal(t, "custom", r.URL.Query().Get("extended"))

		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			page, _ = strconv.Atoi(p)
		}

		pageCount := (total + limit - 1) / limit
		if !paginate {
			pageCount = 1
			limit = total
		}

		items := make([]map[string]any, 0, limit)
		for i := (page - 1) * limit; i < min(page*limit, total); i++ {
			items = append(items, map[string]any{"movie": map[string]any{"title": fmt.Sprintf("Movie %d", i)}})
		}

		w.Header().Set("Content-Type", "application/json")
		if paginate {
			w.Header().Set("X-Pagination-Page", strconv.Itoa(page))
			w.Header().Set("X-Pagination-Limit", strconv.Itoa(limit))
			w.Header().Set("X-Pagination-Page-Count", strconv.Itoa(pageCount))
			w.Header().Set("X-Pagination-Item-Count", strconv.Itoa(total))
		}
		_ = json.NewEncoder(w).Encode(items)
	}))
}

func TestClient_GetList(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		limit    int
		paginate bool
	}{
		{name: "single page", total: 42, limit: 100, paginate: true},
		{name: "two pages", total: 116, limit: 100, paginate: true},
		{name: "exact page boundary", total: 200, limit: 100, paginate: true},
		{name: "no pagination headers", total: 150, limit: 100, paginate: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newPagedServer(t, tt.total, tt.limit, tt.paginate)
			defer srv.Close()

			c := NewClient(zerolog.Nop(), "test", srv.URL, "secret")

			items, err := c.GetList(t.Context(), srv.URL+"/users/john/lists/movies/items?extended=custom")
			require.NoError(t, err)
			require.Len(t, items, tt.total)

			for i, item := range items {
				assert.Equal(t, fmt.Sprintf("Movie %d", i), item.Movie.Title)
			}
		})
	}
}

func TestClient_GetList_PageError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Pagination-Page-Count", "2")
		_, _ = w.Write([]byte(`[{"movie":{"title":"Movie 0"}}]`))
	}))
	defer srv.Close()

	c := NewClient(zerolog.Nop(), "test", srv.URL, "")

	_, err := c.GetList(t.Context(), srv.URL+"/users/john/lists/movies/items")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not fetch page 2 of 2")
}

func TestClient_GetList_PageLimit(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		traktHost bool
		expect    string
	}{
		{name: "trakt host gets max limit", traktHost: true, expect: "250"},
		{name: "trakt host keeps user limit", query: "?limit=50", traktHost: true, expect: "50"},
		{name: "other host untouched", traktHost: false, expect: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, tt.expect, r.URL.Query().Get("limit"))

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Pagination-Page-Count", "2")
				_, _ = w.Write([]byte(`[{"movie":{"title":"Movie"}}]`))
			}))
			defer srv.Close()

			if tt.traktHost {
				orig := apiHost
				apiHost = "127.0.0.1"
				t.Cleanup(func() { apiHost = orig })
			}

			c := NewClient(zerolog.Nop(), "test", srv.URL, "")

			items, err := c.GetList(t.Context(), srv.URL+"/users/john/lists/movies/items"+tt.query)
			require.NoError(t, err)
			assert.Len(t, items, 2)
			assert.Equal(t, int32(2), requests.Load())
		})
	}
}

func TestClient_GetList_RateLimit(t *testing.T) {
	tests := []struct {
		name        string
		rateLimited int32
		wantErr     bool
	}{
		{name: "retries until success", rateLimited: 2, wantErr: false},
		{name: "gives up after max retries", rateLimited: maxRateLimitRetries + 1, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) <= tt.rateLimited {
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"movie":{"title":"Movie"}}]`))
			}))
			defer srv.Close()

			c := NewClient(zerolog.Nop(), "test", srv.URL, "")

			items, err := c.GetList(t.Context(), srv.URL+"/users/john/lists/movies/items")
			if tt.wantErr {
				require.Error(t, err)
				assert.Equal(t, int32(maxRateLimitRetries+1), requests.Load())
				return
			}

			require.NoError(t, err)
			assert.Len(t, items, 1)
			assert.Equal(t, tt.rateLimited+1, requests.Load())
		})
	}
}

func TestClient_GetList_RateLimitContextCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient(zerolog.Nop(), "test", srv.URL, "")

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	_, err := c.GetList(ctx, srv.URL+"/users/john/lists/movies/items")
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func Test_parseRetryAfter(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect time.Duration
	}{
		{name: "seconds", input: "10", expect: 10 * time.Second},
		{name: "zero", input: "0", expect: 0},
		{name: "capped", input: "3600", expect: maxRetryAfter},
		{name: "overflow", input: "99999999999", expect: maxRetryAfter},
		{name: "missing", input: "", expect: defaultRetryAfter},
		{name: "http date", input: "Wed, 04 Nov 2026 10:00:00 GMT", expect: defaultRetryAfter},
		{name: "negative", input: "-5", expect: defaultRetryAfter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expect, parseRetryAfter(tt.input))
		})
	}
}

func TestClient_GetList_SavedPageParam(t *testing.T) {
	srv := newPagedServer(t, 116, 100, true)
	defer srv.Close()

	c := NewClient(zerolog.Nop(), "test", srv.URL, "secret")

	items, err := c.GetList(t.Context(), srv.URL+"/users/john/lists/movies/items?extended=custom&page=2")
	require.NoError(t, err)
	require.Len(t, items, 116)
	assert.Equal(t, "Movie 0", items[0].Movie.Title)
}

func TestClient_GetList_BadPagination(t *testing.T) {
	tests := []struct {
		name      string
		pageCount string
		expectErr string
	}{
		{name: "too many pages", pageCount: strconv.Itoa(maxPages + 1), expectErr: "exceeding the maximum"},
		{name: "malformed header", pageCount: "two", expectErr: "invalid X-Pagination-Page-Count"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-Pagination-Page-Count", tt.pageCount)
				_, _ = w.Write([]byte(`[{"movie":{"title":"Movie"}}]`))
			}))
			defer srv.Close()

			c := NewClient(zerolog.Nop(), "test", srv.URL, "")

			_, err := c.GetList(t.Context(), srv.URL+"/users/john/lists/movies/items")
			require.ErrorContains(t, err, tt.expectErr)
			assert.Equal(t, int32(1), requests.Load())
		})
	}
}
