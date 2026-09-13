// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package hebits

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testCookie = "session=test-session-value; userid=42"

func torrentJSON(id int, free bool, seeders, leechers, snatched, size int) []byte {
	payload := map[string]any{
		"status": "success",
		"response": map[string]any{
			"torrent": map[string]any{
				"id":          id,
				"freeTorrent": free,
				"seeders":     seeders,
				"leechers":    leechers,
				"snatched":    snatched,
				"size":        size,
			},
		},
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestClient_GetTorrentByID(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	var gotCookie atomic.Value
	var gotID atomic.Value

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie.Store(r.Header.Get("Cookie"))
		gotID.Store(r.URL.Query().Get("id"))

		if r.Header.Get("Cookie") != testCookie {
			http.Redirect(w, r, "/login.php", http.StatusFound)
			return
		}

		switch r.URL.Query().Get("id") {
		case "134205":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(torrentJSON(134205, true, 10, 2, 9, 1276432647))
		case "80081":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(torrentJSON(80081, false, 3, 1, 4, 1024))
		case "mismatch":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(torrentJSON(1, true, 1, 0, 0, 1))
		case "failure":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"failure","error":"bad id parameter"}`))
		case "malformed":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{not json`))
		case "html":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<html><body>login</body></html>`))
		case "login":
			http.Redirect(w, r, "/login.php", http.StatusFound)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"failure","error":"not found"}`))
		}
	}))
	defer ts.Close()

	t.Run("sends configured cookie and extracts torrent id", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "134205")
		require.NoError(t, err)
		assert.Equal(t, testCookie, gotCookie.Load())
		assert.Equal(t, "134205", gotID.Load())
		assert.Equal(t, &domain.TorrentBasic{
			Id:               "134205",
			Size:             "1276432647",
			Freeleech:        true,
			FreeleechPercent: 100,
			Seeders:          10,
			Leechers:         2,
		}, got)
	})

	t.Run("preserves freeleech false and swarm fields", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "80081")
		require.NoError(t, err)
		assert.Equal(t, &domain.TorrentBasic{
			Id:       "80081",
			Size:     "1024",
			Seeders:  3,
			Leechers: 1,
		}, got)
		assert.False(t, got.Freeleech)
	})

	t.Run("extracts torrent id from hebits announce link", func(t *testing.T) {
		pattern := regexp.MustCompile(`^Link: (?P<baseUrl>https:\/\/.*\/).*torrentid=(?P<torrentId>\d+)`)
		matches := pattern.FindStringSubmatch("Link: https://hebits.net/torrents.php?torrentid=80081")
		require.Len(t, matches, 3)
		assert.Equal(t, "80081", matches[2])

		c := NewClient(testCookie, WithUrl(ts.URL))
		got, err := c.GetTorrentByID(t.Context(), matches[2])
		require.NoError(t, err)
		assert.Equal(t, "80081", got.Id)
	})

	t.Run("rejects login redirect", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "login")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NotContains(t, err.Error(), testCookie)
	})

	t.Run("rejects html response", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "html")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected content type")
		assert.NotContains(t, err.Error(), testCookie)
	})

	t.Run("rejects malformed json", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "malformed")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not unmarshal body")
	})

	t.Run("rejects non-success api status", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "failure")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "hebits api status")
	})

	t.Run("rejects torrent id mismatch", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "mismatch")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "torrent id mismatch")
	})

	t.Run("rejects missing cookie", func(t *testing.T) {
		c := NewClient("", WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "134205")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "missing cookie")
	})

	t.Run("rejects empty torrent id", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must have torrentID")
	})

	t.Run("rejects unauthenticated cookie", func(t *testing.T) {
		c := NewClient("session=wrong", WithUrl(ts.URL))

		got, err := c.GetTorrentByID(t.Context(), "134205")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication failed")
		assert.NotContains(t, err.Error(), "session=wrong")
	})
}

func TestClient_GetTorrentByID_TimeoutAndCancel(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(200 * time.Millisecond):
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(torrentJSON(1, false, 0, 0, 0, 1))
	}))
	defer ts.Close()

	t.Run("request timeout", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL), WithHTTPClient(&http.Client{Timeout: 20 * time.Millisecond}))

		got, err := c.GetTorrentByID(t.Context(), "1")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), testCookie)
	})

	t.Run("context cancellation", func(t *testing.T) {
		c := NewClient(testCookie, WithUrl(ts.URL))

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		got, err := c.GetTorrentByID(ctx, "1")
		assert.Nil(t, got)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), testCookie)
	})
}

func TestClient_GetTorrentByID_DoesNotLogCookie(t *testing.T) {
	var buf bytes.Buffer
	log := zerolog.New(&buf).Level(zerolog.TraceLevel)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(torrentJSON(134205, true, 1, 0, 0, 1))
	}))
	defer ts.Close()

	c := NewClient(testCookie, WithUrl(ts.URL), WithLog(log))
	_, err := c.GetTorrentByID(t.Context(), "134205")
	require.NoError(t, err)

	assert.NotContains(t, buf.String(), testCookie)
	assert.NotContains(t, buf.String(), "test-session-value")
	assert.False(t, strings.Contains(strings.ToLower(buf.String()), "cookie:"))
}

func TestClient_TestAPI(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != testCookie {
			http.Redirect(w, r, "/login.php", http.StatusFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success"}`))
	}))
	defer ts.Close()

	c := NewClient(testCookie, WithUrl(ts.URL))
	ok, err := c.TestAPI(t.Context())
	require.NoError(t, err)
	assert.True(t, ok)
}

func loadLiveCookie(t *testing.T) string {
	t.Helper()

	if cookie := strings.TrimSpace(os.Getenv("HEBITS_COOKIE")); cookie != "" {
		return cookie
	}

	paths := []string{".hebits.cookie"}
	if file := os.Getenv("HEBITS_COOKIE_FILE"); file != "" {
		paths = append([]string{file}, paths...)
	}

	dir, err := os.Getwd()
	if err == nil {
		for i := 0; i < 4; i++ {
			paths = append(paths, filepath.Join(dir, ".hebits.cookie"))
			dir = filepath.Dir(dir)
		}
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		cookie := strings.TrimSpace(string(data))
		if cookie != "" && !strings.HasPrefix(cookie, "#") {
			return cookie
		}
	}

	t.Skip("no Hebits cookie found; set HEBITS_COOKIE or write it to gitignored .hebits.cookie")
	return ""
}

func TestLiveHebitsAPI(t *testing.T) {
	cookie := loadLiveCookie(t)
	torrentID := os.Getenv("HEBITS_TORRENT_ID")
	if torrentID == "" {
		torrentID = "134205"
	}

	c := NewClient(cookie)
	got, err := c.GetTorrentByID(t.Context(), torrentID)
	if err != nil {
		assert.NotContains(t, err.Error(), cookie)
		require.NoError(t, err)
	}

	require.NotNil(t, got)
	assert.Equal(t, torrentID, got.Id)
	t.Logf("torrent %s freeleech=%t seeders=%d leechers=%d size=%s", got.Id, got.Freeleech, got.Seeders, got.Leechers, got.Size)
}
