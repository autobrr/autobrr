// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package radarr

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func Test_client_Push(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	key := "mock-key"

	mux.HandleFunc("/api/v3/release/push", func(w http.ResponseWriter, r *http.Request) {
		// request validation logic
		apiKey := r.Header.Get("X-Api-Key")
		if apiKey != "" {
			if apiKey != key {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write(nil)
				return
			}
		}

		defer r.Body.Close()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		if strings.Contains(string(data), "Minx 1 epi 9 2160p") {
			jsonPayload, _ := os.ReadFile("testdata/release_push_parse_error.json")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(jsonPayload)
			return
		}

		// read json response
		jsonPayload, _ := os.ReadFile("testdata/release_push_response.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonPayload)
	})

	type fields struct {
		config Config
	}
	type args struct {
		release ReleasePushRequest
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
					APIKey:    "",
					BasicAuth: false,
					Username:  "",
					Password:  "",
				},
			},
			args: args{release: ReleasePushRequest{
				Title:            "Some.Old.Movie.1996.Remastered.1080p.BluRay.REMUX.AVC.MULTI.TrueHD.Atmos.7.1-NOGROUP",
				DownloadUrl:      "https://www.test.org/rss/download/0000001/00000000000000000000/Some.Old.Movie.1996.Remastered.1080p.BluRay.REMUX.AVC.MULTI.TrueHD.Atmos.7.1-NOGROUP.torrent",
				Size:             0,
				Indexer:          "test",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2021-08-21T15:36:00Z",
			}},
			rejections: []string{"Could not find Some Old Movie"},
		},
		{
			name: "push_error",
			fields: fields{
				config: Config{
					Hostname:  ts.URL,
					APIKey:    key,
					BasicAuth: false,
					Username:  "",
					Password:  "",
				},
			},
			args: args{release: ReleasePushRequest{
				Title:            "Some.Old.Movie.1996.Remastered.1080p.BluRay.REMUX.AVC.MULTI.TrueHD.Atmos.7.1-NOGROUP",
				DownloadUrl:      "https://www.test.org/rss/download/0000001/00000000000000000000/Some.Old.Movie.1996.Remastered.1080p.BluRay.REMUX.AVC.MULTI.TrueHD.Atmos.7.1-NOGROUP.torrent",
				Size:             0,
				Indexer:          "test",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2021-08-21T15:36:00Z",
			}},
			rejections: []string{"Could not find Some Old Movie"},
		},
		{
			name: "push_parse_error",
			fields: fields{
				config: Config{
					Hostname:  ts.URL,
					APIKey:    key,
					BasicAuth: false,
					Username:  "",
					Password:  "",
				},
			},
			args: args{release: ReleasePushRequest{
				Title:            "Minx 1 epi 9 2160p",
				DownloadUrl:      "https://www.test.org/rss/download/0000001/00000000000000000000/Minx.1.epi.9.2160p.torrent",
				Size:             0,
				Indexer:          "test",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2021-08-21T15:36:00Z",
			}},
			rejections: []string{"[error: ] Title: Unable to parse - got value: Minx 1 epi 9 2160p"},
			wantErr:    false,
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

func Test_client_Test(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

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
			want:        &SystemStatusResponse{Version: "3.2.2.5080"},
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
			wantErr:     true,
			expectedErr: "unauthorized: bad credentials",
		},
		{
			name: "fetch_subfolder",
			cfg: Config{
				Hostname:  srv.URL + "/radarr",
				APIKey:    key,
				BasicAuth: false,
				Username:  "",
				Password:  "",
			},
			want:        &SystemStatusResponse{Version: "3.2.2.5080"},
			expectedErr: "",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.cfg)

			got, err := c.Test(context.Background())
			if tt.wantErr && assert.Error(t, err) {
				assert.EqualErrorf(t, err, tt.expectedErr, "Error should be: %v, got: %v", tt.wantErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_client_Push_invalid_download_client(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	mux.HandleFunc("/api/v3/release/push", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`[{
			"propertyName": "DownloadClient",
			"errorMessage": "Download client does not exist.",
			"errorCode": "InvalidValue",
			"attemptedValue": "bad-client",
			"severity": "Error"
		}]`))
	})

	client := New(Config{
		Hostname: ts.URL,
	})

	rejections, err := client.Push(context.Background(), Release{
		Title:            "Example",
		DownloadUrl:      "https://example.invalid/release.torrent",
		Size:             0,
		Indexer:          "test",
		DownloadProtocol: "torrent",
		Protocol:         "torrent",
		PublishDate:      "2024-01-01T00:00:00Z",
		DownloadClient:   "bad-client",
	})

	assert.Nil(t, rejections)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "invalid configuration")
		assert.Contains(t, err.Error(), "Download client does not exist.")
	}
}
