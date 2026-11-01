// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
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

// minValidPubDate is the earliest pubDate treated as real; indexers emit zero
// or epoch dates for items they have no date for.
var minValidPubDate = time.Date(1970, time.April, 1, 0, 0, 0, 0, time.UTC)

// page is one result page from a paginated torznab or newznab feed.
type page[T any] struct {
	// Limit is the page size requested from the indexer.
	Limit int
	// Total is the item count the indexer reports for the query, zero when unreported.
	Total int
	Items []T
}

// paginator walks a paginated feed back to the first item cached by a previous run.
type paginator[T any] struct {
	Log       zerolog.Logger
	CacheRepo jobFeedCacheRepo
	FeedID    int
	TTL       time.Time
	MaxPages  int
	// Cutoff stops pagination after a page reaching items older than it; zero disables it.
	Cutoff    time.Time
	KeyOf     func(item T) string
	TitleOf   func(item T) string
	PubDateOf func(item T) time.Time
}

// Fetch returns the new items across all pages, oldest first. Pagination stops
// at a cached key, the max age cutoff, an empty, short or final page, a page
// with no new items, MaxPages, or an error after the first page.
func (p *paginator[T]) Fetch(ctx context.Context, fetchPage func(ctx context.Context, offset int) (page[T], error)) ([]T, error) {
	var (
		items  []T
		offset int
		seen   = make(map[string]struct{})
	)

	for pageNum := 0; pageNum < p.MaxPages; pageNum++ {
		// errors after the first page keep the earlier pages, which are already
		// cached and would otherwise never be processed
		page, err := fetchPage(ctx, offset)
		if err != nil {
			if pageNum == 0 {
				return nil, err
			}

			p.Log.Error().Err(err).Int("page", pageNum+1).Msg("could not fetch page, stopping pagination")
			break
		}

		type keyedItem struct {
			item T
			key  string
		}

		keyedItems := make([]keyedItem, 0, len(page.Items))
		unseen := make([]string, 0, len(page.Items))
		for _, item := range page.Items {
			key := p.KeyOf(item)
			if key == "" {
				p.Log.Error().Str("title", p.TitleOf(item)).Msg("missing GUID from feed item")
				continue
			}

			keyedItems = append(keyedItems, keyedItem{item: item, key: key})
			if _, ok := seen[key]; !ok {
				unseen = append(unseen, key)
			}
		}

		existing, err := p.CacheRepo.ExistingItems(ctx, p.FeedID, unseen)
		if err != nil {
			if pageNum == 0 {
				return nil, errors.Wrap(err, "could not check existing items")
			}

			p.Log.Error().Err(err).Int("page", pageNum+1).Msg("could not check existing items, stopping pagination")
			break
		}

		boundary := false
		toCache := make([]domain.FeedCacheItem, 0, len(unseen))

		for _, k := range keyedItems {
			// pages can overlap when new items shift positions between requests,
			// so keys seen earlier in this run are not run boundaries
			if _, ok := seen[k.key]; ok {
				continue
			}
			seen[k.key] = struct{}{}

			if existing[k.key] {
				boundary = true
				p.Log.Trace().Str("item", p.TitleOf(k.item)).Msg("cache item exists, skipping release..")
				continue
			}

			p.Log.Debug().Str("item", p.TitleOf(k.item)).Msg("found new release")

			toCache = append(toCache, domain.FeedCacheItem{
				FeedId: strconv.Itoa(p.FeedID),
				Key:    k.key,
				Value:  []byte(p.TitleOf(k.item)),
				TTL:    p.TTL,
			})

			items = append(items, k.item)
		}

		if len(toCache) > 0 {
			if err := p.CacheRepo.PutMany(ctx, toCache); err != nil {
				// the cache write is what marks the page as seen, so without it
				// the boundary state is unreliable: drop the page and leave it
				// for the next run instead of paginating past un-cached items
				p.Log.Error().Err(err).Int("page", pageNum+1).Msg("cache.PutMany: error storing items in cache, stopping pagination")

				items = items[:len(items)-len(toCache)]
				break
			}
		}

		if boundary {
			break
		}

		if p.reachedCutoff(page.Items) {
			p.Log.Debug().Int("page", pageNum+1).Msg("page reached feed max age, stopping pagination")
			break
		}

		if len(page.Items) == 0 {
			break
		}

		// indexers that ignore the limit param make every page look short, so a
		// reported total is the reliable end-of-feed signal when present
		if page.Total > 0 {
			if offset+len(page.Items) >= page.Total {
				break
			}
		} else if len(page.Items) < page.Limit {
			break
		}

		// a page that added nothing new didn't advance: it repeated items
		// already seen this run (an indexer that ignores the offset) or had no
		// usable GUIDs, and the next page would return the same items
		if len(toCache) == 0 {
			p.Log.Debug().Int("page", pageNum+1).Msg("page produced no new items, stopping pagination")
			break
		}

		offset += len(page.Items)
	}

	slices.Reverse(items)

	return items, nil
}

// reachedCutoff reports whether the oldest dated item on the page is older than Cutoff.
func (p *paginator[T]) reachedCutoff(items []T) bool {
	if p.Cutoff.IsZero() || p.PubDateOf == nil {
		return false
	}

	var oldest time.Time
	for _, item := range items {
		pubDate := p.PubDateOf(item)
		if !pubDate.After(minValidPubDate) {
			continue
		}

		if oldest.IsZero() || pubDate.Before(oldest) {
			oldest = pubDate
		}
	}

	return !oldest.IsZero() && oldest.Before(p.Cutoff)
}
