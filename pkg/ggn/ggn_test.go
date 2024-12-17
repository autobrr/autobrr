// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package ggn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func Test_client_GetTorrentByID(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	key := "mock-key"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// request validation logic
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != key {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		id := r.URL.Query().Get("id")
		var jsonPayload []byte
		var err error
		switch id {
		case "422368":
			jsonPayload, err = os.ReadFile("testdata/ggn_get_torrent_by_id.json")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			break

		case "100002":
			jsonPayload, err = os.ReadFile("testdata/ggn_get_by_id_not_found.json")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			break
		}

		// read json response
		//jsonPayload, _ := os.ReadFile("testdata/ggn_get_torrent_by_id.json")
		//w.Header().Set("Content-Type", "application/json")
		//w.WriteHeader(http.StatusOK)
		w.Write(jsonPayload)
	}))
	defer ts.Close()

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
			name: "get_by_id_1",
			fields: fields{
				Url:    ts.URL,
				APIKey: key,
			},
			args: args{torrentID: "422368"},
			want: &domain.TorrentBasic{
				Id:       "422368",
				InfoHash: "78DA2811E6732012B8224198D4DC2FD49A5E950F",
				Size:     "134800",
			},
			wantErr: false,
		},
		{
			name: "get_by_invalid_id",
			fields: fields{
				Url:    ts.URL,
				APIKey: key,
			},
			args:    args{torrentID: "100002"},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.fields.APIKey, WithUrl(ts.URL))

			got, err := c.GetTorrentByID(context.Background(), tt.args.torrentID)
			if tt.wantErr && assert.Error(t, err) {
				t.Logf("got err: %v", err)
				assert.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
