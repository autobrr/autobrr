// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceRefreshEmitsOutcome(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantType  events.EventType
		wantError string
	}{
		{name: "success", wantType: events.FeedRefreshSuccess},
		{name: "error", err: errors.New("unexpected status code: 503"), wantType: events.FeedRefreshError, wantError: "could not fetch feed: unexpected status code: 503"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bus := events.NewEventBus(zerolog.Nop())

			var got []events.FeedRefreshEvent
			bus.OnFeedRefresh(func(_ context.Context, event events.FeedRefreshEvent) error {
				got = append(got, event)
				return nil
			})

			s := &Service{log: zerolog.Nop(), eventBus: bus}
			f := &domain.Feed{ID: 4, Name: "Mock Indexer"}

			job := &refreshJob{
				log:       zerolog.Nop(),
				feed:      f,
				src:       &stubSource{res: &fetchResult{}, err: tt.err},
				repo:      &recordingFeedRepo{},
				cacheRepo: &recordingCacheRepo{},
			}

			err := s.refresh(t.Context(), job)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}

			require.Len(t, got, 1)
			assert.Equal(t, tt.wantType, got[0].Type)
			assert.Equal(t, f, got[0].Feed)
			assert.Equal(t, tt.wantError, got[0].Error)
		})
	}
}

type stubSource struct {
	res *fetchResult
	err error
}

func (s *stubSource) fetch(context.Context) (*fetchResult, error) {
	return s.res, s.err
}

type recordingFeedRepo struct {
	raw []string
}

func (r *recordingFeedRepo) UpdateLastRunWithData(_ context.Context, _ int, data string) error {
	r.raw = append(r.raw, data)
	return nil
}

type recordingCacheRepo struct {
	existing    map[string]bool
	existingErr error
	stored      []domain.FeedCacheItem
}

func (r *recordingCacheRepo) ExistingItems(context.Context, int, []string) (map[string]bool, error) {
	return r.existing, r.existingErr
}

func (r *recordingCacheRepo) PutMany(_ context.Context, items []domain.FeedCacheItem) error {
	r.stored = append(r.stored, items...)
	return nil
}

type recordingReleaseSvc struct {
	got chan []*domain.Release
}

func (r *recordingReleaseSvc) ProcessMultipleFromIndexer(_ context.Context, releases []*domain.Release, _ domain.IndexerMinimal) error {
	r.got <- releases
	return nil
}

func newEntry(key string, pubDate time.Time) entry {
	return entry{
		key:     key,
		title:   key,
		pubDate: pubDate,
		release: func() *domain.Release { return &domain.Release{TorrentName: key} },
	}
}

func TestRefreshJob_RunE(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		maxAge      int
		res         *fetchResult
		fetchErr    error
		existing    map[string]bool
		existingErr error
		wantErr     bool
		wantRaw     []string
		wantCached  []string
		wantHanded  []string
	}{
		{
			name: "new items are cached and handed off oldest first",
			res: &fetchResult{raw: "raw", entries: []entry{
				newEntry("c", now),
				newEntry("b", now.Add(-time.Minute)),
				newEntry("a", now.Add(-2*time.Minute)),
			}},
			wantRaw:    []string{"raw"},
			wantCached: []string{"a", "b", "c"},
			wantHanded: []string{"a", "b", "c"},
		},
		{
			name:       "cached items are skipped",
			res:        &fetchResult{entries: []entry{newEntry("b", now), newEntry("a", now)}},
			existing:   map[string]bool{"a": true},
			wantRaw:    []string{""},
			wantCached: []string{"b"},
			wantHanded: []string{"b"},
		},
		{
			name:       "a key listed twice is handled once",
			res:        &fetchResult{entries: []entry{newEntry("a", now), newEntry("a", now)}},
			wantRaw:    []string{""},
			wantCached: []string{"a"},
			wantHanded: []string{"a"},
		},
		{
			name:   "items older than max age are cached but not handed off",
			maxAge: 3600,
			res: &fetchResult{entries: []entry{
				newEntry("new", now.Add(-time.Minute)),
				newEntry("old", now.Add(-2*time.Hour)),
				newEntry("undated", time.Time{}),
			}},
			wantRaw:    []string{""},
			wantCached: []string{"undated", "old", "new"},
			wantHanded: []string{"undated", "new"},
		},
		{
			name:       "nothing new is not handed off",
			res:        &fetchResult{entries: []entry{newEntry("a", now)}},
			existing:   map[string]bool{"a": true},
			wantRaw:    []string{""},
			wantCached: nil,
		},
		{
			name:     "fetch error stops the run",
			fetchErr: errors.New("unexpected status code: 503"),
			wantErr:  true,
		},
		{
			name:        "cache lookup error stops the run",
			res:         &fetchResult{entries: []entry{newEntry("a", now)}},
			existingErr: errors.New("database is locked"),
			wantErr:     true,
			wantRaw:     []string{""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &recordingFeedRepo{}
			cacheRepo := &recordingCacheRepo{existing: tt.existing, existingErr: tt.existingErr}
			releaseSvc := &recordingReleaseSvc{got: make(chan []*domain.Release, 1)}

			j := &refreshJob{
				log:        zerolog.Nop(),
				feed:       &domain.Feed{ID: 4, MaxAge: tt.maxAge},
				src:        &stubSource{res: tt.res, err: tt.fetchErr},
				repo:       repo,
				cacheRepo:  cacheRepo,
				releaseSvc: releaseSvc,
			}

			err := j.Run(t.Context())
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.wantRaw, repo.raw, "last run data")

			var cached []string
			for _, item := range cacheRepo.stored {
				assert.Equal(t, "4", item.FeedId)
				cached = append(cached, item.Key)
			}
			assert.Equal(t, tt.wantCached, cached, "cached keys")

			if tt.wantHanded == nil {
				assert.Empty(t, releaseSvc.got, "nothing should be handed off")
				return
			}

			select {
			case releases := <-releaseSvc.got:
				var handed []string
				for _, rls := range releases {
					handed = append(handed, rls.TorrentName)
				}
				assert.Equal(t, tt.wantHanded, handed, "handed off releases")
			case <-time.After(time.Second):
				t.Fatal("releases were not handed off")
			}
		})
	}
}

func TestNewHTTPClientTimeout(t *testing.T) {
	proxyConf := &domain.Proxy{Enabled: true, Type: domain.ProxyTypeHTTP, Addr: "http://127.0.0.1:8888"}

	tests := []struct {
		name string
		feed domain.Feed
		want time.Duration
	}{
		{name: "direct", feed: domain.Feed{Timeout: 5}, want: 5 * time.Second},
		{name: "direct default", feed: domain.Feed{}, want: defaultTimeout},
		{name: "proxy", feed: domain.Feed{Timeout: 5, UseProxy: true, Proxy: proxyConf}, want: 5 * time.Second},
		{name: "proxy default", feed: domain.Feed{UseProxy: true, Proxy: proxyConf}, want: defaultTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := newHTTPClient(&tt.feed)
			require.NoError(t, err)

			assert.Equal(t, tt.want, client.Timeout)
		})
	}
}
