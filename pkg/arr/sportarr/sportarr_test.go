// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package sportarr

import (
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

	mux.HandleFunc("/api/release/push", func(w http.ResponseWriter, r *http.Request) {
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
				Title:            "Formula.1.2026x10.Belgium.Race.SkyF1HD.1080p",
				DownloadUrl:      "https://www.test.org/rss/download/0000001/00000000000000000000/Formula.1.2026x10.Belgium.Race.SkyF1HD.1080p.torrent",
				Size:             0,
				Indexer:          "test",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2026-07-19T15:00:00Z",
			}},
			rejections: []string{"No matching event found"},
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
				Title:            "Formula.1.2026x10.Belgium.Race.SkyF1HD.1080p",
				DownloadUrl:      "https://www.test.org/rss/download/0000001/00000000000000000000/Formula.1.2026x10.Belgium.Race.SkyF1HD.1080p.torrent",
				Size:             0,
				Indexer:          "test",
				DownloadProtocol: "torrent",
				Protocol:         "torrent",
				PublishDate:      "2026-07-19T15:00:00Z",
			}},
			rejections: []string{"No matching event found"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.fields.config)

			rejections, err := c.Push(t.Context(), tt.args.release)
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
			want:        &SystemStatusResponse{Version: "4.0.1024.1104"},
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

			got, err := c.Test(t.Context())
			if tt.wantErr && assert.Error(t, err) {
				assert.EqualErrorf(t, err, tt.expectedErr, "Error should be: %v, got: %v", tt.expectedErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_SystemStatusResponse_SupportsNativeAPI(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{version: "4.0.1024.1104", want: true},
		{version: "4.1.0", want: true},
		{version: "5.0.0", want: true},
		{version: "4.0.1024", want: true},
		{version: "4.0.1023.1103", want: false},
		{version: "4.0.999", want: false},
		{version: "", want: false},
		{version: "garbage", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			r := SystemStatusResponse{Version: tt.version}
			assert.Equal(t, tt.want, r.SupportsNativeAPI())
		})
	}
}

// A temporarily rejected release comes back with rejected false and temporarilyRejected
// true, and waits in Sportarr's pending queue instead of being grabbed, so its rejections
// still have to reach the caller.
func Test_client_Push_temporarilyRejected(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.Disabled)
	log.SetOutput(io.Discard)

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	mux.HandleFunc("/api/release/push", func(w http.ResponseWriter, r *http.Request) {
		jsonPayload, _ := os.ReadFile("testdata/release_push_temporarily_rejected_response.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonPayload)
	})

	c := New(Config{Hostname: ts.URL})

	rejections, err := c.Push(t.Context(), ReleasePushRequest{
		Title:            "Formula.1.2026x10.Belgium.Race.SkyF1HD.1080p",
		DownloadUrl:      "https://www.mock-indexer.test/tor/download.php?tid=000000",
		Size:             1048576,
		Indexer:          "mock-indexer",
		DownloadProtocol: "torrent",
		Protocol:         "torrent",
		PublishDate:      "2026-08-15T17:36:15Z",
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"Waiting for a better quality release"}, rejections)
}
