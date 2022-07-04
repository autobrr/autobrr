package sonarr

import (
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/rs/zerolog"
)

func Test_client_Push(t *testing.T) {
	// disable logger
	zerolog.SetGlobalLevel(zerolog.Disabled)
	log.SetOutput(ioutil.Discard)

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
		jsonPayload, _ := ioutil.ReadFile("testdata/release_push_response.json")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
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
					APIKey:    "",
					BasicAuth: false,
					Username:  "",
					Password:  "",
				},
			},
			args: args{release: Release{
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
			args: args{release: Release{
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

			rejections, err := c.Push(tt.args.release)
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
	log.SetOutput(ioutil.Discard)

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
		jsonPayload, _ := ioutil.ReadFile("testdata/system_status_response.json")
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

			got, err := c.Test()
			if tt.wantErr && assert.Error(t, err) {
				assert.EqualErrorf(t, err, tt.expectedErr, "Error should be: %v, got: %v", tt.wantErr, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
