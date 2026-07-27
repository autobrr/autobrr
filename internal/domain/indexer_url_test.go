// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Parsed templates are cached and shared, so the same template has to keep
// rendering per-announce variables rather than the first announce's values.
func TestParseTemplateURL_CachedTemplateRendersEachCall(t *testing.T) {
	t.Parallel()

	const source = "/torrent/{{ .torrentId }}/{{ .rsskey }}"

	first, err := parseTemplateURL("https://tracker.test", source, map[string]string{
		"torrentId": "1",
		"rsskey":    "abc",
	}, "downloadurl")
	require.NoError(t, err)
	assert.Equal(t, "https://tracker.test/torrent/1/abc", first.String())

	second, err := parseTemplateURL("https://tracker.test", source, map[string]string{
		"torrentId": "2",
		"rsskey":    "def",
	}, "downloadurl")
	require.NoError(t, err)
	assert.Equal(t, "https://tracker.test/torrent/2/def", second.String())
}

// Announce processors run one goroutine per channel and share the cache.
func TestParseTemplateURL_ConcurrentRenders(t *testing.T) {
	t.Parallel()

	const source = "/torrent/{{ .torrentId }}"

	var wg sync.WaitGroup
	for i := range 32 {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			got, err := parseTemplateURL("https://tracker.test", source, map[string]string{
				"torrentId": string(rune('a' + id%26)),
			}, "downloadurl")

			assert.NoError(t, err)
			assert.Equal(t, "https://tracker.test/torrent/"+string(rune('a'+id%26)), got.String())
		}(i)
	}

	wg.Wait()
}

func TestParseTemplateURL_InvalidTemplate(t *testing.T) {
	t.Parallel()

	_, err := parseTemplateURL("https://tracker.test", "/torrent/{{ .unclosed ", map[string]string{}, "downloadurl")
	assert.Error(t, err)
}

func TestGetStringMapValue(t *testing.T) {
	t.Parallel()

	vars := map[string]string{
		"torrentName":      "Some.Release-GRP",
		"freeleechPercent": "100%",
		"TorrentSize":      "1.5 GB",
	}

	t.Run("exact match", func(t *testing.T) {
		got, ok := getStringMapValue(vars, "torrentName")
		assert.True(t, ok)
		assert.Equal(t, "Some.Release-GRP", got)
	})

	// Definitions written before capture-group names were normalised can still
	// announce differently cased vars, so the fallback has to stay.
	t.Run("case-insensitive fallback", func(t *testing.T) {
		got, ok := getStringMapValue(vars, "torrentsize")
		assert.True(t, ok)
		assert.Equal(t, "1.5 GB", got)
	})

	t.Run("miss", func(t *testing.T) {
		got, ok := getStringMapValue(vars, "uploader")
		assert.False(t, ok)
		assert.Empty(t, got)
	})

	t.Run("empty map", func(t *testing.T) {
		got, ok := getStringMapValue(map[string]string{}, "torrentName")
		assert.False(t, ok)
		assert.Empty(t, got)
	})
}

func TestGetStringMapValueAlt(t *testing.T) {
	t.Parallel()

	t.Run("prefers the first key present", func(t *testing.T) {
		got, ok := getStringMapValueAlt(map[string]string{
			"releaseName": "from-release-name",
			"torrentName": "from-torrent-name",
		}, "releaseName", "torrentName")

		assert.True(t, ok)
		assert.Equal(t, "from-release-name", got)
	})

	t.Run("falls through to a later key", func(t *testing.T) {
		got, ok := getStringMapValueAlt(map[string]string{
			"torrentName": "from-torrent-name",
		}, "releaseName", "torrentName")

		assert.True(t, ok)
		assert.Equal(t, "from-torrent-name", got)
	})

	t.Run("miss", func(t *testing.T) {
		got, ok := getStringMapValueAlt(map[string]string{}, "releaseName", "torrentName")
		assert.False(t, ok)
		assert.Empty(t, got)
	})
}
