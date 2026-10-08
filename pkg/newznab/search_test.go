// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package newznab

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_Search_Offset(t *testing.T) {
	tests := []struct {
		name       string
		offset     int
		wantOffset string
	}{
		{name: "first page sends no offset", offset: 0, wantOffset: ""},
		{name: "later pages send the offset", offset: 100, wantOffset: "100"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotOffset, gotLimit string

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Query().Get("t") {
				case "caps":
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(`<caps><limits max="100" default="50"/></caps>`))
				case "search":
					gotOffset = r.URL.Query().Get("offset")
					gotLimit = r.URL.Query().Get("limit")
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(`<rss><channel><title>test</title></channel></rss>`))
				}
			}))
			defer srv.Close()

			c := NewClient(Config{Host: srv.URL})

			resp, err := c.Search(t.Context(), "", nil, tt.offset)
			require.NoError(t, err)
			require.NotNil(t, resp)

			assert.Equal(t, tt.wantOffset, gotOffset)
			assert.Equal(t, "100", gotLimit, "page size comes from the caps max")
			assert.Equal(t, 100, resp.Limit)
		})
	}
}

func TestClient_Search_DefaultLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("t") {
		case "caps":
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<caps></caps>`))
		case "search":
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<rss><channel><title>test</title></channel></rss>`))
		}
	}))
	defer srv.Close()

	c := NewClient(Config{Host: srv.URL})

	resp, err := c.Search(t.Context(), "", nil, 0)
	require.NoError(t, err)

	assert.Equal(t, defaultSearchLimit, resp.Limit, "caps without limits fall back to the default page size")
}

func TestClient_Search_PagingResponse(t *testing.T) {
	tests := []struct {
		name      string
		response  string
		wantTotal int
	}{
		{
			name:      "newznab namespaced response element",
			response:  `<rss><channel><title>test</title><newznab:response xmlns:newznab="http://www.newznab.com/DTD/2010/feeds/attributes/" offset="0" total="7"/><item><title>a</title><guid>g1</guid></item></channel></rss>`,
			wantTotal: 7,
		},
		{
			name:      "torznab namespaced response element",
			response:  `<rss><channel><title>test</title><torznab:response xmlns:torznab="http://torznab.com/schemas/2015/feed" offset="0" total="7"/><item><title>a</title><guid>g1</guid></item></channel></rss>`,
			wantTotal: 7,
		},
		{
			name:      "missing response element reports zero",
			response:  `<rss><channel><title>test</title><item><title>a</title><guid>g1</guid></item></channel></rss>`,
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Query().Get("t") {
				case "caps":
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(`<caps><limits max="100" default="50"/></caps>`))
				case "search":
					w.Header().Set("Content-Type", "application/xml")
					w.Write([]byte(tt.response))
				}
			}))
			defer srv.Close()

			c := NewClient(Config{Host: srv.URL})

			resp, err := c.Search(t.Context(), "", nil, 0)
			require.NoError(t, err)

			assert.Equal(t, tt.wantTotal, resp.Total)
		})
	}
}

func TestClient_Search_KeepsHostQueryParams(t *testing.T) {
	var capsQuery, searchQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("t") {
		case "caps":
			capsQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<caps><limits max="100" default="50"/></caps>`))
		case "search":
			searchQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<rss><channel><title>test</title></channel></rss>`))
		}
	}))
	defer srv.Close()

	// params embedded in the feed url must survive, e.g. an api key
	c := NewClient(Config{Host: srv.URL + "/api?foo=bar&apikey=hostkey", ApiKey: "fieldkey"})

	_, err := c.Search(t.Context(), "", nil, 25)
	require.NoError(t, err)

	assert.Equal(t, "bar", searchQuery.Get("foo"), "host params are preserved")
	assert.Equal(t, "fieldkey", searchQuery.Get("apikey"), "the api key from the config wins")
	assert.Equal(t, "25", searchQuery.Get("offset"))

	assert.Equal(t, "bar", capsQuery.Get("foo"), "caps requests keep host params too")
	assert.Equal(t, "fieldkey", capsQuery.Get("apikey"))
}

func TestClient_Search_KeepsValidHostParamsOnMalformedQuery(t *testing.T) {
	var capsQuery, searchQuery url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Query().Get("t") {
		case "caps":
			capsQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<caps><limits max="100" default="50"/></caps>`))
		case "search":
			searchQuery = r.URL.Query()
			w.Header().Set("Content-Type", "application/xml")
			w.Write([]byte(`<rss><channel><title>test</title></channel></rss>`))
		}
	}))
	defer srv.Close()

	c := NewClient(Config{Host: srv.URL + "/api?foo=bar&bad=%zz"})

	_, err := c.Search(t.Context(), "", nil, 0)
	require.NoError(t, err)

	assert.Equal(t, "bar", searchQuery.Get("foo"))
	assert.Equal(t, "bar", capsQuery.Get("foo"))
}
