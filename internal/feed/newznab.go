// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"crypto/tls"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/newznab"

	"github.com/rs/zerolog"
)

type NewznabJob struct {
	Feed       *domain.Feed
	Name       string
	Log        zerolog.Logger
	URL        string
	Client     newznabClient
	Repo       jobFeedRepo
	CacheRepo  jobFeedCacheRepo
	ReleaseSvc jobReleaseSvc

	attempts int
	errors   []error

	JobID int
}

type newznabClient interface {
	WithHTTPClient(client *http.Client)
	Search(ctx context.Context, query string, categories []int) (*newznab.SearchResponse, error)
}

func NewNewznabJob(feed *domain.Feed, name string, log zerolog.Logger, url string, client newznabClient, repo jobFeedRepo, cacheRepo jobFeedCacheRepo, releaseSvc jobReleaseSvc) RefreshFeedJob {
	return &NewznabJob{
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

func (j *NewznabJob) Run() {
	ctx := context.Background()

	if err := j.RunE(ctx); err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("newznab process error")

		j.errors = append(j.errors, err)
	}

	j.attempts = 0
	j.errors = j.errors[:0]
}

func (j *NewznabJob) RunE(ctx context.Context) error {
	if err := j.process(ctx); err != nil {
		j.Log.Err(err).Msg("newznab process error")
		return err
	}

	return nil
}

func (j *NewznabJob) process(ctx context.Context) error {
	// get feed
	items, err := j.getFeed(ctx)
	if err != nil {
		return errors.Wrap(err, "error getting feed items")
	}

	if len(items) == 0 {
		j.Log.Debug().Int("items_count", len(items)).Msg("found zero new items to process")
		return nil
	}

	j.Log.Debug().Int("items_count", len(items)).Msg("found new items to process")

	releases, err := j.processItems(items)
	if err != nil {
		return errors.Wrap(err, "error processing items")
	}

	// process all new releases
	go j.ReleaseSvc.ProcessMultipleFromIndexer(context.WithoutCancel(ctx), releases, j.Feed.Indexer)

	return nil
}

func (j *NewznabJob) processItems(items []newznab.FeedItem) ([]*domain.Release, error) {
	releases := make([]*domain.Release, 0)
	now := time.Now()
	for _, item := range items {
		j.Log.Trace().Str("item", item.Title).Msg("processing item..")

		if j.Feed.MaxAge > 0 {
			if item.PubDate.After(time.Date(1970, time.April, 1, 0, 0, 0, 0, time.UTC)) {
				if !isNewerThanMaxAge(j.Feed.MaxAge, item.PubDate.Time, now) {
					j.Log.Debug().Str("item", item.Title).Int("feed_max_age", j.Feed.MaxAge).Time("pub_date", item.PubDate.Time).Msg("item is older than feed max age, skipping")
					continue
				}
			}
		}

		rls := domain.NewRelease(j.Feed.Indexer)
		rls.Implementation = domain.ReleaseImplementationNewznab
		rls.Protocol = domain.ReleaseProtocolNzb

		rls.TorrentName = item.Title
		rls.InfoURL = item.GUID

		rls.ParseString(item.Title)

		rls.MetaIMDB = item.ImdbId
		if item.TmdbId != "" {
			if tmdbId, err := strconv.Atoi(item.TmdbId); err == nil {
				rls.MetaTMDB = tmdbId
			}
		}

		rls.Size = item.Size

		if item.Enclosure != nil && item.Enclosure.Type == "application/x-nzb" {
			rls.DownloadURL = item.Enclosure.Url
			if rls.Size == 0 && item.Enclosure.Length > item.Size {
				rls.Size = item.Enclosure.Length
			}
		}

		if len(item.Categories) == 1 {
			rls.Category = item.Categories[0].Name
		}

		// map newznab categories ID and Name into rls.Categories
		// so we can filter on both ID and Name
		for _, category := range item.Categories {
			rls.Categories = append(rls.Categories, []string{category.Name, strconv.Itoa(category.ID)}...)
		}

		releases = append(releases, rls)
	}

	return releases, nil
}

func (j *NewznabJob) getFeed(ctx context.Context) ([]newznab.FeedItem, error) {
	// add proxy if enabled and exists
	if j.Feed.UseProxy && j.Feed.Proxy != nil {
		proxyClient, err := proxy.GetProxiedHTTPClient(j.Feed.Proxy)
		if err != nil {
			return nil, errors.Wrap(err, "could not get proxy client")
		}

		if j.Feed.TLSSkipVerify {
			if t, ok := proxyClient.Transport.(*http.Transport); ok {
				t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}
		}

		j.Client.WithHTTPClient(proxyClient)

		j.Log.Debug().Str("proxy", j.Feed.Proxy.Name).Msg("using proxy for feed")
	}

	// get feed
	feed, err := j.Client.Search(ctx, "", j.Feed.Categories)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching feed items")
	}

	if err := j.Repo.UpdateLastRunWithData(ctx, j.Feed.ID, feed.Raw); err != nil {
		j.Log.Error().Err(err).Msg("error updating last run for feed")
	}

	items := make([]newznab.FeedItem, 0)
	if len(feed.Items) == 0 {
		j.Log.Trace().Int("items_count", len(feed.Items)).Msg("feed refresh found zero items")
		return items, nil
	}

	j.Log.Trace().Int("items_count", len(feed.Items)).Msg("feed refresh found new items")

	//sort.SliceStable(feed.Channel.Items, func(i, j int) bool {
	//	return feed.Channel.Items[i].PubDate.After(feed.Channel.Items[j].PubDate.Time)
	//})

	// Collect all valid GUIDs first
	guidItemMap := make(map[string]*newznab.FeedItem)
	var guids []string

	for _, item := range feed.Items {
		if item.GUID == "" {
			j.Log.Error().Str("title", item.Title).Msg("item missing GUID")
			continue
		}

		guidItemMap[item.GUID] = item
		guids = append(guids, item.GUID)
	}

	// reverse order so oldest items are processed first
	slices.Reverse(guids)

	existingGuids, err := j.CacheRepo.ExistingItems(ctx, j.Feed.ID, guids)
	if err != nil {
		return nil, errors.Wrap(err, "could not get existing items from cache")
	}

	// set ttl to 1 month
	ttl := time.Now().AddDate(0, 1, 0)
	toCache := make([]domain.FeedCacheItem, 0)

	for _, guid := range guids {
		item := guidItemMap[guid]
		if existingGuids[guid] {
			j.Log.Trace().Str("item", item.Title).Msg("cache item exists, skipping release..")
			continue
		}

		j.Log.Debug().Str("item", item.Title).Msg("found new release")

		toCache = append(toCache, domain.FeedCacheItem{
			FeedId: strconv.Itoa(j.Feed.ID),
			Key:    guid,
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
