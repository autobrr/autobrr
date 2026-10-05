// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package newznab

import (
	"encoding/xml"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestFeedItem_parseAttributesTvdb(t *testing.T) {
	tests := []struct {
		name       string
		attributes []ItemAttr
		want       string
	}{
		{
			name:       "tvdbid attr",
			attributes: []ItemAttr{{Name: "tvdbid", Value: "77537"}},
			want:       "77537",
		},
		{
			name:       "tvdb attr",
			attributes: []ItemAttr{{Name: "tvdb", Value: "77537"}},
			want:       "77537",
		},
		{
			name:       "first value wins",
			attributes: []ItemAttr{{Name: "tvdbid", Value: "77537"}, {Name: "tvdb", Value: "1"}},
			want:       "77537",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &FeedItem{Attributes: tt.attributes}
			f.parseAttributes()

			assert.Equal(t, tt.want, f.TvdbId)
		})
	}
}

func TestFeedItem_UnmarshalXML(t *testing.T) {
	const payload = `<item xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/">
	<title>That Show S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-NOGROUP</title>
	<guid>https://mock.local/details/1</guid>
	<pubDate>Thu, 24 Sep 2026 05:58:24 +0000</pubDate>
	<size>589927041</size>
	<enclosure url="https://mock.local/getnzb/1" length="589927041" type="application/x-nzb"/>
	<newznab:attr name="tvdbid" value="12345"/>
	<newznab:attr name="tvmazeid" value="234"/>
</item>`

	var item FeedItem
	require.NoError(t, xml.Unmarshal([]byte(payload), &item))

	assert.Equal(t, "12345", item.TvdbId)
	assert.Equal(t, time.Date(2026, time.September, 24, 5, 58, 24, 0, time.UTC), item.PubDate.UTC())
}
