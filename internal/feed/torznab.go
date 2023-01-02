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

	for _, item := range items {
		rls := domain.NewRelease(j.IndexerIdentifier)

		rls.TorrentName = item.Title
		rls.TorrentURL = item.Link
		rls.Implementation = domain.ReleaseImplementationTorznab

		// parse size bytes string
		rls.ParseSizeBytesString(item.Size)

		rls.ParseString(item.Title)

		if parseFreeleechTorznab(item) {
			rls.Freeleech = true
			rls.Bonus = []string{"Freeleech"}
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

func parseFreeleechTorznab(item torznab.FeedItem) bool {
	for _, attr := range item.Attributes {
		if attr.Name == "downloadvolumefactor" {
			if attr.Value == "0" {
				return true
			}
		}
	}

	return false
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

	for _, i := range feed.Channel.Items {
		if i.GUID == "" {
			j.Log.Error().Err(err).Msgf("missing GUID from feed: %s", j.Feed.Name)
			continue
		}

		exists, err := j.CacheRepo.Exists(j.Name, i.GUID)
		if err != nil {
			j.Log.Error().Err(err).Msg("could not check if item exists")
			continue
		}
		if exists {
			j.Log.Trace().Msgf("cache item exists, skipping release: %s", i.Title)
			continue
		}

		j.Log.Debug().Msgf("found new release: %s", i.Title)

		// set ttl to 1 month
		ttl := time.Now().AddDate(0, 1, 0)

		if err := j.CacheRepo.Put(j.Name, i.GUID, []byte(i.Title), ttl); err != nil {
			j.Log.Error().Stack().Err(err).Str("guid", i.GUID).Msg("cache.Put: error storing item in cache")
			continue
		}

		// only append if we successfully added to cache
		items = append(items, i)
	}

	// send to filters
	return items, nil
}
