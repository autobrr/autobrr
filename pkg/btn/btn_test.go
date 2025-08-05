// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package btn

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestAPI(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	key := "mock-key"

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// request validation logic
		//apiKey := r.Header.Get("ApiKey")
		//if apiKey != key {
		//	w.WriteHeader(http.StatusUnauthorized)
		//	w.Write(nil)
		//	return
		//}

		// read json response
		jsonPayload, _ := os.ReadFile("testdata/btn_get_user_info.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonPayload)
	})

	type fields struct {
		Url    string
		APIKey string
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "test_user",
			fields: fields{
				Url:    ts.URL,
				APIKey: key,
			},
			want:    true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.fields.APIKey, WithUrl(ts.URL))

			got, err := c.TestAPI(context.Background())
			if tt.wantErr && assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClient_GetTorrentByID(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	key := "mock-key"

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected 'POST' reqeust, got '%v'", r.Method)
		}

		defer r.Body.Close()
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("expected error to be nil got %v", err)
		}

		if !strings.Contains(string(data), "1555073") {
			//t.Errorf(
			//	`response body "%s" does not contain "1555073"`,
			//	string(data),
			//)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if !strings.Contains(string(data), key) {
			jsonPayload, _ := os.ReadFile("testdata/btn_bad_creds.json")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(jsonPayload)
			return
		}

		// read json response
		jsonPayload, _ := os.ReadFile("testdata/btn_get_torrent_by_id.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonPayload)
	})

	type fields struct {
		Url    string
		APIKey string
	}
	type args struct {
		torrentID string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *domain.TorrentBasic
		wantErr bool
	}{
		{
			name: "btn_get_torrent_by_id",
			fields: fields{
				Url:    ts.URL,
				APIKey: key,
			},
			args: args{torrentID: "1555073"},
			want: &domain.TorrentBasic{
				Id:        "",
				TorrentId: "1555073",
				InfoHash:  "56CD94119F6BF7FC294A92D7A4099C3D1815C907",
				Size:      "3288852849",
			},
			wantErr: false,
		},
		{
			name: "btn_get_torrent_by_id_not_found",
			fields: fields{
				Url:    ts.URL,
				APIKey: key,
			},
			args:    args{torrentID: "9555073"},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.fields.APIKey, WithUrl(ts.URL))

			got, err := c.GetTorrentByID(context.Background(), tt.args.torrentID)
			if tt.wantErr && assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
