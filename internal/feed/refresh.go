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

// minPubDate guards max age against feeds that send a zero or epoch pub date for unknown.
var minPubDate = time.Date(1970, time.April, 1, 0, 0, 0, 0, time.UTC)

// refreshJob runs one feed refresh: fetch through the source, skip cached and too old items,
// and hand the rest to the release pipeline. A scheduled job lives across runs.
type refreshJob struct {
	log        zerolog.Logger
	feed       *domain.Feed
	src        source
	repo       jobFeedRepo
	cacheRepo  jobFeedCacheRepo
	releaseSvc jobReleaseSvc

	// hasFetched is set once a run has cached its first page, whose items bound the next run's
	// pagination even though feed.LastRun is still the snapshot from scheduling
	hasFetched bool
}

func (j *refreshJob) Run(ctx context.Context) error {
	entries, err := j.fetchNew(ctx)
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

	if j.feed.ProcessNewestFirst() {
		slices.Reverse(releases)
	}

	j.log.Debug().Int("items_count", len(releases)).Msg("found new items to process")

	go j.releaseSvc.ProcessMultipleFromIndexer(context.WithoutCancel(ctx), releases, j.feed.Indexer)

	return nil
}

// maxPages fetches a single page when the cache holds no boundary from a previous run, so a
// new or long-paused feed doesn't backfill old releases into filters.
func (j *refreshJob) maxPages() int {
	pages := j.feed.PaginationMaxPages()
	if pages > 1 && !j.hasFetched && !j.feed.CacheCoversLastRun(time.Now()) {
		j.log.Debug().Time("last_run", j.feed.LastRun).Msg("no cache boundary from a previous run, fetching only the first page")
		return 1
	}

	return pages
}

// fetchNew walks the feed page by page back to the first item cached by an earlier run, caching
// each page's new items as it goes, and returns them oldest first. Pagination stops at a cached
// key, the max age cutoff, the end of the feed, a page with no new items, max pages, or an error
// after the first page.
func (j *refreshJob) fetchNew(ctx context.Context) ([]entry, error) {
	maxPages := j.maxPages()

	var cutoff time.Time
	if j.feed.MaxAge > 0 {
		cutoff = time.Now().Add(-time.Duration(j.feed.MaxAge) * time.Second)
	}

	ttl := j.feed.CacheTTL()
	feedID := strconv.Itoa(j.feed.ID)

	var (
		fresh  []entry
		offset int
		seen   = make(map[string]struct{})
	)

	for page := 1; page <= maxPages; page++ {
		// errors after the first page keep the earlier pages, which are already cached and
		// would otherwise never be processed
		res, err := j.src.fetch(ctx, offset)
		if err != nil {
			if page == 1 {
				return nil, errors.Wrap(err, "could not fetch feed")
			}

			j.log.Error().Err(err).Int("page", page).Msg("could not fetch page, stopping pagination")
			break
		}

		if page == 1 {
			if err := j.repo.UpdateLastRunWithData(ctx, j.feed.ID, res.raw); err != nil {
				j.log.Error().Err(err).Msg("could not update last run for feed")
			}
		}

		unseen := make([]string, 0, len(res.entries))
		for _, e := range res.entries {
			if _, ok := seen[e.key]; !ok {
				unseen = append(unseen, e.key)
			}
		}

		existing, err := j.cacheRepo.ExistingItems(ctx, j.feed.ID, unseen)
		if err != nil {
			if page == 1 {
				return nil, errors.Wrap(err, "could not check existing items")
			}

			j.log.Error().Err(err).Int("page", page).Msg("could not check existing items, stopping pagination")
			break
		}

		boundary := false
		pageFresh := make([]entry, 0, len(unseen))
		toCache := make([]domain.FeedCacheItem, 0, len(unseen))

		for _, e := range res.entries {
			// pages can overlap when new items shift positions between requests, and a key can
			// be listed twice in one response, so keys seen earlier in this run are neither new
			// nor the previous run's boundary
			if _, ok := seen[e.key]; ok {
				continue
			}
			seen[e.key] = struct{}{}

			if existing[e.key] {
				boundary = true
				j.log.Trace().Str("item", e.title).Msg("cache item exists, skipping release")
				continue
			}

			j.log.Debug().Str("item", e.title).Msg("found new release")

			toCache = append(toCache, domain.FeedCacheItem{
				FeedId: feedID,
				Key:    e.key,
				Value:  []byte(e.title),
				TTL:    ttl,
			})

			pageFresh = append(pageFresh, e)
		}

		if len(toCache) > 0 {
			if err := j.cacheRepo.PutMany(ctx, toCache); err != nil {
				// the cache write is what marks the page as seen, so without it the boundary is
				// unreliable: drop the page and leave it for the next run instead of paginating
				// past uncached items
				j.log.Error().Err(err).Int("page", page).Msg("could not store items in feed cache, stopping pagination")
				break
			}
		}

		if page == 1 {
			j.hasFetched = true
		}

		fresh = append(fresh, pageFresh...)

		if boundary {
			break
		}

		if reachedCutoff(res.entries, cutoff) {
			j.log.Debug().Int("page", page).Msg("page reached feed max age, stopping pagination")
			break
		}

		if !res.hasMore(offset) {
			break
		}

		// a page that added nothing new didn't advance: it repeated items already seen this run
		// (an indexer that ignores the offset) or had no usable keys, and the next page would
		// return the same items
		if len(toCache) == 0 {
			j.log.Debug().Int("page", page).Msg("page produced no new items, stopping pagination")
			break
		}

		offset += res.items
	}

	// feeds list newest first
	slices.Reverse(fresh)

	return fresh, nil
}

// hasMore reports whether the feed has items past the page fetched at offset.
func (r *fetchResult) hasMore(offset int) bool {
	if r.items == 0 {
		return false
	}

	// indexers that ignore the limit param make every page look short, so a reported total is
	// the reliable end-of-feed signal when present
	if r.total > 0 {
		return offset+r.items < r.total
	}

	return r.limit > 0 && r.items >= r.limit
}

// reachedCutoff reports whether the oldest dated entry is older than cutoff; a zero cutoff never is.
func reachedCutoff(entries []entry, cutoff time.Time) bool {
	if cutoff.IsZero() {
		return false
	}

	var oldest time.Time
	for _, e := range entries {
		if !e.pubDate.After(minPubDate) {
			continue
		}

		if oldest.IsZero() || e.pubDate.Before(oldest) {
			oldest = e.pubDate
		}
	}

	return !oldest.IsZero() && oldest.Before(cutoff)
}

func isNewerThanMaxAge(maxAge int, item, now time.Time) bool {
	return item.After(now.Add(time.Duration(-maxAge) * time.Second))
}
