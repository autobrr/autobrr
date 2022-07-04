package red

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestREDClient_GetTorrentByID(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)

	key := "mock-key"

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// request validation logic
		apiKey := r.Header.Get("Authorization")
		if apiKey != key {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write(nil)
			return
		}

		if !strings.Contains(r.RequestURI, "29991962") {
			jsonPayload, _ := ioutil.ReadFile("testdata/get_torrent_by_id_not_found.json")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			w.Write(jsonPayload)
			return
		}

		// read json response
		jsonPayload, _ := ioutil.ReadFile("testdata/get_torrent_by_id.json")
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
			args: args{torrentID: "29991962"},
			want: &domain.TorrentBasic{
				Id:       "29991962",
				InfoHash: "B2BABD3A361EAFC6C4E9142C422DF7DDF5D7E163",
				Size:     "527749302",
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
			wantErr: "could not get torrent by id: 100002: bad id parameter",
		},
		{
			name: "get_by_id_3",
			fields: fields{
				Url:    ts.URL,
				APIKey: "",
			},
			args:    args{torrentID: "100002"},
			want:    nil,
			wantErr: "could not get torrent by id: 100002: unauthorized: bad credentials",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewClient(tt.fields.Url, tt.fields.APIKey)

			got, err := c.GetTorrentByID(tt.args.torrentID)
			if tt.wantErr != "" && assert.Error(t, err) {
				assert.EqualErrorf(t, err, tt.wantErr, "Error should be: %v, got: %v", tt.wantErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
