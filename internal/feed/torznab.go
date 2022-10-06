package feed

import (
	"sort"
	"strconv"
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
	Client            torznab.Client
	Repo              domain.FeedCacheRepo
	ReleaseSvc        release.Service

	attempts int
	errors   []error

	JobID int
}

func NewTorznabJob(name string, indexerIdentifier string, log zerolog.Logger, url string, client torznab.Client, repo domain.FeedCacheRepo, releaseSvc release.Service) *TorznabJob {
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

		if parseFreeleech(item) {
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

func parseFreeleech(item torznab.FeedItem) bool {
	for _, attr := range item.Attributes {
		if attr.Name == "downloadvolumefactor" {
			if attr.Value == "0" {
				return true
			}
		}
	}

	return false
}

func (j *TorznabJob) getFeed() ([]torznab.FeedItem, error) {
	// get feed
	feedItems, err := j.Client.FetchFeed()
	if err != nil {
		j.Log.Error().Err(err).Msgf("error fetching feed items")
		return nil, errors.Wrap(err, "error fetching feed items")
	}

	j.Log.Debug().Msgf("refreshing feed: %v, found (%d) items", j.Name, len(feedItems))

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

		exists, err := j.Repo.Exists(j.Name, i.GUID)
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

		if err := j.Repo.Put(j.Name, i.GUID, []byte(i.Title), ttl); err != nil {
			j.Log.Error().Stack().Err(err).Str("guid", i.GUID).Msg("cache.Put: error storing item in cache")
			continue
		}

		// only append if we successfully added to cache
		items = append(items, i)
	}

	// send to filters
	return items, nil
}
