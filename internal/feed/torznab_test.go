package feed

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestTorznabJob_process(t *testing.T) {
	key := "mock-key"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestType := r.URL.Query().Get("t")

		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			if apiKey != key {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write(nil)
				return
			}
		}

		var payload []byte
		var err error

		switch requestType {
		case "caps":
			payload, err = os.ReadFile("testdata/torznab_caps_response.xml")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case "search":
			payload, err = os.ReadFile("testdata/torznab_response.xml")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/xml")
		w.Write(payload)
	}))
	defer srv.Close()

	type fields struct {
		Feed         *domain.Feed
		Name         string
		Log          zerolog.Logger
		URL          string
		Client       torznab.Client
		Repo         feedRepo
		CacheRepo    cacheRepo
		ReleaseSvc   releaseService
		SchedulerSvc scheduler.Service
		attempts     int
		errors       []error
		JobID        int
	}
	type args struct {
		ctx context.Context
	}

	// setup logger
	logger := log.With().Str("feed", "feenname").Logger()

	// setup torznab Client
	client := torznab.NewClient(torznab.Config{Host: srv.URL, ApiKey: key, Timeout: time.Second * 30})

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "test1",
			fields: fields{
				Feed: &domain.Feed{
					ID:   1,
					Name: "torznab feed",
					Indexer: domain.IndexerMinimal{
						ID:                 1,
						Name:               "Torznab",
						Identifier:         "torznab-rss",
						IdentifierExternal: "torznab rss (prowlarr)",
					},
					Type:         string(domain.FeedTypeTorznab),
					Enabled:      true,
					URL:          srv.URL + "/api",
					Interval:     0,
					Timeout:      0,
					MaxAge:       0,
					Capabilities: nil,
					ApiKey:       key,
					Cookie:       "",
					Settings:     nil,
					CreatedAt:    time.Time{},
					UpdatedAt:    time.Time{},
					IndexerID:    0,
					LastRun:      time.Time{},
					LastRunData:  "",
					NextRun:      time.Time{},
				},
				Name:         "feed",
				Log:          logger,
				URL:          srv.URL + "/api",
				Client:       client,
				Repo:         &mockFeedRepo{},
				CacheRepo:    &mockFeedCacheRepo{},
				ReleaseSvc:   &mockReleaseService{},
				SchedulerSvc: nil,
				attempts:     0,
				errors:       nil,
				JobID:        0,
			},
			args: args{context.Background()},
			wantErr: func(t assert.TestingT, err error, i ...interface{}) bool {
				return false
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &TorznabJob{
				Feed:         tt.fields.Feed,
				Name:         tt.fields.Name,
				Log:          tt.fields.Log,
				URL:          tt.fields.URL,
				Client:       tt.fields.Client,
				Repo:         tt.fields.Repo,
				CacheRepo:    tt.fields.CacheRepo,
				ReleaseSvc:   tt.fields.ReleaseSvc,
				SchedulerSvc: tt.fields.SchedulerSvc,
				attempts:     tt.fields.attempts,
				errors:       tt.fields.errors,
				JobID:        tt.fields.JobID,
			}
			tt.wantErr(t, j.process(tt.args.ctx), fmt.Sprintf("process(%v)", tt.args.ctx))
		})
	}
}
