// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"cmp"
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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
	ProcessMultipleFromIndexer(ctx context.Context, releases []*domain.Release, indexer domain.IndexerMinimal) error
}

type TorznabJob struct {
	Feed       *domain.Feed
	Name       string
	Log        zerolog.Logger
	URL        string
	Client     torznabClient
	Repo       jobFeedRepo
	CacheRepo  jobFeedCacheRepo
	ReleaseSvc jobReleaseSvc

	attempts   int
	errors     []error
	hasFetched bool

	JobID int
}

type RefreshFeedJob interface {
	Run()
	RunE(ctx context.Context) error
}

type torznabClient interface {
	WithHTTPClient(client *http.Client)
	Search(ctx context.Context, query string, categories []int, offset int) (*torznab.SearchResponse, error)
}

func NewTorznabJob(feed *domain.Feed, name string, log zerolog.Logger, url string, client torznabClient, repo jobFeedRepo, cacheRepo jobFeedCacheRepo, releaseSvc jobReleaseSvc) RefreshFeedJob {
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
		j.Log.Error().Err(err).Msg("error fetching feed items")
		return errors.Wrap(err, "error getting feed items")
	}

	if len(items) == 0 {
		j.Log.Debug().Int("items_count", len(items)).Msg("found zero new items to process")
		return nil
	}

	j.Log.Debug().Int("items_count", len(items)).Msg("found new items to process")

	releases, err := j.processItems(items)
	if err != nil {
		j.Log.Error().Err(err).Msg("error processing items")
		return errors.Wrap(err, "error processing items")
	}

	// process all new releases
	go j.ReleaseSvc.ProcessMultipleFromIndexer(context.WithoutCancel(ctx), releases, j.Feed.Indexer)

	return nil
}

func (j *TorznabJob) processItems(items []torznab.FeedItem) ([]*domain.Release, error) {
	releases := make([]*domain.Release, 0)
	now := time.Now()
	for _, item := range items {
		j.Log.Trace().Str("item", item.Title).Msg("processing item..")

		if j.Feed.MaxAge > 0 {
			if item.PubDate.After(minValidPubDate) {
				if !isNewerThanMaxAge(j.Feed.MaxAge, item.PubDate.Time, now) {
					j.Log.Debug().Str("item", item.Title).Int("feed_max_age", j.Feed.MaxAge).Time("pub_date", item.PubDate.Time).Msg("item is older than feed max age skipping")
					continue
				}
			}
		}

		rls := domain.NewRelease(j.Feed.Indexer)
		rls.Implementation = domain.ReleaseImplementationTorznab

		rls.TorrentName = item.Title
		rls.DownloadURL = item.Link

		if comments := strings.TrimSpace(item.Comments); rls.InfoURL == "" && comments != "" {
			if u, err := url.Parse(comments); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
				rls.InfoURL = comments
			}
		}

		if guid := strings.TrimSpace(item.GUID); rls.InfoURL == "" && guid != "" {
			if u, err := url.Parse(guid); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
				rls.InfoURL = guid
			}
		}

		if item.Enclosure != nil && item.Enclosure.Type == "application/x-bittorrent" {
			rls.DownloadURL = item.Enclosure.URL
		}

		if j.Feed.Settings != nil && j.Feed.Settings.DownloadType == domain.FeedDownloadTypeMagnet {
			// Jackett and Prowlarr publish the magnet in the magneturl attr but put their
			// own proxy url in the link, which only redirects to it. Mirror a non-magnet
			// url into MagnetURI for ResolveMagnetURI to follow, and leave it in
			// DownloadURL as the fallback for when it never resolves.
			if strings.HasPrefix(item.MagnetURI, domain.MagnetURIPrefix) {
				rls.MagnetURI = item.MagnetURI
			} else {
				rls.MagnetURI = rls.DownloadURL
			}
		}

		// a magnet can not be fetched over http, so it never belongs in DownloadURL,
		// whatever the feed is configured as
		if strings.HasPrefix(rls.DownloadURL, domain.MagnetURIPrefix) {
			rls.MagnetURI = cmp.Or(rls.MagnetURI, rls.DownloadURL)

			rls.DownloadURL = ""
		}

		rls.ParseString(item.Title)
		rls.Size = item.Size
		rls.Seeders = item.Seeders
		rls.Leechers = item.Leechers
		rls.Uploader = item.Author

		rls.MetaIMDB = item.ImdbId
		rls.MetaTMDB = parseMetaID(item.TmdbId)
		rls.MetaTVDB = parseMetaID(item.TvdbId)
		rls.PublishDate = item.PubDate.Time

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

// maxPages fetches a single page when the cache holds no boundary from a previous run,
// so a new or long-paused feed doesn't backfill old releases into filters.
func (j *TorznabJob) maxPages() int {
	if !j.hasFetched && !j.Feed.CacheCoversLastRun(time.Now()) {
		j.Log.Debug().Time("last_run", j.Feed.LastRun).Msg("no cache boundary from a previous run, fetching only the first page")
		return 1
	}

	return j.Feed.PaginationMaxPages()
}

func (j *TorznabJob) getFeed(ctx context.Context) ([]torznab.FeedItem, error) {
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

	p := paginator[torznab.FeedItem]{
		Log:       j.Log,
		CacheRepo: j.CacheRepo,
		FeedID:    j.Feed.ID,
		TTL:       j.Feed.CacheTTL(),
		MaxPages:  j.maxPages(),
		KeyOf:     func(item torznab.FeedItem) string { return item.GUID },
		TitleOf:   func(item torznab.FeedItem) string { return item.Title },
		PubDateOf: func(item torznab.FeedItem) time.Time { return item.PubDate.Time },
	}

	if j.Feed.MaxAge > 0 {
		p.Cutoff = time.Now().Add(-time.Duration(j.Feed.MaxAge) * time.Second)
	}

	items, err := p.Fetch(ctx, func(ctx context.Context, offset int) (page[torznab.FeedItem], error) {
		feed, err := j.Client.Search(ctx, "", j.Feed.Categories, offset)
		if err != nil {
			return page[torznab.FeedItem]{}, errors.Wrap(err, "error fetching feed items")
		}

		if offset == 0 {
			if err := j.Repo.UpdateLastRunWithData(ctx, j.Feed.ID, feed.Raw); err != nil {
				j.Log.Error().Err(err).Msg("error updating last run for feed")
			}
		}

		j.Log.Trace().Int("offset", offset).Int("items_count", len(feed.Items)).Msg("feed refresh fetched page")

		items := make([]torznab.FeedItem, 0, len(feed.Items))
		for _, item := range feed.Items {
			items = append(items, *item)
		}

		return page[torznab.FeedItem]{Limit: feed.Limit, Total: feed.Total, Items: items}, nil
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
