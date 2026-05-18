package feed

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestTorznabJob_RunE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryType := r.URL.Query().Get("t")
		switch queryType {
		case "search":
			payload, err := os.ReadFile("testdata/torznab/torznab_response.xml")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.Write(payload)
			break

		case "caps":
			payload, err := os.ReadFile("testdata/torznab/caps_response.xml")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/xml")
			w.Write(payload)
			break
		}
	}))
	defer srv.Close()

	type fields struct {
		Feed       *domain.Feed
		Name       string
		Log        zerolog.Logger
		URL        string
		Client     *torznab.Client
		Repo       jobFeedRepo
		CacheRepo  jobFeedCacheRepo
		ReleaseSvc jobReleaseSvc
		attempts   int
		errors     []error
		JobID      int
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "test",
			fields: fields{
				Name: "test",
				Log:  zerolog.New(io.Discard),
				Feed: &domain.Feed{
					MaxAge: 0,
					Indexer: domain.IndexerMinimal{
						ID:                 0,
						Name:               "Mock Feed",
						Identifier:         "mock-feed",
						IdentifierExternal: "Mock Indexer",
					},
				},
				URL:        srv.URL,
				Client:     torznab.NewClient(torznab.Config{Host: srv.URL}),
				Repo:       &mockFeedRepo{},
				CacheRepo:  &mockFeedCacheRepo{},
				ReleaseSvc: &mockReleaseSvc{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &TorznabJob{
				Feed:       tt.fields.Feed,
				Name:       tt.fields.Name,
				Log:        tt.fields.Log,
				URL:        tt.fields.URL,
				Client:     tt.fields.Client,
				Repo:       tt.fields.Repo,
				CacheRepo:  tt.fields.CacheRepo,
				ReleaseSvc: tt.fields.ReleaseSvc,
				attempts:   tt.fields.attempts,
				errors:     tt.fields.errors,
				JobID:      tt.fields.JobID,
			}
			err := j.RunE(t.Context())
			assert.NoError(t, err)
		})
	}
}
