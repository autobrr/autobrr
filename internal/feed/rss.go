package feed

import (
	"github.com/mmcdole/gofeed"
	"sort"
	"time"
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
)

type RSSJob struct {
	Name              string
	IndexerIdentifier string
	Log               zerolog.Logger
	URL               string
	Repo              domain.FeedCacheRepo
	ReleaseSvc        release.Service

	attempts int
	errors   []error

	JobID int
}

func NewRSSJob(name string, indexerIdentifier string, log zerolog.Logger, url string, repo domain.FeedCacheRepo, releaseSvc release.Service) *RSSJob {
	return &RSSJob{
		Name:              name,
		IndexerIdentifier: indexerIdentifier,
		Log:               log,
		URL:               url,
		Repo:              repo,
		ReleaseSvc:        releaseSvc,
	}
}

func (j *RSSJob) Run() {
	if err := j.process(); err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("rss feed process error")

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
		rls := domain.NewRelease(j.IndexerIdentifier)

		rls.TorrentName = item.Title
		rls.TorrentURL = item.Link
		for _, v := range item.Categories {
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

		rls.Implementation = domain.ReleaseImplementationRSS

		// parse size bytes string
		if sz, ok := item.Custom["size"]; ok {
			rls.ParseSizeBytesString(sz)
		}

		rls.ParseString(item.Title)

		releases = append(releases, rls)
	}

	// process all new releases
	go j.ReleaseSvc.ProcessMultiple(releases)

	return nil
}

func (j *RSSJob) getFeed() (items []*gofeed.Item, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	feed, err := gofeed.NewParser().ParseURLWithContext(j.URL, ctx) // there's an RSS specific parser as well.
	if err != nil {
		j.Log.Error().Err(err).Msgf("error fetching rss feed items")
		return nil, errors.Wrap(err, "error fetching rss feed items")
	}

	j.Log.Debug().Msgf("refreshing rss feed: %v, found (%d) items", j.Name, len(feed.Items))

	if len(feed.Items) == 0 {
		return
	}

	sort.Sort(feed)

	for _, i := range feed.Items {
		s := i.GUID
		if len(s) == 0 {
			s = i.Title
			if len(s) == 0 {
				continue
			}
		}

		exists, err := j.Repo.Exists(j.Name, s)
		if err != nil {
			j.Log.Error().Err(err).Msg("could not check if item exists")
			continue
		}
		if exists {
			j.Log.Trace().Msgf("cache item exists, skipping release: %v", i.Title)
			continue
		}

		// set ttl to 1 month
		ttl := time.Now().AddDate(0, 1, 0)

		if err := j.Repo.Put(j.Name, s, []byte(i.Title), ttl); err != nil {
			j.Log.Error().Stack().Err(err).Str("entry", s).Msg("cache.Put: error storing item in cache")
			continue
		}

		// only append if we successfully added to cache
		items = append(items, i)
	}

	// send to filters
	return
}
