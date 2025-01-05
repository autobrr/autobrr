// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
)

type TorznabJob struct {
	Feed         *domain.Feed
	Name         string
	Log          zerolog.Logger
	URL          string
	Client       torznab.Client
	Repo         domain.FeedRepo
	CacheRepo    domain.FeedCacheRepo
	ReleaseSvc   release.Service
	SchedulerSvc scheduler.Service

	attempts int
	errors   []error

	JobID int
}

type FeedJob interface {
	Run()
	RunE(ctx context.Context) error
}

func NewTorznabJob(feed *domain.Feed, name string, log zerolog.Logger, url string, client torznab.Client, repo domain.FeedRepo, cacheRepo domain.FeedCacheRepo, releaseSvc release.Service) FeedJob {
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

		rls := domain.NewRelease(domain.IndexerMinimal{ID: j.Feed.Indexer.ID, Name: j.Feed.Indexer.Name, Identifier: j.Feed.Indexer.Identifier, IdentifierExternal: j.Feed.Indexer.IdentifierExternal})
		rls.Implementation = domain.ReleaseImplementationTorznab

		rls.TorrentName = item.Title
		rls.DownloadURL = item.Link

		// parse size bytes string
		rls.ParseSizeBytesString(item.Size)

		rls.ParseString(item.Title)

		rls.Seeders, err = parseIntAttribute(item, "seeders")
		if err != nil {
			rls.Seeders = 0
		}

		var peers, err = parseIntAttribute(item, "peers")

		rls.Leechers = peers - rls.Seeders
		if err != nil {
			rls.Leechers = 0
		}

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

func parseIntAttribute(item torznab.FeedItem, attrName string) (int, error) {
	for _, attr := range item.Attributes {
		if attr.Name == attrName {
			// Parse the value as decimal number
			intValue, err := strconv.Atoi(attr.Value)
			if err != nil {
				return 0, err
			}
			return intValue, err
		}
	}
	return 0, nil
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

	sort.SliceStable(feed.Channel.Items, func(i, j int) bool {
		return feed.Channel.Items[i].PubDate.After(feed.Channel.Items[j].PubDate.Time)
	})

	toCache := make([]domain.FeedCacheItem, 0)

	// set ttl to 1 month
	ttl := time.Now().AddDate(0, 1, 0)

	for _, item := range feed.Channel.Items {
		if item.GUID == "" {
			j.Log.Error().Msgf("missing GUID from feed: %s", j.Feed.Name)
			continue
		}

		exists, err := j.CacheRepo.Exists(j.Feed.ID, item.GUID)
		if err != nil {
			j.Log.Error().Err(err).Msg("could not check if item exists")
			continue
		}
		if exists {
			j.Log.Trace().Msgf("cache item exists, skipping release: %s", item.Title)
			continue
		}

		j.Log.Debug().Msgf("found new release: %s", item.Title)

		toCache = append(toCache, domain.FeedCacheItem{
			FeedId: strconv.Itoa(j.Feed.ID),
			Key:    item.GUID,
			Value:  []byte(item.Title),
			TTL:    ttl,
		})

		// only append if we successfully added to cache
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
