// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package ptp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestPTPClient_GetTorrentByID(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	user := "mock-user"
	key := "mock-key"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// request validation logic
		apiKey := r.Header.Get("ApiKey")
		if apiKey != key {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		apiUser := r.Header.Get("ApiUser")
		if apiUser != user {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		// read json response
		jsonPayload, _ := os.ReadFile("testdata/ptp_get_torrent_by_id.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(jsonPayload)
	}))
	defer ts.Close()

	type fields struct {
		Url     string
		APIUser string
		APIKey  string
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
			name: "get_by_id_1",
			fields: fields{
				Url:     ts.URL,
				APIUser: user,
				APIKey:  key,
			},
			args: args{torrentID: "1"},
			want: &domain.TorrentBasic{
				Id:       "1",
				InfoHash: "F57AA86DFB03F87FCC7636E310D35918442EAE5C",
				Size:     "1344512700",
			},
			wantErr: false,
		},
		{
			name: "get_by_id_not_found",
			fields: fields{
				Url:     ts.URL,
				APIUser: user,
				APIKey:  key,
			},
			args:    args{torrentID: "100002"},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.fields.APIUser, tt.fields.APIKey, WithUrl(ts.URL))

			got, err := c.GetTorrentByID(context.Background(), tt.args.torrentID)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}

func Test(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	user := "mock-user"
	key := "mock-key"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// request validation logic
		apiKey := r.Header.Get("ApiKey")
		if apiKey != key {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		apiUser := r.Header.Get("ApiUser")
		if apiUser != user {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		// read json response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(TorrentListResponse{
			TotalResults: "10",
			Movies:       []Movie{},
			Page:         "1",
		})
		//w.Write(nil)
	}))
	defer ts.Close()

	type fields struct {
		Url     string
		APIUser string
		APIKey  string
	}
	tests := []struct {
		name    string
		fields  fields
		want    bool
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{
				Url:     ts.URL,
				APIUser: user,
				APIKey:  key,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "bad_creds",
			fields: fields{
				Url:     ts.URL,
				APIUser: user,
				APIKey:  "",
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.fields.APIUser, tt.fields.APIKey, WithUrl(ts.URL))

			got, err := c.TestAPI(context.Background())

			if tt.wantErr && assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err)
			}
			assert.Equalf(t, tt.want, got, "Test()")
		})
	}
}
