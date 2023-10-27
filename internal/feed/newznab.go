// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/newznab"

	"github.com/rs/zerolog"
)

type NewznabJob struct {
	Feed              *domain.Feed
	Name              string
	IndexerIdentifier string
	Log               zerolog.Logger
	URL               string
	Client            newznab.Client
	Repo              domain.FeedRepo
	CacheRepo         domain.FeedCacheRepo
	ReleaseSvc        release.Service
	SchedulerSvc      scheduler.Service

	attempts int
	errors   []error

	JobID int
}

func NewNewznabJob(feed *domain.Feed, name string, indexerIdentifier string, log zerolog.Logger, url string, client newznab.Client, repo domain.FeedRepo, cacheRepo domain.FeedCacheRepo, releaseSvc release.Service) *NewznabJob {
	return &NewznabJob{
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

func (j *NewznabJob) Run() {
	ctx := context.Background()

	if err := j.process(ctx); err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("newznab process error")

		j.errors = append(j.errors, err)
	}

	j.attempts = 0
	j.errors = j.errors[:0]
}

func (j *NewznabJob) process(ctx context.Context) error {
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
		rls.InfoURL = item.GUID
		rls.Implementation = domain.ReleaseImplementationNewznab
		rls.Protocol = domain.ReleaseProtocolNzb

		// parse size bytes string
		rls.ParseSizeBytesString(item.Size)

		rls.ParseString(item.Title)

		if item.Enclosure != nil {
			if item.Enclosure.Type == "application/x-nzb" {
				rls.DownloadURL = item.Enclosure.Url
			}
		}

		// map newznab categories ID and Name into rls.Categories
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

func (j *NewznabJob) getFeed(ctx context.Context) ([]newznab.FeedItem, error) {
	// get feed
	feed, err := j.Client.GetFeed(ctx)
	if err != nil {
		j.Log.Error().Err(err).Msgf("error fetching feed items")
		return nil, errors.Wrap(err, "error fetching feed items")
	}

	if err := j.Repo.UpdateLastRunWithData(ctx, j.Feed.ID, feed.Raw); err != nil {
		j.Log.Error().Err(err).Msgf("error updating last run for feed id: %v", j.Feed.ID)
	}

	j.Log.Debug().Msgf("refreshing feed: %s, found (%d) items", j.Name, len(feed.Channel.Items))

	items := make([]newznab.FeedItem, 0)
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
