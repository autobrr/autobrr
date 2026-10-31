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

func (s *stubSource) fetch(context.Context, int) (*fetchResult, error) {
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
			wantCached: []string{"c", "b", "a"},
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
			wantCached: []string{"new", "old", "undated"},
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

func TestRefreshJob_RunE_processOrder(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name  string
		order domain.FeedProcessOrder
		want  []string
	}{
		{name: "default is oldest first", want: []string{"a", "b", "c"}},
		{name: "oldest first", order: domain.FeedProcessOrderOldestFirst, want: []string{"a", "b", "c"}},
		{name: "newest first", order: domain.FeedProcessOrderNewestFirst, want: []string{"c", "b", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releaseSvc := &recordingReleaseSvc{got: make(chan []*domain.Release, 1)}

			j := &refreshJob{
				log:  zerolog.Nop(),
				feed: &domain.Feed{ID: 4, Settings: &domain.FeedSettingsJSON{ProcessOrder: tt.order}},
				src: &stubSource{res: &fetchResult{entries: []entry{
					newEntry("c", now),
					newEntry("b", now.Add(-time.Minute)),
					newEntry("a", now.Add(-2*time.Minute)),
				}}},
				repo:       &recordingFeedRepo{},
				cacheRepo:  &recordingCacheRepo{},
				releaseSvc: releaseSvc,
			}

			require.NoError(t, j.Run(t.Context()))

			select {
			case releases := <-releaseSvc.got:
				var handed []string
				for _, rls := range releases {
					handed = append(handed, rls.TorrentName)
				}
				assert.Equal(t, tt.want, handed)
			case <-time.After(time.Second):
				t.Fatal("releases were not handed off")
			}
		})
	}
}

// pagedSource serves one page per offset, falling back to fallback for unknown offsets.
type pagedSource struct {
	limit      int
	total      int
	pages      map[int][]entry
	fallback   []entry
	errOffsets map[int]error
	offsets    []int
}

func (s *pagedSource) fetch(_ context.Context, offset int) (*fetchResult, error) {
	s.offsets = append(s.offsets, offset)

	if err := s.errOffsets[offset]; err != nil {
		return nil, err
	}

	entries, ok := s.pages[offset]
	if !ok {
		entries = s.fallback
	}

	return &fetchResult{entries: entries, items: len(entries), limit: s.limit, total: s.total}, nil
}

// statefulCacheRepo makes keys written by PutMany visible to later ExistingItems calls.
type statefulCacheRepo struct {
	existing map[string]bool
	putErr   error
	putCalls [][]string

	// existingErrOnCall and putErrOnCall fail only the nth call (1-based), 0 never
	existingErrOnCall int
	putErrOnCall      int
	existingCalls     int
	putAttempts       int
}

func newStatefulCacheRepo(keys ...string) *statefulCacheRepo {
	r := &statefulCacheRepo{existing: map[string]bool{}}
	for _, k := range keys {
		r.existing[k] = true
	}
	return r
}

func (r *statefulCacheRepo) ExistingItems(_ context.Context, _ int, keys []string) (map[string]bool, error) {
	r.existingCalls++
	if r.existingErrOnCall == r.existingCalls {
		return nil, errors.New("existing items failed")
	}

	res := make(map[string]bool, len(keys))
	for _, key := range keys {
		if r.existing[key] {
			res[key] = true
		}
	}

	return res, nil
}

func (r *statefulCacheRepo) PutMany(_ context.Context, items []domain.FeedCacheItem) error {
	r.putAttempts++
	if r.putErr != nil {
		return r.putErr
	}
	if r.putErrOnCall == r.putAttempts {
		return errors.New("put many failed")
	}

	keys := make([]string, 0, len(items))
	for _, item := range items {
		r.existing[item.Key] = true
		keys = append(keys, item.Key)
	}

	r.putCalls = append(r.putCalls, keys)

	return nil
}

func TestRefreshJob_fetchNew_pagination(t *testing.T) {
	item := func(key string) entry {
		return newEntry(key, time.Time{})
	}

	keys := func(entries []entry) []string {
		out := make([]string, 0, len(entries))
		for _, e := range entries {
			out = append(out, e.key)
		}
		return out
	}

	newJob := func(src source, cache jobFeedCacheRepo, maxPages int) *refreshJob {
		return &refreshJob{
			log: zerolog.Nop(),
			feed: &domain.Feed{
				ID:       4,
				Settings: &domain.FeedSettingsJSON{MaxPages: maxPages},
				LastRun:  time.Now().Add(-time.Hour),
			},
			src:       src,
			repo:      &recordingFeedRepo{},
			cacheRepo: cache,
		}
	}

	twoFullPages := func() map[int][]entry {
		return map[int][]entry{
			0: {item("new-1"), item("new-2"), item("new-3")},
			3: {item("new-4"), item("new-5"), item("new-6")},
			6: {item("new-7")},
		}
	}

	t.Run("stops at the cached boundary on page 2", func(t *testing.T) {
		src := &pagedSource{
			limit: 3,
			pages: map[int][]entry{
				0: {item("new-1"), item("new-2"), item("new-3")},
				3: {item("new-4"), item("old-1"), item("old-2")},
			},
		}
		cache := newStatefulCacheRepo("old-1", "old-2")

		got, err := newJob(src, cache, domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		// oldest first: the second page's items precede the first page's
		assert.Equal(t, []string{"new-4", "new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 3}, src.offsets)

		require.Len(t, cache.putCalls, 2)
		assert.Equal(t, []string{"new-1", "new-2", "new-3"}, cache.putCalls[0])
		assert.Equal(t, []string{"new-4"}, cache.putCalls[1])
	})

	t.Run("fully cached first page stops immediately", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: map[int][]entry{0: {item("old-1"), item("old-2"), item("old-3")}}}

		got, err := newJob(src, newStatefulCacheRepo("old-1", "old-2", "old-3"), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Empty(t, got)
		assert.Equal(t, []int{0}, src.offsets)
	})

	t.Run("short page is the end of the feed", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: map[int][]entry{0: {item("new-1"), item("new-2")}}}

		got, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0}, src.offsets)
	})

	t.Run("source without pagination fetches one page", func(t *testing.T) {
		src := &pagedSource{fallback: []entry{item("new-1"), item("new-2")}}

		got, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0}, src.offsets)
	})

	t.Run("catch-up backfills until the short page", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}

		got, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-7", "new-6", "new-5", "new-4", "new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 3, 6}, src.offsets)
	})

	t.Run("max pages caps the backfill", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}

		got, err := newJob(src, newStatefulCacheRepo(), 2).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Len(t, got, 6)
		assert.Equal(t, []int{0, 3}, src.offsets)
	})

	t.Run("unset max pages fetches only the first page", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}

		_, err := newJob(src, newStatefulCacheRepo(), 0).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0}, src.offsets)
	})

	t.Run("indexer ignoring the offset stops after a repeated page", func(t *testing.T) {
		src := &pagedSource{limit: 3, fallback: []entry{item("new-1"), item("new-2"), item("new-3")}}

		got, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		// the repeated items are deduped within the run, no release twice
		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 3}, src.offsets)
	})

	t.Run("repeated page with shifted order stops on zero new items", func(t *testing.T) {
		src := &pagedSource{
			limit: 3,
			pages: map[int][]entry{
				0: {item("new-1"), item("new-2"), item("new-3")},
				3: {item("new-2"), item("new-1"), item("new-3")},
			},
		}

		got, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 3}, src.offsets)
	})

	t.Run("reported total paginates past short pages", func(t *testing.T) {
		src := &pagedSource{
			limit: 3,
			total: 7,
			pages: map[int][]entry{
				0: {item("new-1"), item("new-2")},
				2: {item("new-3"), item("new-4")},
				4: {item("new-5"), item("new-6")},
				6: {item("new-7")},
			},
		}

		got, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-7", "new-6", "new-5", "new-4", "new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 2, 4, 6}, src.offsets)
	})

	t.Run("overlapping pages are deduped within the run", func(t *testing.T) {
		src := &pagedSource{
			limit: 3,
			pages: map[int][]entry{
				0: {item("new-1"), item("new-2"), item("new-3")},
				3: {item("new-3"), item("new-4")},
			},
		}

		got, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		// new-3 shifted onto page 2 but must not be processed twice, and the overlap must not
		// count as the previous run's boundary
		assert.Equal(t, []string{"new-4", "new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 3}, src.offsets)
	})

	t.Run("cache write failure drops the page and stops paginating", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}
		cache := newStatefulCacheRepo()
		cache.putErr = errors.New("disk full")

		got, err := newJob(src, cache, domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		// nothing was cached, so nothing may be processed this run
		assert.Empty(t, got)
		assert.Empty(t, cache.putCalls)
		assert.Equal(t, []int{0}, src.offsets)
	})

	t.Run("fetch error on page 2 keeps the cached first page", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages(), errOffsets: map[int]error{3: errors.New("429 too many requests")}}

		got, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 3}, src.offsets)
	})

	t.Run("cache lookup error on page 2 keeps the cached first page", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}
		cache := newStatefulCacheRepo()
		cache.existingErrOnCall = 2

		got, err := newJob(src, cache, domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 3}, src.offsets)
	})

	t.Run("cache write error on page 2 keeps the cached first page", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}
		cache := newStatefulCacheRepo()
		cache.putErrOnCall = 2

		got, err := newJob(src, cache, domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, keys(got))
		assert.Len(t, cache.putCalls, 1)
		assert.Equal(t, []int{0, 3}, src.offsets)
	})

	t.Run("fetch error on page 1 fails the run", func(t *testing.T) {
		src := &pagedSource{limit: 3, errOffsets: map[int]error{0: errors.New("503")}}

		j := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages)
		j.feed.LastRun = time.Time{}

		_, err := j.fetchNew(t.Context())
		require.Error(t, err)

		assert.False(t, j.hasFetched, "a failed first fetch must retry the single page next run")
	})

	t.Run("cache write failure on the first run keeps the one-page limit", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}
		cache := newStatefulCacheRepo()
		cache.putErr = errors.New("disk full")

		j := newJob(src, cache, domain.MaxFeedPages)
		j.feed.LastRun = time.Time{}

		got, err := j.fetchNew(t.Context())
		require.NoError(t, err)
		assert.Empty(t, got)
		assert.False(t, j.hasFetched, "nothing was cached, so there is no boundary for the next run")

		// with no cached boundary, a full page count would backfill old releases
		cache.putErr = nil

		got, err = j.fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0, 0}, src.offsets)
		assert.True(t, j.hasFetched)
	})

	t.Run("first run fetches only the first page", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}
		j := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages)
		j.feed.LastRun = time.Time{}

		got, err := j.fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"new-3", "new-2", "new-1"}, keys(got))
		assert.Equal(t, []int{0}, src.offsets)

		// the job's own run now marks the cache boundary, so the next run paginates
		src.pages = map[int][]entry{
			0: {item("next-1"), item("next-2"), item("next-3")},
			3: {item("next-4"), item("new-1"), item("new-2")},
		}

		got, err = j.fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []string{"next-4", "next-3", "next-2", "next-1"}, keys(got))
		assert.Equal(t, []int{0, 0, 3}, src.offsets)
	})

	t.Run("last run older than the cache ttl fetches only the first page", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}
		j := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages)
		j.feed.LastRun = time.Now().AddDate(0, 0, -40)

		_, err := j.fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0}, src.offsets)
	})

	t.Run("last run within the cache ttl paginates", func(t *testing.T) {
		src := &pagedSource{limit: 3, pages: twoFullPages()}
		j := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages)
		j.feed.LastRun = time.Now().AddDate(0, 0, -3)

		_, err := j.fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0, 3, 6}, src.offsets)
	})

	t.Run("max age stops after a page reaching older items", func(t *testing.T) {
		now := time.Now()
		src := &pagedSource{
			limit: 3,
			pages: map[int][]entry{
				0: {newEntry("new-1", now), newEntry("new-2", now.Add(-time.Hour)), newEntry("new-3", now.Add(-7*24*time.Hour))},
				3: {newEntry("new-4", now.Add(-8*24*time.Hour))},
			},
		}
		j := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages)
		j.feed.MaxAge = 3600 * 24

		got, err := j.fetchNew(t.Context())
		require.NoError(t, err)

		// the page's items still reach Run, which applies the age filter
		assert.Len(t, got, 3)
		assert.Equal(t, []int{0}, src.offsets)
	})

	t.Run("max age reached on page 2 stops after page 2", func(t *testing.T) {
		now := time.Now()
		src := &pagedSource{
			limit: 2,
			pages: map[int][]entry{
				0: {newEntry("new-1", now), newEntry("new-2", now.Add(-time.Hour))},
				2: {newEntry("new-3", now.Add(-2*time.Hour)), newEntry("new-4", now.Add(-48*time.Hour))},
				4: {newEntry("new-5", now.Add(-72*time.Hour)), newEntry("new-6", now.Add(-96*time.Hour))},
			},
		}
		j := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages)
		j.feed.MaxAge = 3600 * 24

		_, err := j.fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0, 2}, src.offsets)
	})

	t.Run("max age disabled paginates past old items", func(t *testing.T) {
		old := time.Now().AddDate(0, 0, -30)
		src := &pagedSource{
			limit: 2,
			pages: map[int][]entry{
				0: {newEntry("new-1", old), newEntry("new-2", old)},
				2: {newEntry("new-3", old)},
			},
		}

		_, err := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages).fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0, 2}, src.offsets)
	})

	t.Run("items without a valid pub date never reach max age", func(t *testing.T) {
		src := &pagedSource{
			limit: 2,
			pages: map[int][]entry{
				0: {newEntry("new-1", time.Time{}), newEntry("new-2", time.Unix(0, 0))},
				2: {newEntry("new-3", time.Time{})},
			},
		}
		j := newJob(src, newStatefulCacheRepo(), domain.MaxFeedPages)
		j.feed.MaxAge = 3600

		_, err := j.fetchNew(t.Context())
		require.NoError(t, err)

		assert.Equal(t, []int{0, 2}, src.offsets)
	})
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
