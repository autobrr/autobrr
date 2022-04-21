package feed

import (
	"fmt"
	"sort"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type torznabJob struct {
	name              string
	indexerIdentifier string
	log               zerolog.Logger
	url               string
	client            *torznab.Client
	repo              domain.FeedCacheRepo
	releaseSvc        release.Service

	attempts int
	errors   []error

	cron  *cron.Cron
	jobID cron.EntryID
}

func (j *torznabJob) Run() {
	err := j.process()
	if err != nil {
		j.log.Err(err).Int("attempts", j.attempts).Msg("torznab process error")

		j.errors = append(j.errors, err)
	}

	j.attempts = 0
	j.errors = j.errors[:0]
}

func (j *torznabJob) process() error {
	// get feed
	items, err := j.getFeed()
	if err != nil {
		j.log.Error().Err(err).Msgf("torznab.process: error fetching feed items")
		return fmt.Errorf("torznab.process: error getting feed items: %w", err)
	}

	if len(items) == 0 {
		return nil
	}

	j.log.Debug().Msgf("torznab.process: refreshing feed: %v, found (%d) new items to check", j.name, len(items))

	releases := make([]*domain.Release, 0)

	for _, item := range items {
		rls, err := domain.NewRelease(item.Title, "")
		if err != nil {
			continue
		}

		rls.TorrentName = item.Title
		rls.TorrentURL = item.GUID
		rls.Implementation = domain.ReleaseImplementationTorznab
		rls.Indexer = j.indexerIdentifier
		//rls.Size = item.Size // TODO parse size

		if err := rls.Parse(); err != nil {
			j.log.Error().Err(err).Msgf("torznab.process: error parsing release")
			continue
		}

		releases = append(releases, rls)
	}

	// process all new releases
	go j.releaseSvc.ProcessMultiple(releases)

	return nil
}

func (j *torznabJob) getFeed() ([]torznab.FeedItem, error) {
	// get feed
	feedItems, err := j.client.GetFeed()
	if err != nil {
		j.log.Error().Err(err).Msgf("torznab.getFeed: error fetching feed items")
		return nil, err
	}

	j.log.Trace().Msgf("torznab getFeed: refreshing feed: %v, found (%d) items", j.name, len(feedItems))

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

		//if cacheValue, err := j.repo.Get(j.name, i.GUID); err == nil {
		//	j.log.Trace().Msgf("torznab getFeed: cacheValue: %v", cacheValue)
		//}

		if exists, err := j.repo.Exists(j.name, i.GUID); err == nil {
			if exists {
				j.log.Trace().Msg("torznab getFeed: cache item exists, skip")
				continue
			}
		}

		// do something more

		items = append(items, i)

		ttl := (24 * time.Hour) * 28

		if err := j.repo.Put(j.name, i.GUID, []byte("test"), ttl); err != nil {
			j.log.Error().Err(err).Str("guid", i.GUID).Msg("torznab getFeed: cache.Put: error storing item in cache")
		}
	}

	// send to filters
	return items, nil
}
