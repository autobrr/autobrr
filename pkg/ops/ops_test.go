package ops

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestOrpheusClient_GetTorrentByID(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	key := "mock-key"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// request validation logic
		apiKey := r.Header.Get("Authorization")
		if !strings.Contains(apiKey, key) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		if !strings.Contains(r.RequestURI, "2156788") {
			jsonPayload, _ := os.ReadFile("testdata/get_torrent_by_id_not_found.json")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(jsonPayload)
			return
		}

		// read json response
		jsonPayload, _ := os.ReadFile("testdata/get_torrent_by_id.json")
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
		wantErr string
	}{
		{
			name: "get_by_id_1",
			fields: fields{
				Url:    ts.URL,
				APIKey: key,
			},
			args: args{torrentID: "2156788"},
			want: &domain.TorrentBasic{
				Id:       "2156788",
				InfoHash: "",
				Size:     "255299244",
				Uploader: "uploader",
			},
			wantErr: "",
		},
		{
			name: "get_by_id_2",
			fields: fields{
				Url:    ts.URL,
				APIKey: key,
			},
			args:    args{torrentID: "100002"},
			want:    nil,
			wantErr: "could not get torrent by id: 100002: status code: 400 status: failure error: bad id parameter",
		},
		{
			name: "get_by_id_3",
			fields: fields{
				Url:    ts.URL,
				APIKey: "",
			},
			args:    args{torrentID: "100002"},
			want:    nil,
			wantErr: "could not get torrent by id: 100002: orpheus client missing API key!",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.fields.APIKey)
			c.UseURL(tt.fields.Url)

			got, err := c.GetTorrentByID(context.Background(), tt.args.torrentID)
			if tt.wantErr != "" && assert.Error(t, err) {
				assert.EqualErrorf(t, err, tt.wantErr, "Error should be: %v, got: %v", tt.wantErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
