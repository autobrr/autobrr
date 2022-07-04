package feed

import (
	"sort"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
)

type TorznabJob struct {
	Name              string
	IndexerIdentifier string
	Log               zerolog.Logger
	URL               string
	Client            *torznab.Client
	Repo              domain.FeedCacheRepo
	ReleaseSvc        release.Service

	attempts int
	errors   []error

	JobID int
}

func NewTorznabJob(name string, indexerIdentifier string, log zerolog.Logger, url string, client *torznab.Client, repo domain.FeedCacheRepo, releaseSvc release.Service) *TorznabJob {
	return &TorznabJob{
		Name:              name,
		IndexerIdentifier: indexerIdentifier,
		Log:               log,
		URL:               url,
		Client:            client,
		Repo:              repo,
		ReleaseSvc:        releaseSvc,
	}
}

func (j *TorznabJob) Run() {
	err := j.process()
	if err != nil {
		j.Log.Err(err).Int("attempts", j.attempts).Msg("torznab process error")

		j.errors = append(j.errors, err)
	}

	j.attempts = 0
	j.errors = j.errors[:0]
}

func (j *TorznabJob) process() error {
	// get feed
	items, err := j.getFeed()
	if err != nil {
		j.Log.Error().Err(err).Msgf("torznab.process: error fetching feed items")
		return errors.Wrap(err, "torznab.process: error getting feed items: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	j.Log.Debug().Msgf("torznab.process: refreshing feed: %v, found (%d) new items to check", j.Name, len(items))

	releases := make([]*domain.Release, 0)

	for _, item := range items {
		rls := domain.NewRelease(j.IndexerIdentifier)

		rls.TorrentName = item.Title
		rls.TorrentURL = item.GUID
		rls.Implementation = domain.ReleaseImplementationTorznab
		rls.Indexer = j.IndexerIdentifier

		// parse size bytes string
		rls.ParseSizeBytesString(item.Size)

		rls.ParseString(item.Title)

		releases = append(releases, rls)
	}

	// process all new releases
	go j.ReleaseSvc.ProcessMultiple(releases)

	return nil
}

func (j *TorznabJob) getFeed() ([]torznab.FeedItem, error) {
	// get feed
	feedItems, err := j.Client.GetFeed()
	if err != nil {
		j.Log.Error().Err(err).Msgf("torznab.getFeed: error fetching feed items")
		return nil, errors.Wrap(err, "error fetching feed items")
	}

	j.Log.Trace().Msgf("torznab getFeed: refreshing feed: %v, found (%d) items", j.Name, len(feedItems))

	items := make([]torznab.FeedItem, 0)
	if len(feedItems) == 0 {
		return items, nil
	}

	sort.SliceStable(feedItems, func(i, j int) bool {
		return feedItems[i].PubDate.After(feedItems[j].PubDate.Time)
	})

	for _, i := range feedItems {
		if i.GUID == "" {
			continue
		}

		//if cacheValue, err := j.Repo.Get(j.Name, i.GUID); err == nil {
		//	j.Log.Trace().Msgf("torznab getFeed: cacheValue: %v", cacheValue)
		//}

		if exists, err := j.Repo.Exists(j.Name, i.GUID); err == nil {
			if exists {
				j.Log.Trace().Msg("torznab getFeed: cache item exists, skip")
				continue
			}
		}

		// do something more

		items = append(items, i)

		// set ttl to 1 month
		ttl := time.Now().AddDate(0, 1, 0)

		if err := j.Repo.Put(j.Name, i.GUID, []byte(i.Title), ttl); err != nil {
			j.Log.Error().Stack().Err(err).Str("guid", i.GUID).Msg("torznab getFeed: cache.Put: error storing item in cache")
		}
	}

	// send to filters
	return items, nil
}
