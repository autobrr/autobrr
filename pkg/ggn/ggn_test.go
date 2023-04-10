package ggn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/autobrr/autobrr/internal/domain"
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

		if !strings.Contains(r.RequestURI, "422368") {
			jsonPayload, _ := os.ReadFile("testdata/ggn_get_torrent_by_id_not_found.json")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(jsonPayload)
			return
		}

		// read json response
		jsonPayload, _ := os.ReadFile("testdata/ggn_get_torrent_by_id.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
			name: "get_by_id_2",
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

			c := NewClient(tt.fields.Url, tt.fields.APIKey)

			got, err := c.GetTorrentByID(context.Background(), tt.args.torrentID)
			if tt.wantErr && assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
