// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package torznab

import (
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient_FetchFeed(t *testing.T) {
	key := "mock-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			if apiKey != key {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write(nil)
				return
			}
		}
		payload, err := os.ReadFile("testdata/torznab_response.xml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Write(payload)
	}))
	defer srv.Close()

	type fields struct {
		Host      string
		ApiKey    string
		BasicAuth BasicAuth
	}
	tests := []struct {
		name    string
		fields  fields
		want    []FeedItem
		wantErr bool
	}{
		{
			name: "get feed",
			fields: fields{
				Host:      srv.URL,
				ApiKey:    key,
				BasicAuth: BasicAuth{},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(Config{Host: tt.fields.Host, ApiKey: tt.fields.ApiKey})
			_, err := c.FetchFeed(t.Context())
			if tt.wantErr && assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err)
			}
			//assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_GetCaps(t *testing.T) {
	key := "mock-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		//apiKey := r.Header.Get("X-API-Key")
		//if apiKey != key {
		//	w.WriteHeader(http.StatusUnauthorized)
		//	w.Write(nil)
		//	return
		//}

		if !strings.Contains(r.RequestURI, key) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		payload, err := os.ReadFile("testdata/caps_response.xml")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/xml")
		w.Write(payload)
	}))
	defer srv.Close()

	type fields struct {
		Host      string
		ApiKey    string
		BasicAuth BasicAuth
	}
	tests := []struct {
		name        string
		fields      fields
		want        *Caps
		wantErr     bool
		expectedErr string
	}{
		{
			name: "get caps",
			fields: fields{
				Host:      srv.URL + "/api/",
				ApiKey:    key,
				BasicAuth: BasicAuth{},
			},
			want: &Caps{
				XMLName: xml.Name{
					Space: "",
					Local: "caps",
				},
				Server: Server{
					Version:   "1.1",
					Title:     "...",
					Strapline: "...",
					Email:     "...",
					URL:       "http://indexer.local/",
					Image:     "http://indexer.local/content/banner.jpg",
				},
				Limits: Limits{
					Max:     100,
					Default: 50,
				},
				Retention: Retention{
					Days: 400,
				},
				Registration: Registration{
					Available: "yes",
					Open:      "yes",
				},
				Searching: Searching{
					Search: Search{
						Available:       "yes",
						SupportedParams: "q",
					},
					TvSearch: Search{
						Available:       "yes",
						SupportedParams: "q,rid,tvdbid,season,ep",
					},
					MovieSearch: Search{
						Available:       "no",
						SupportedParams: "q,imdbid,genre",
					},
					AudioSearch: Search{
						Available:       "no",
						SupportedParams: "q",
					},
					BookSearch: Search{
						Available:       "no",
						SupportedParams: "q",
					},
				},
				Categories: CapCategories{Categories: []Category{
					{
						ID:   2000,
						Name: "Movies",
						SubCategories: []Category{
							{
								ID:   2010,
								Name: "Foreign",
							},
						},
					},
					{
						ID:   5000,
						Name: "TV",
						SubCategories: []Category{
							{
								ID:   5040,
								Name: "HD",
							},
							{
								ID:   5070,
								Name: "Anime",
							},
						},
					},
				}},
				Groups: Groups{Group: Group{
					ID:          "1",
					Name:        "alt.binaries....",
					Description: "...",
					Lastupdate:  "...",
				}},
				Genres: Genres{
					Genre: Genre{
						ID:         "1",
						Categoryid: "5000",
						Name:       "Kids",
					},
				},
				Tags: Tags{Tag: []Tag{
					{
						Name:        "anonymous",
						Description: "Uploader is anonymous",
					},
					{
						Name:        "trusted",
						Description: "Uploader has high reputation",
					},
					{
						Name:        "internal",
						Description: "Uploader is an internal release group",
					},
				}},
			},
			wantErr: false,
		},
		{
			name: "bad key",
			fields: fields{
				Host:      srv.URL,
				ApiKey:    "badkey",
				BasicAuth: BasicAuth{},
			},
			want:        nil,
			wantErr:     true,
			expectedErr: "could not get caps for feed: unauthorized",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(Config{Host: tt.fields.Host, ApiKey: tt.fields.ApiKey})

			got, err := c.FetchCaps(t.Context())
			if tt.wantErr && assert.Error(t, err) {
				assert.EqualErrorf(t, err, tt.expectedErr, "Error should be: %v, got: %v", tt.wantErr, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
