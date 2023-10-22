// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
)

type TorznabJob struct {
	Feed              *domain.Feed
	Name              string
	IndexerIdentifier string
	Log               zerolog.Logger
	URL               string
	Client            torznab.Client
	Repo              domain.FeedRepo
	CacheRepo         domain.FeedCacheRepo
	ReleaseSvc        release.Service
	SchedulerSvc      scheduler.Service

	attempts int
	errors   []error

	JobID int
}

func NewTorznabJob(feed *domain.Feed, name string, indexerIdentifier string, log zerolog.Logger, url string, client torznab.Client, repo domain.FeedRepo, cacheRepo domain.FeedCacheRepo, releaseSvc release.Service) *TorznabJob {
	return &TorznabJob{
		Feed:              feed,
		Name:              name,
		IndexerIdentifier: indexerIdentifier,
		Log:               log,
		URL:               url,
		Client:            client,
		Repo:              repo,
		CacheRepo:         cacheRepo,
		ReleaseSvc:        releaseSvc,
	}
}

func (j *TorznabJob) Run() {
	ctx := context.Background()

	if err := j.process(ctx); err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("torznab process error")

		j.errors = append(j.errors, err)
	}

	j.attempts = 0
	j.errors = j.errors[:0]
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

	releases := make([]*domain.Release, 0)
	now := time.Now()
	for _, item := range items {
		if j.Feed.MaxAge > 0 {
			if item.PubDate.After(time.Date(1970, time.April, 1, 0, 0, 0, 0, time.UTC)) {
				if !isNewerThanMaxAge(j.Feed.MaxAge, item.PubDate.Time, now) {
					continue
				}
			}
		}

		rls := domain.NewRelease(j.IndexerIdentifier)

		rls.TorrentName = item.Title
		rls.DownloadURL = item.Link
		rls.Implementation = domain.ReleaseImplementationTorznab

		// parse size bytes string
		rls.ParseSizeBytesString(item.Size)

		rls.ParseString(item.Title)

		if j.Feed.Settings != nil && j.Feed.Settings.DownloadType == domain.FeedDownloadTypeMagnet {
			rls.MagnetURI = item.Link
			rls.DownloadURL = ""
		}

		// Get freeleech percentage between 0 - 100. The value is ignored if
		// an error occurrs
		freeleechPercentage, err := parseFreeleechTorznab(item)
		if err != nil {
			j.Log.Debug().Err(err).Msgf("error parsing torznab freeleech")
		} else {
			if freeleechPercentage == 100 {
				// Release is 100% freeleech
				rls.Freeleech = true
				rls.Bonus = []string{"Freeleech"}
			}

			rls.FreeleechPercent = freeleechPercentage
			if bonus := mapFreeleechToBonus(freeleechPercentage); bonus != "" {
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

	// process all new releases
	go j.ReleaseSvc.ProcessMultiple(releases)

	return nil
}

// Parse the downloadvolumefactor attribute. The returned value is the percentage
// of downloaded data that does NOT count towards a user's total download amount.
func parseFreeleechTorznab(item torznab.FeedItem) (int, error) {
	for _, attr := range item.Attributes {
		if attr.Name == "downloadvolumefactor" {
			// Parse the value as decimal number
			downloadVolumeFactor, err := strconv.ParseFloat(attr.Value, 64)
			if err != nil {
				return 0, err
			}

			// Values below 0.0 and above 1.0 are rejected
			if downloadVolumeFactor < 0 || downloadVolumeFactor > 1 {
				return 0, errors.New("invalid downloadvolumefactor: %s", attr.Value)
			}

			// Multiply by 100 to convert from ratio to percentage and round it
			// to the nearest integer value
			downloadPercentage := math.Round(downloadVolumeFactor * 100)

			// To convert from download percentage to freeleech percentage the
			// value is inverted
			freeleechPercentage := 100 - int(downloadPercentage)

			return freeleechPercentage, nil
		}
	}

	return 0, nil
}

// Maps a freeleech percentage of 25, 50, 75 or 100 to a bonus.
func mapFreeleechToBonus(percentage int) string {
	switch percentage {
	case 25:
		return "Freeleech25"
	case 50:
		return "Freeleech50"
	case 75:
		return "Freeleech75"
	case 100:
		return "Freeleech100"
	default:
		return ""
	}
}

func (j *TorznabJob) getFeed(ctx context.Context) ([]torznab.FeedItem, error) {
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

	sort.SliceStable(feed.Channel.Items, func(i, j int) bool {
		return feed.Channel.Items[i].PubDate.After(feed.Channel.Items[j].PubDate.Time)
	})

	toCache := make([]domain.FeedCacheItem, 0)

	// set ttl to 1 month
	ttl := time.Now().AddDate(0, 1, 0)

	for _, i := range feed.Channel.Items {
		i := i

		if i.GUID == "" {
			j.Log.Error().Msgf("missing GUID from feed: %s", j.Feed.Name)
			continue
		}

		exists, err := j.CacheRepo.Exists(j.Feed.ID, i.GUID)
		if err != nil {
			j.Log.Error().Err(err).Msg("could not check if item exists")
			continue
		}
		if exists {
			j.Log.Trace().Msgf("cache item exists, skipping release: %s", i.Title)
			continue
		}

		j.Log.Debug().Msgf("found new release: %s", i.Title)

		toCache = append(toCache, domain.FeedCacheItem{
			FeedId: strconv.Itoa(j.Feed.ID),
			Key:    i.GUID,
			Value:  []byte(i.Title),
			TTL:    ttl,
		})

		// only append if we successfully added to cache
		items = append(items, *i)
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
