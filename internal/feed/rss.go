// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"encoding/xml"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog"
)

var (
	rxpSize      = regexp.MustCompile(`(?mi)(([0-9.]+)\s*(b|kb|kib|kilobyte|mb|mib|megabyte|gb|gib|gigabyte|tb|tib|terabyte))`)
	rxpFreeleech = regexp.MustCompile(`(?mi)(\bfreeleech\b)`)
	rxpHTML      = regexp.MustCompile(`(?mi)<.*?>`)
)

type RSSJob struct {
	Feed              *domain.Feed
	Name              string
	IndexerIdentifier string
	Log               zerolog.Logger
	URL               string
	Repo              domain.FeedRepo
	CacheRepo         domain.FeedCacheRepo
	ReleaseSvc        release.Service
	Timeout           time.Duration

	attempts int
	errors   []error

	JobID int
}

func NewRSSJob(feed *domain.Feed, name string, indexerIdentifier string, log zerolog.Logger, url string, repo domain.FeedRepo, cacheRepo domain.FeedCacheRepo, releaseSvc release.Service, timeout time.Duration) FeedJob {
	return &RSSJob{
		Feed:              feed,
		Name:              name,
		IndexerIdentifier: indexerIdentifier,
		Log:               log,
		URL:               url,
		Repo:              repo,
		CacheRepo:         cacheRepo,
		ReleaseSvc:        releaseSvc,
		Timeout:           timeout,
	}
}

func (j *RSSJob) Run() {
	ctx := context.Background()

	if err := j.RunE(ctx); err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("rss feed process error")

		j.errors = append(j.errors, err)
	}

	j.attempts = 0
	j.errors = j.errors[:0]
}

func (j *RSSJob) RunE(ctx context.Context) error {
	if err := j.process(ctx); err != nil {
		j.Log.Err(err).Msg("rss feed process error")
		return err
	}

	return nil
}

func (j *RSSJob) process(ctx context.Context) error {
	items, err := j.getFeed(ctx)
	if err != nil {
		j.Log.Error().Err(err).Msgf("error fetching rss feed items")
		return errors.Wrap(err, "error getting rss feed items")
	}

	j.Log.Debug().Msgf("found (%d) new items to process", len(items))

	if len(items) == 0 {
		return nil
	}

	releases := make([]*domain.Release, 0)

	for _, item := range items {
		item := item
		j.Log.Debug().Msgf("item: %v", item.Title)

		rls := j.processItem(item)
		if rls != nil {
			releases = append(releases, rls)
		}
	}

	// process all new releases
	go j.ReleaseSvc.ProcessMultiple(releases)

	return nil
}

func (j *RSSJob) processItem(item *gofeed.Item) *domain.Release {
	now := time.Now()

	if j.Feed.MaxAge > 0 {
		if item.PublishedParsed != nil && item.PublishedParsed.After(time.Date(1970, time.April, 1, 0, 0, 0, 0, time.UTC)) {
			if !isNewerThanMaxAge(j.Feed.MaxAge, *item.PublishedParsed, now) {
				return nil
			}
		}
	}

	rls := domain.NewRelease(j.IndexerIdentifier)
	rls.Implementation = domain.ReleaseImplementationRSS

	rls.ParseString(item.Title)

	if j.Feed.Settings != nil && j.Feed.Settings.DownloadType == domain.FeedDownloadTypeMagnet {
		rls.MagnetURI = item.Link
		rls.DownloadURL = ""
	}

	if len(item.Enclosures) > 0 {
		e := item.Enclosures[0]
		if e.Type == "application/x-bittorrent" && e.URL != "" {
			rls.DownloadURL = e.URL
		}
		if e.Length != "" && e.Length != "39399" {
			rls.ParseSizeBytesString(e.Length)
		}
	}

	if rls.DownloadURL == "" && item.Link != "" {
		rls.DownloadURL = item.Link
	}

	if rls.DownloadURL != "" {
		// handle no baseurl with only relative url
		// grab url from feed url and create full url
		if parsedURL, _ := url.Parse(rls.DownloadURL); parsedURL != nil && len(parsedURL.Hostname()) == 0 {
			if parentURL, _ := url.Parse(j.URL); parentURL != nil {
				parentURL.Path, parentURL.RawPath = "", ""

				// unescape the query params for max compatibility
				escapedUrl, _ := url.QueryUnescape(parentURL.JoinPath(rls.DownloadURL).String())
				rls.DownloadURL = escapedUrl
			}
		}
	}

	for _, v := range item.Categories {
		rls.Categories = append(rls.Categories, item.Categories...)

		if len(rls.Category) != 0 {
			rls.Category += ", "
		}

		rls.Category += v
	}

	for _, v := range item.Authors {
		if len(rls.Uploader) != 0 {
			rls.Uploader += ", "
		}

		rls.Uploader += v.Name
	}

	// When custom->size and enclosures->size differ, `ParseSizeBytesString` will pick the largest one.
	if size, ok := item.Custom["size"]; ok {
		rls.ParseSizeBytesString(size)
	}

	// additional size parsing
	// some feeds have a fixed size for enclosure so lets check for custom elements
	// and parse size from there if it differs
	if customTorrent, ok := item.Custom["torrent"]; ok {
		var element itemCustomElement
		if err := xml.Unmarshal([]byte("<torrent>"+customTorrent+"</torrent>"), &element); err != nil {
			j.Log.Error().Err(err).Msg("could not unmarshal item.Custom.Torrent")
		}

		if element.ContentLength > 0 {
			if uint64(element.ContentLength) > rls.Size {
				rls.Size = uint64(element.ContentLength)
			}
		}

		if rls.TorrentHash == "" && element.InfoHash != "" {
			rls.TorrentHash = element.InfoHash
		}
	}

	// basic freeleech parsing
	if isFreeleech([]string{item.Title, item.Description}) {
		rls.Freeleech = true
		rls.Bonus = []string{"Freeleech"}
	}

	if item.Description != "" {
		rls.Description = item.Description

		if rls.Size == 0 {
			readSizeFromDescription(item.Description, rls)
			j.Log.Trace().Msgf("Set new size %d from description", rls.Size)
		}
	}

	// add cookie to release for download if needed
	if j.Feed.Cookie != "" {
		rls.RawCookie = j.Feed.Cookie
	}

	return rls
}

func (j *RSSJob) getFeed(ctx context.Context) (items []*gofeed.Item, err error) {
	ctx, cancel := context.WithTimeout(ctx, j.Timeout)
	defer cancel()

	feed, err := NewFeedParser(j.Timeout, j.Feed.Cookie).ParseURLWithContext(ctx, j.URL)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching rss feed items")
	}

	// get feed as JSON string
	feedData := feed.String()

	if err := j.Repo.UpdateLastRunWithData(ctx, j.Feed.ID, feedData); err != nil {
		j.Log.Error().Err(err).Msgf("error updating last run for feed id: %v", j.Feed.ID)
	}

	j.Log.Debug().Msgf("refreshing rss feed: %v, found (%d) items", j.Name, len(feed.Items))

	if len(feed.Items) == 0 {
		return
	}

	//sort.Sort(feed)

	toCache := make([]domain.FeedCacheItem, 0)

	// set ttl to 1 month
	ttl := time.Now().AddDate(0, 1, 0)

	for _, i := range feed.Items {
		item := i

		key := item.GUID
		if len(key) == 0 {
			key = item.Link
			if len(key) == 0 {
				key = item.Title
			}
		}

		exists, err := j.CacheRepo.Exists(j.Feed.ID, key)
		if err != nil {
			j.Log.Error().Err(err).Msg("could not check if item exists")
			continue
		}
		if exists {
			j.Log.Trace().Msgf("cache item exists, skipping release: %s", item.Title)
			continue
		}

		j.Log.Debug().Msgf("found new release: %s", i.Title)

		toCache = append(toCache, domain.FeedCacheItem{
			FeedId: strconv.Itoa(j.Feed.ID),
			Key:    key,
			Value:  []byte(i.Title),
			TTL:    ttl,
		})

		// only append if we successfully added to cache
		items = append(items, item)
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
	return
}

func isNewerThanMaxAge(maxAge int, item, now time.Time) bool {
	// now minus max age
	nowMaxAge := now.Add(time.Duration(-maxAge) * time.Second)

	if item.After(nowMaxAge) {
		return true
	}

	return false
}

// isFreeleech basic freeleech parsing
func isFreeleech(str []string) bool {
	for _, s := range str {
		match := rxpFreeleech.FindAllString(s, -1)

		if len(match) > 0 {
			return true
		}
	}

	return false
}

// readSizeFromDescription get size from description
func readSizeFromDescription(str string, r *domain.Release) {
	clean := rxpHTML.ReplaceAllString(str, " ")
	for _, sz := range rxpSize.FindAllString(clean, -1) {
		r.ParseSizeBytesString(sz)
	}
}

// itemCustomElement
// used for some feeds like Aviztas network
type itemCustomElement struct {
	ContentLength int64  `xml:"contentLength"`
	InfoHash      string `xml:"infoHash"`
}
