// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package sonarr

import (
	"context"
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
				Title:            "That Show S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-NOGROUP",
				DownloadUrl:      "https://www.test.org/rss/download/0000001/00000000000000000000/That Show S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-NOGROUP.torrent",
				Size:             0,
				Indexer:          "test",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2021-08-21T15:36:00Z",
			}},
			rejections: []string{"Unknown Series"},
			//err:     errors.New("sonarr push rejected Unknown Series"),
			//wantErr: true,
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
				Title:            "That Show S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-NOGROUP",
				DownloadUrl:      "https://www.test.org/rss/download/0000001/00000000000000000000/That Show S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-NOGROUP.torrent",
				Size:             0,
				Indexer:          "test",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2021-08-21T15:36:00Z",
			}},
			rejections: []string{"Unknown Series"},
			//err:     errors.New("sonarr push rejected Unknown Series"),
			//wantErr: true,
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
			want:        &SystemStatusResponse{Version: "3.0.6.1196"},
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
	log.SetOutput(io.Discard)

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
