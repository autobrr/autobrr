// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/pkg/errors"
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
	ProcessMultipleFromIndexer(releases []*domain.Release, indexer domain.IndexerMinimal) error
}

type TorznabJob struct {
	Feed       *domain.Feed
	Name       string
	Log        zerolog.Logger
	URL        string
	Client     torznab.Client
	Repo       jobFeedRepo
	CacheRepo  jobFeedCacheRepo
	ReleaseSvc jobReleaseSvc

	attempts int
	errors   []error

	JobID int
}

type RefreshFeedJob interface {
	Run()
	RunE(ctx context.Context) error
}

func NewTorznabJob(feed *domain.Feed, name string, log zerolog.Logger, url string, client torznab.Client, repo jobFeedRepo, cacheRepo jobFeedCacheRepo, releaseSvc jobReleaseSvc) RefreshFeedJob {
	return &TorznabJob{
		Feed:       feed,
		Name:       name,
		Log:        log,
		URL:        url,
		Client:     client,
		Repo:       repo,
		CacheRepo:  cacheRepo,
		ReleaseSvc: releaseSvc,
	}
}

func (j *TorznabJob) Run() {
	ctx := context.Background()

	if err := j.RunE(ctx); err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("torznab process error")

		j.errors = append(j.errors, err)
	}

	j.attempts = 0
	j.errors = j.errors[:0]
}

func (j *TorznabJob) RunE(ctx context.Context) error {
	if err := j.process(ctx); err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("torznab process error")
		return err
	}

	return nil
}

func (j *TorznabJob) process(ctx context.Context) error {
	// get feed
	items, err := j.getFeed(ctx)
	if err != nil {
		j.Log.Error().Err(err).Msgf("error fetching feed items")
		return errors.Wrap(err, "error getting feed items")
	}

	j.Log.Debug().Msgf("found (%d) new items to process", len(items))

	if len(items) == 0 {
		return nil
	}

	releases, err := j.processItems(items)
	if err != nil {
		j.Log.Error().Err(err).Msgf("error processing items")
		return errors.Wrap(err, "error processing items")
	}

	// process all new releases
	go j.ReleaseSvc.ProcessMultipleFromIndexer(releases, j.Feed.Indexer)

	return nil
}

func (j *TorznabJob) processItems(items []torznab.FeedItem) ([]*domain.Release, error) {
	releases := make([]*domain.Release, 0)
	now := time.Now()
	for _, item := range items {
		j.Log.Trace().Str("item", item.Title).Msg("processing item..")

		if j.Feed.MaxAge > 0 {
			if item.PubDate.After(time.Date(1970, time.April, 1, 0, 0, 0, 0, time.UTC)) {
				if !isNewerThanMaxAge(j.Feed.MaxAge, item.PubDate.Time, now) {
					j.Log.Debug().Msgf("item is older than feed max age, skipping: %s", item.Title)
					continue
				}
			}
		}

		rls := domain.NewRelease(j.Feed.Indexer)
		rls.Implementation = domain.ReleaseImplementationTorznab

		rls.TorrentName = item.Title
		rls.DownloadURL = item.Link
		if j.Feed.Settings != nil && j.Feed.Settings.DownloadType == domain.FeedDownloadTypeMagnet {
			rls.MagnetURI = item.Link
			rls.DownloadURL = ""
		}

		rls.ParseString(item.Title)
		rls.Size = uint64(item.Size)
		rls.Seeders = item.Seeders
		rls.Leechers = item.Leechers
		rls.Uploader = item.Author

		// Get freeleech percentage between 0 - 100
		if freeleechPercentage := parseFreeleechTorznab(item.DownloadVolumeFactor); freeleechPercentage >= 0 {
			if freeleechPercentage == 100 {
				// Release is 100% freeleech
				rls.Freeleech = true
				rls.Bonus = []string{"Freeleech"}
			}

			rls.FreeleechPercent = freeleechPercentage
			if bonus, ok := mapFreeleechToBonus(freeleechPercentage); ok && bonus != "" {
				rls.Bonus = append(rls.Bonus, bonus)
			}
		}

		// map torznab categories ID and Name into rls.Categories
		// so we can filter on both ID and Name
		for _, category := range item.Categories {
			rls.Categories = append(rls.Categories, []string{category.Name, strconv.Itoa(category.ID)}...)
		}

		releases = append(releases, rls)
	}

	return releases, nil
}

//func parseIntAttribute(item torznab.FeedItem, attrName string) (int, error) {
//	for _, attr := range item.Attributes {
//		if attr.Name == attrName {
//			// Parse the value as decimal number
//			intValue, err := strconv.Atoi(attr.Value)
//			if err != nil {
//				return 0, err
//			}
//			return intValue, err
//		}
//	}
//	return 0, nil
//}

// Parse the downloadvolumefactor attribute. The returned value is the percentage
// of downloaded data that does NOT count towards a user's total download amount.
func parseFreeleechTorznab(factor float64) int {
	// Values below 0.0 and above 1.0 are rejected
	if factor < 0 || factor > 1 {
		return 0
	}

	// Multiply by 100 to convert from float to percentage and round it
	// to the nearest integer value
	downloadPercentage := math.Round(factor * 100)

	// To convert from download percentage to freeleech percentage the
	// value is inverted
	freeleechPercentage := 100 - int(downloadPercentage)

	return freeleechPercentage
}

// Maps a freeleech percentage of 25, 50, 75 or 100 to a bonus.
func mapFreeleechToBonus(percentage int) (string, bool) {
	if percentage <= 0 || percentage > 100 {
		return "", false
	}

	switch percentage {
	case 25:
		return "Freeleech25", true
	case 50:
		return "Freeleech50", true
	case 75:
		return "Freeleech75", true
	case 100:
		return "Freeleech100", true
	default:
		return fmt.Sprintf("Freeleech%d", percentage), false
	}
}

func (j *TorznabJob) getFeed(ctx context.Context) ([]torznab.FeedItem, error) {
	// add proxy if enabled and exists
	if j.Feed.UseProxy && j.Feed.Proxy != nil {
		proxyClient, err := proxy.GetProxiedHTTPClient(j.Feed.Proxy)
		if err != nil {
			return nil, errors.Wrap(err, "could not get proxy client")
		}

		j.Client.WithHTTPClient(proxyClient)

		j.Log.Debug().Msgf("using proxy %s for feed %s", j.Feed.Proxy.Name, j.Feed.Name)
	}

	// get feed
	feed, err := j.Client.FetchFeed(ctx)
	if err != nil {
		j.Log.Error().Err(err).Msgf("error fetching feed items")
		return nil, errors.Wrap(err, "error fetching feed items")
	}

	if err := j.Repo.UpdateLastRunWithData(ctx, j.Feed.ID, feed.Raw); err != nil {
		j.Log.Error().Err(err).Msgf("error updating last run for feed id: %v", j.Feed.ID)
	}

	j.Log.Debug().Msgf("refreshing feed: %v, found (%d) items", j.Name, len(feed.Channel.Items))

	items := make([]torznab.FeedItem, 0)
	if len(feed.Channel.Items) == 0 {
		return items, nil
	}

	// Collect all valid GUIDs first
	guidItemMap := make(map[string]*torznab.FeedItem)
	var guids []string

	for _, item := range feed.Channel.Items {
		if item.GUID == "" {
			j.Log.Error().Msgf("missing GUID from feed: %s", j.Feed.Name)
			continue
		}

		guidItemMap[item.GUID] = item
		guids = append(guids, item.GUID)
	}

	// reverse order so oldest items are processed first
	slices.Reverse(guids)

	// Batch check which GUIDs already exist in the cache
	existingGuids, err := j.CacheRepo.ExistingItems(ctx, j.Feed.ID, guids)
	if err != nil {
		j.Log.Error().Err(err).Msg("could not check existing items")
		return nil, errors.Wrap(err, "could not check existing items")
	}

	// set ttl to 1 month
	ttl := time.Now().AddDate(0, 1, 0)
	toCache := make([]domain.FeedCacheItem, 0)

	// Process items that don't exist in the cache
	for _, guid := range guids {
		item := guidItemMap[guid]
		if existingGuids[guid] {
			j.Log.Trace().Msgf("cache item exists, skipping release: %s", item.Title)
			continue
		}

		j.Log.Debug().Msgf("found new release: %s", item.Title)

		toCache = append(toCache, domain.FeedCacheItem{
			FeedId: strconv.Itoa(j.Feed.ID),
			Key:    guid,
			Value:  []byte(item.Title),
			TTL:    ttl,
		})

		// Add item to result list
		items = append(items, *item)
	}

	if len(toCache) > 0 {
		go func(items []domain.FeedCacheItem) {
			ctx := context.Background()
			if err := j.CacheRepo.PutMany(ctx, items); err != nil {
				j.Log.Error().Err(err).Msg("cache.PutMany: error storing items in cache")
			}
		}(toCache)
	}

	// send to filters
	return items, nil
}
