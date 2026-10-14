// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"crypto/tls"
	"net/http"
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

	attempts   int
	errors     []error
	hasFetched bool

	JobID int
}

type newznabClient interface {
	WithHTTPClient(client *http.Client)
	Search(ctx context.Context, query string, categories []int, offset int) (*newznab.SearchResponse, error)
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
			if item.PubDate.After(minValidPubDate) {
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
		rls.MetaTMDB = parseMetaID(item.TmdbId)
		rls.MetaTVDB = parseMetaID(item.TvdbId)
		rls.PublishDate = item.PubDate.Time

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

// maxPages fetches a single page when the cache holds no boundary from a previous run,
// so a new or long-paused feed doesn't backfill old releases into filters.
func (j *NewznabJob) maxPages() int {
	if !j.hasFetched && !j.Feed.CacheCoversLastRun(time.Now()) {
		j.Log.Debug().Time("last_run", j.Feed.LastRun).Msg("no cache boundary from a previous run, fetching only the first page")
		return 1
	}

	return j.Feed.PaginationMaxPages()
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

	p := paginator[newznab.FeedItem]{
		Log:       j.Log,
		CacheRepo: j.CacheRepo,
		FeedID:    j.Feed.ID,
		TTL:       j.Feed.CacheTTL(),
		MaxPages:  j.maxPages(),
		KeyOf:     func(item newznab.FeedItem) string { return item.GUID },
		TitleOf:   func(item newznab.FeedItem) string { return item.Title },
		PubDateOf: func(item newznab.FeedItem) time.Time { return item.PubDate.Time },
	}

	if j.Feed.MaxAge > 0 {
		p.Cutoff = time.Now().Add(-time.Duration(j.Feed.MaxAge) * time.Second)
	}

	items, err := p.Fetch(ctx, func(ctx context.Context, offset int) (page[newznab.FeedItem], error) {
		feed, err := j.Client.Search(ctx, "", j.Feed.Categories, offset)
		if err != nil {
			return page[newznab.FeedItem]{}, errors.Wrap(err, "error fetching feed items")
		}

		if offset == 0 {
			if err := j.Repo.UpdateLastRunWithData(ctx, j.Feed.ID, feed.Raw); err != nil {
				j.Log.Error().Err(err).Msg("error updating last run for feed")
			}
		}

		j.Log.Trace().Int("offset", offset).Int("items_count", len(feed.Items)).Msg("feed refresh fetched page")

		items := make([]newznab.FeedItem, 0, len(feed.Items))
		for _, item := range feed.Items {
			items = append(items, *item)
		}

		return page[newznab.FeedItem]{Limit: feed.Limit, Total: feed.Total, Items: items}, nil
	})
	if err != nil {
		return nil, err
	}

	j.hasFetched = true

	if len(items) == 0 {
		j.Log.Trace().Msg("feed refresh found zero items")
	}

	// send to filters
	return items, nil
}
