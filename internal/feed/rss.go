package feed

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog"
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

func NewRSSJob(feed *domain.Feed, name string, indexerIdentifier string, log zerolog.Logger, url string, repo domain.FeedRepo, cacheRepo domain.FeedCacheRepo, releaseSvc release.Service, timeout time.Duration) *RSSJob {
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
	if err := j.process(); err != nil {
		j.Log.Error().Err(err).Int("attempts", j.attempts).Msg("rss feed process error")

		j.errors = append(j.errors, err)
		return
	}

	j.attempts = 0
	j.errors = []error{}

	return
}

func (j *RSSJob) process() error {
	items, err := j.getFeed()
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
		if item.PublishedParsed != nil {
			if !isNewerThanMaxAge(j.Feed.MaxAge, *item.PublishedParsed, now) {
				return nil
			}
		}
	}

	rls := domain.NewRelease(j.IndexerIdentifier)
	rls.Implementation = domain.ReleaseImplementationRSS

	rls.ParseString(item.Title)

	if len(item.Enclosures) > 0 {
		e := item.Enclosures[0]
		if e.Type == "application/x-bittorrent" && e.URL != "" {
			rls.TorrentURL = e.URL
		}
		if e.Length != "" {
			rls.ParseSizeBytesString(e.Length)
		}
	}

	if rls.TorrentURL == "" && item.Link != "" {
		rls.TorrentURL = item.Link
	}

	if rls.TorrentURL != "" {
		// handle no baseurl with only relative url
		// grab url from feed url and create full url
		if parsedURL, _ := url.Parse(rls.TorrentURL); parsedURL != nil && len(parsedURL.Hostname()) == 0 {
			if parentURL, _ := url.Parse(j.URL); parentURL != nil {
				parentURL.Path, parentURL.RawPath = "", ""

				// unescape the query params for max compatibility
				escapedUrl, _ := url.QueryUnescape(parentURL.JoinPath(rls.TorrentURL).String())
				rls.TorrentURL = escapedUrl
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

	if rls.Size == 0 {
		// parse size bytes string
		if sz, ok := item.Custom["size"]; ok {
			rls.ParseSizeBytesString(sz)
		}
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
			if uint64(element.ContentLength) != rls.Size {
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

	// add cookie to release for download if needed
	if j.Feed.Cookie != "" {
		rls.RawCookie = j.Feed.Cookie
	}

	return rls
}

func (j *RSSJob) getFeed() (items []*gofeed.Item, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), j.Timeout)
	defer cancel()

	feed, err := NewFeedParser(j.Timeout, j.Feed.Cookie).ParseURLWithContext(ctx, j.URL)
	if err != nil {
		return nil, errors.Wrap(err, "error fetching rss feed items")
	}

	// get feed as JSON string
	feedData := feed.String()

	if err := j.Repo.UpdateLastRunWithData(context.Background(), j.Feed.ID, feedData); err != nil {
		j.Log.Error().Err(err).Msgf("error updating last run for feed id: %v", j.Feed.ID)
	}

	j.Log.Debug().Msgf("refreshing rss feed: %v, found (%d) items", j.Name, len(feed.Items))

	if len(feed.Items) == 0 {
		return
	}

	bucketKey := fmt.Sprintf("%v+%v", j.IndexerIdentifier, j.Name)

	//sort.Sort(feed)

	bucketCount, err := j.CacheRepo.GetCountByBucket(ctx, bucketKey)
	if err != nil {
		j.Log.Error().Err(err).Msg("could not check if item exists")
		return nil, err
	}

	// set ttl to 1 month
	ttl := time.Now().AddDate(0, 1, 0)

	for _, i := range feed.Items {
		item := i

		key := item.GUID
		if len(key) == 0 {
			key = item.Title
			if len(key) == 0 {
				continue
			}
		}

		exists, err := j.CacheRepo.Exists(bucketKey, key)
		if err != nil {
			j.Log.Error().Err(err).Msg("could not check if item exists")
			continue
		}
		if exists {
			j.Log.Trace().Msgf("cache item exists, skipping release: %v", item.Title)
			continue
		}

		if err := j.CacheRepo.Put(bucketKey, key, []byte(item.Title), ttl); err != nil {
			j.Log.Error().Err(err).Str("entry", key).Msg("cache.Put: error storing item in cache")
			continue
		}

		// first time we fetch the feed the cached bucket count will be 0
		// only append to items if it's bigger than 0, so we get new items only
		if bucketCount > 0 {
			items = append(items, item)
		}
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
		var re = regexp.MustCompile(`(?mi)(\bfreeleech\b)`)

		match := re.FindAllString(s, -1)

		if len(match) > 0 {
			return true
		}
	}

	return false
}

// itemCustomElement
// used for some feeds like Aviztas network
type itemCustomElement struct {
	ContentLength int64  `xml:"contentLength"`
	InfoHash      string `xml:"infoHash"`
}
