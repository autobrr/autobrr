// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"slices"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/newznab"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
)

type jobFeedRepo interface {
	UpdateLastRunWithData(ctx context.Context, feedID int, data string) error
}

type jobFeedCacheRepo interface {
	ExistingItems(ctx context.Context, feedId int, keys []string) (map[string]bool, error)
	PutMany(ctx context.Context, items []domain.FeedCacheItem) error
}

type jobReleaseSvc interface {
	ProcessMultipleFromIndexer(ctx context.Context, releases []*domain.Release, indexer domain.IndexerMinimal) error
}

// source fetches one feed format and maps its items to releases; everything format
// independent lives in refreshJob.
type source interface {
	fetch(ctx context.Context) (*fetchResult, error)
}

type fetchResult struct {
	// raw is stored as the feed's last run data
	raw     string
	entries []entry
}

type entry struct {
	key     string
	title   string
	pubDate time.Time
	release func() *domain.Release
}

// minPubDate guards max age against feeds that send a zero or epoch pub date for unknown.
var minPubDate = time.Date(1970, time.April, 1, 0, 0, 0, 0, time.UTC)

func newSource(f *domain.Feed, log zerolog.Logger) (source, error) {
	client, err := newHTTPClient(f)
	if err != nil {
		return nil, err
	}

	if f.UseProxy && f.Proxy != nil {
		log.Debug().Str("proxy", f.Proxy.Name).Msg("using proxy for feed")
	}

	switch f.Type {
	case string(domain.FeedTypeTorznab):
		c := torznab.NewClient(torznab.Config{Host: f.URL, ApiKey: f.ApiKey, Log: log})
		c.WithHTTPClient(client)

		return &torznabSource{log: log, feed: f, client: c}, nil

	case string(domain.FeedTypeNewznab):
		c := newznab.NewClient(newznab.Config{Host: f.URL, ApiKey: f.ApiKey, Log: log})
		c.WithHTTPClient(client)

		return &newznabSource{log: log, feed: f, client: c}, nil

	case string(domain.FeedTypeRSS):
		return &rssSource{log: log, feed: f, client: client}, nil

	default:
		return nil, errors.New("unsupported feed type: %s", f.Type)
	}
}

// refreshJob runs one feed refresh: fetch through the source, skip cached and too old items,
// and hand the rest to the release pipeline.
type refreshJob struct {
	log        zerolog.Logger
	feed       *domain.Feed
	src        source
	repo       jobFeedRepo
	cacheRepo  jobFeedCacheRepo
	releaseSvc jobReleaseSvc
}

func (j *refreshJob) RunE(ctx context.Context) error {
	res, err := j.src.fetch(ctx)
	if err != nil {
		return errors.Wrap(err, "could not fetch feed")
	}

	if err := j.repo.UpdateLastRunWithData(ctx, j.feed.ID, res.raw); err != nil {
		j.log.Error().Err(err).Msg("could not update last run for feed")
	}

	entries, err := j.uncached(ctx, res.entries)
	if err != nil {
		return err
	}

	now := time.Now()
	releases := make([]*domain.Release, 0, len(entries))

	for _, e := range entries {
		if j.feed.MaxAge > 0 && e.pubDate.After(minPubDate) && !isNewerThanMaxAge(j.feed.MaxAge, e.pubDate, now) {
			j.log.Debug().Str("item", e.title).Int("feed_max_age", j.feed.MaxAge).Time("pub_date", e.pubDate).Msg("item is older than feed max age, skipping")
			continue
		}

		releases = append(releases, e.release())
	}

	if len(releases) == 0 {
		j.log.Debug().Msg("found zero new items to process")
		return nil
	}

	j.log.Debug().Int("items_count", len(releases)).Msg("found new items to process")

	go j.releaseSvc.ProcessMultipleFromIndexer(context.WithoutCancel(ctx), releases, j.feed.Indexer)

	return nil
}

// uncached returns the entries not seen on an earlier run, oldest first, and stores them in the
// feed cache so the next run skips them.
func (j *refreshJob) uncached(ctx context.Context, entries []entry) ([]entry, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(entries))
	for _, e := range entries {
		keys = append(keys, e.key)
	}

	existing, err := j.cacheRepo.ExistingItems(ctx, j.feed.ID, keys)
	if err != nil {
		return nil, errors.Wrap(err, "could not check existing items")
	}

	ttl := j.feed.CacheTTL()
	seen := make(map[string]struct{}, len(entries))
	fresh := make([]entry, 0)
	toCache := make([]domain.FeedCacheItem, 0)

	// feeds list newest first
	for _, e := range slices.Backward(entries) {
		if existing[e.key] {
			j.log.Trace().Str("item", e.title).Msg("cache item exists, skipping release")
			continue
		}

		// a key listed twice in one response would be processed and cached twice
		if _, ok := seen[e.key]; ok {
			continue
		}
		seen[e.key] = struct{}{}

		j.log.Debug().Str("item", e.title).Msg("found new release")

		toCache = append(toCache, domain.FeedCacheItem{
			FeedId: strconv.Itoa(j.feed.ID),
			Key:    e.key,
			Value:  []byte(e.title),
			TTL:    ttl,
		})

		fresh = append(fresh, e)
	}

	if len(toCache) > 0 {
		if err := j.cacheRepo.PutMany(ctx, toCache); err != nil {
			j.log.Error().Err(err).Msg("could not store items in feed cache")
		}
	}

	return fresh, nil
}

func isNewerThanMaxAge(maxAge int, item, now time.Time) bool {
	return item.After(now.Add(time.Duration(-maxAge) * time.Second))
}
