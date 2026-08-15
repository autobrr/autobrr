// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package chaptarr

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func Test_client_Push(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)
	log.SetOutput(io.Discard)

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	key := "mock-key"

	mux.HandleFunc("/api/v1/release/push", func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-Api-Key")
		if apiKey != "" {
			if apiKey != key {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write(nil)
				return
			}
		}

		var release Release
		if err := json.NewDecoder(r.Body).Decode(&release); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write(nil)
			return
		}

		status := http.StatusOK
		fixture := "testdata/release_push_response.json"

		switch {
		case release.Title == "":
			status = http.StatusBadRequest
			fixture = "testdata/release_push_bad_request_response.json"
		case release.Title == "Some Unknown Author - A Book Nobody Has [MKV]":
			fixture = "testdata/release_push_rejected_response.json"
		case release.Indexer == "myanonamouse":
			fixture = "testdata/release_push_temp_rejected_response.json"
		}

		jsonPayload, _ := os.ReadFile(fixture)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(jsonPayload)
	})

	type fields struct {
		config Config
	}
	type args struct {
		release Release
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		rejections []string
		err        error
		wantErr    bool
	}{
		{
			name: "push",
			fields: fields{
				config: Config{
					Hostname:  ts.URL,
					APIKey:    key,
					BasicAuth: false,
					Username:  "",
					Password:  "",
				},
			},
			args: args{release: Release{
				Title:            "Brandon Sanderson - The Way of Kings (2010) [M4B]",
				DownloadUrl:      "https://www.mock-indexer.test/tor/download.php?tid=000000",
				Size:             734003200,
				Indexer:          "mock-indexer",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2026-08-09T17:36:15Z",
			}},
			rejections: nil,
		},
		{
			name: "push_rejected",
			fields: fields{
				config: Config{
					Hostname:  ts.URL,
					APIKey:    key,
					BasicAuth: false,
					Username:  "",
					Password:  "",
				},
			},
			args: args{release: Release{
				Title:            "Some Unknown Author - A Book Nobody Has [MKV]",
				DownloadUrl:      "https://www.mock-indexer.test/tor/download.php?tid=000001",
				Size:             1048576,
				Indexer:          "mock-indexer",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2026-08-09T17:36:15Z",
			}},
			rejections: []string{"Unknown Author"},
		},
		{
			// a MAM safety pause comes back with rejected false and only
			// temporarilyRejected set, and must still surface as a rejection
			name: "push_temporarily_rejected",
			fields: fields{
				config: Config{
					Hostname:  ts.URL,
					APIKey:    key,
					BasicAuth: false,
					Username:  "",
					Password:  "",
				},
			},
			args: args{release: Release{
				Title:            "Brandon Sanderson - The Way of Kings (2010) [M4B]",
				DownloadUrl:      "https://www.mock-indexer.test/tor/download.php?tid=12345",
				Size:             734003200,
				Indexer:          "myanonamouse",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2026-08-09T17:36:15Z",
			}},
			rejections: []string{"MAM safety pause: this release has no valid MAM torrent identity"},
		},
		{
			name: "push_bad_request",
			fields: fields{
				config: Config{
					Hostname:  ts.URL,
					APIKey:    key,
					BasicAuth: false,
					Username:  "",
					Password:  "",
				},
			},
			args: args{release: Release{
				DownloadUrl:      "https://www.mock-indexer.test/tor/download.php?tid=000002",
				Indexer:          "mock-indexer",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2026-08-09T17:36:15Z",
			}},
			rejections: []string{"Title: 'Title' must not be empty."},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.fields.config)

			rejections, err := c.Push(context.Background(), tt.args.release)
			assert.Equal(t, tt.rejections, rejections)
			if tt.wantErr && assert.Error(t, err) {
				assert.Equal(t, tt.err, err)
			}
		})
	}
}

// genres has shipped as both a string and an array (autobrr/autobrr#2413), which is
// why the field is absent from Book - neither shape may fail the decode.
func Test_client_GetBooks(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)
	log.SetOutput(io.Discard)

	key := "mock-key"

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	var gotMediaType string

	mux.HandleFunc("/api/v1/book", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != key {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		gotMediaType = r.URL.Query().Get("mediaType")

		jsonPayload, _ := os.ReadFile("testdata/book_response.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonPayload)
	})

	c := New(Config{Hostname: ts.URL, APIKey: key})

	books, err := c.GetBooks(context.Background(), "")
	assert.NoError(t, err)
	assert.Equal(t, "", gotMediaType)
	assert.Len(t, books, 3)

	assert.Equal(t, "The Best Book", books[0].Title)
	assert.Equal(t, "audiobook", books[0].MediaType)
	assert.Equal(t, "Famous Narrator", books[0].Narrator)
	assert.True(t, books[0].Monitored)
	assert.Equal(t, "Famous Author", books[0].Author.AuthorName)
	assert.False(t, books[2].Monitored)

	_, err = c.GetBooks(context.Background(), "audiobook")
	assert.NoError(t, err)
	assert.Equal(t, "audiobook", gotMediaType)

	_, err = New(Config{Hostname: ts.URL, APIKey: "bad-mock-key"}).GetBooks(context.Background(), "")
	assert.EqualError(t, err, "could not get books: unauthorized: bad credentials")
}

func Test_client_Test(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)
	log.SetOutput(io.Discard)

	key := "mock-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-Api-Key")
		if apiKey != "" {
			if apiKey != key {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write(nil)
				return
			}
		}
		jsonPayload, _ := os.ReadFile("testdata/system_status_response.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonPayload)
	}))
	defer srv.Close()

	tests := []struct {
		name        string
		cfg         Config
		want        *SystemStatusResponse
		expectedErr string
		wantErr     bool
	}{
		{
			name: "fetch",
			cfg: Config{
				Hostname:  srv.URL,
				APIKey:    key,
				BasicAuth: false,
				Username:  "",
				Password:  "",
			},
			want:        &SystemStatusResponse{AppName: "Chaptarr", Version: "0.9.911.0"},
			expectedErr: "",
			wantErr:     false,
		},
		{
			name: "fetch_unauthorized",
			cfg: Config{
				Hostname:  srv.URL,
				APIKey:    "bad-mock-key",
				BasicAuth: false,
				Username:  "",
				Password:  "",
			},
			want:        nil,
			expectedErr: "unauthorized: bad credentials",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.cfg)

			got, err := c.Test(context.Background())
			if tt.wantErr && assert.Error(t, err) {
				assert.EqualErrorf(t, err, tt.expectedErr, "Error should be: %v, got: %v", tt.expectedErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
