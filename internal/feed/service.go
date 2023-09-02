// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/newznab"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/mmcdole/gofeed"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Service interface {
	FindByID(ctx context.Context, id int) (*domain.Feed, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) (*domain.Feed, error)
	Find(ctx context.Context) ([]domain.Feed, error)
	GetCacheByID(ctx context.Context, feedId int) ([]domain.FeedCacheItem, error)
	Store(ctx context.Context, feed *domain.Feed) error
	Update(ctx context.Context, feed *domain.Feed) error
	Test(ctx context.Context, feed *domain.Feed) error
	ToggleEnabled(ctx context.Context, id int, enabled bool) error
	Delete(ctx context.Context, id int) error
	DeleteFeedCache(ctx context.Context, id int) error
	GetLastRunData(ctx context.Context, id int) (string, error)
	DeleteFeedCacheStale(ctx context.Context) error

	Start() error
}

type feedInstance struct {
	Feed              *domain.Feed
	Name              string
	IndexerIdentifier string
	URL               string
	ApiKey            string
	Implementation    string
	CronSchedule      time.Duration
	Timeout           time.Duration
}

// feedKey creates a unique identifier to be used for controlling jobs in the scheduler
type feedKey struct {
	id int
}

// ToString creates a string of the unique id to be used for controlling jobs in the scheduler
func (k feedKey) ToString() string {
	return fmt.Sprintf("feed-%d", k.id)
}

type service struct {
	log  zerolog.Logger
	jobs map[string]int

	repo       domain.FeedRepo
	cacheRepo  domain.FeedCacheRepo
	releaseSvc release.Service
	scheduler  scheduler.Service
}

func NewService(log logger.Logger, repo domain.FeedRepo, cacheRepo domain.FeedCacheRepo, releaseSvc release.Service, scheduler scheduler.Service) Service {
	return &service{
		log:        log.With().Str("module", "feed").Logger(),
		jobs:       map[string]int{},
		repo:       repo,
		cacheRepo:  cacheRepo,
		releaseSvc: releaseSvc,
		scheduler:  scheduler,
	}
}

func (s *service) FindByID(ctx context.Context, id int) (*domain.Feed, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) FindByIndexerIdentifier(ctx context.Context, indexer string) (*domain.Feed, error) {
	return s.repo.FindByIndexerIdentifier(ctx, indexer)
}

func (s *service) Find(ctx context.Context) ([]domain.Feed, error) {
	feeds, err := s.repo.Find(ctx)
	if err != nil {
		return nil, err
	}

	for i, feed := range feeds {
		t, err := s.scheduler.GetNextRun(feedKey{id: feed.ID}.ToString())
		if err != nil {
			continue
		}
		feed.NextRun = t
		feeds[i] = feed
	}

	return feeds, nil
}

func (s *service) GetCacheByID(ctx context.Context, feedId int) ([]domain.FeedCacheItem, error) {
	return s.cacheRepo.GetByFeed(ctx, feedId)
}

func (s *service) Store(ctx context.Context, feed *domain.Feed) error {
	return s.repo.Store(ctx, feed)
}

func (s *service) Update(ctx context.Context, feed *domain.Feed) error {
	return s.update(ctx, feed)
}

func (s *service) Delete(ctx context.Context, id int) error {
	return s.delete(ctx, id)
}

func (s *service) DeleteFeedCache(ctx context.Context, id int) error {
	return s.cacheRepo.DeleteByFeed(ctx, id)
}

func (s *service) DeleteFeedCacheStale(ctx context.Context) error {
	return s.cacheRepo.DeleteStale(ctx)
}

func (s *service) ToggleEnabled(ctx context.Context, id int, enabled bool) error {
	return s.toggleEnabled(ctx, id, enabled)
}

func (s *service) Test(ctx context.Context, feed *domain.Feed) error {
	return s.test(ctx, feed)
}

func (s *service) Start() error {
	return s.start()
}

func (s *service) update(ctx context.Context, feed *domain.Feed) error {
	if err := s.repo.Update(ctx, feed); err != nil {
		s.log.Error().Err(err).Msg("error updating feed")
		return err
	}

	if err := s.restartJob(feed); err != nil {
		s.log.Error().Err(err).Msg("error restarting feed")
		return err
	}

	return nil
}

func (s *service) delete(ctx context.Context, id int) error {
	f, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msg("error finding feed")
		return err
	}

	s.log.Debug().Msgf("stopping and removing feed: %s", f.Name)

	if err := s.stopFeedJob(f.ID); err != nil {
		s.log.Error().Err(err).Msgf("error stopping rss job: %s id: %d", f.Name, f.ID)
		return err
	}

	// delete feed and cascade delete feed_cache by fk
	if err := s.repo.Delete(ctx, f.ID); err != nil {
		s.log.Error().Err(err).Msgf("error deleting feed: %s", f.Name)
		return err
	}

	return nil
}

func (s *service) toggleEnabled(ctx context.Context, id int, enabled bool) error {
	f, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msg("error finding feed")
		return err
	}

	if err := s.repo.ToggleEnabled(ctx, id, enabled); err != nil {
		s.log.Error().Err(err).Msg("error feed toggle enabled")
		return err
	}

	if f.Enabled != enabled {
		if enabled {
			// override enabled
			f.Enabled = true

			if err := s.startJob(f); err != nil {
				s.log.Error().Err(err).Msg("error starting feed job")
				return err
			}

			s.log.Debug().Msgf("feed started: %s", f.Name)

			return nil
		}

		s.log.Debug().Msgf("stopping feed: %s", f.Name)

		if err := s.stopFeedJob(f.ID); err != nil {
			s.log.Error().Err(err).Msg("error stopping feed job")
			return err
		}

		s.log.Debug().Msgf("feed stopped: %s", f.Name)

		return nil
	}

	return nil
}

func (s *service) test(ctx context.Context, feed *domain.Feed) error {
	// create sub logger
	subLogger := zstdlog.NewStdLoggerWithLevel(s.log.With().Logger(), zerolog.DebugLevel)

	// test feeds
	switch feed.Type {
	case string(domain.FeedTypeTorznab):
		if err := s.testTorznab(ctx, feed, subLogger); err != nil {
			return err
		}

	case string(domain.FeedTypeNewznab):
		if err := s.testNewznab(ctx, feed, subLogger); err != nil {
			return err
		}

	case string(domain.FeedTypeRSS):
		if err := s.testRSS(ctx, feed); err != nil {
			return err
		}

	default:
		return errors.New("unsupported feed type: %s", feed.Type)
	}

	s.log.Info().Msgf("feed test successful - connected to feed: %s", feed.URL)

	return nil
}

func (s *service) testRSS(ctx context.Context, feed *domain.Feed) error {
	f, err := gofeed.NewParser().ParseURLWithContext(feed.URL, ctx)
	if err != nil {
		s.log.Error().Err(err).Msgf("error fetching rss feed items")
		return errors.Wrap(err, "error fetching rss feed items")
	}

	s.log.Info().Msgf("refreshing rss feed: %s, found (%d) items", feed.Name, len(f.Items))

	return nil
}

func (s *service) testTorznab(ctx context.Context, feed *domain.Feed, subLogger *log.Logger) error {
	// setup torznab Client
	c := torznab.NewClient(torznab.Config{Host: feed.URL, ApiKey: feed.ApiKey, Log: subLogger})

	items, err := c.FetchFeed(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("error getting torznab feed")
		return err
	}

	s.log.Info().Msgf("refreshing torznab feed: %s, found (%d) items", feed.Name, len(items.Channel.Items))

	return nil
}

func (s *service) testNewznab(ctx context.Context, feed *domain.Feed, subLogger *log.Logger) error {
	// setup newznab Client
	c := newznab.NewClient(newznab.Config{Host: feed.URL, ApiKey: feed.ApiKey, Log: subLogger})

	items, err := c.GetFeed(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("error getting newznab feed")
		return err
	}

	s.log.Info().Msgf("refreshing newznab feed: %s, found (%d) items", feed.Name, len(items.Channel.Items))

	return nil
}

func (s *service) start() error {
	// always run feed cache maintenance job
	if err := s.createCleanupJob(); err != nil {
		s.log.Error().Err(err).Msg("could not start feed cache cleanup job")
	}

	// get all feeds
	feeds, err := s.repo.Find(context.TODO())
	if err != nil {
		s.log.Error().Err(err).Msg("error finding feeds")
		return err
	}

	for _, feed := range feeds {
		feed := feed
		if err := s.startJob(&feed); err != nil {
			s.log.Error().Err(err).Msgf("failed to initialize feed job: %s", feed.Name)
			continue
		}
	}

	return nil
}

func (s *service) restartJob(f *domain.Feed) error {
	s.log.Debug().Msgf("stopping feed: %s", f.Name)

	// stop feed job
	if err := s.stopFeedJob(f.ID); err != nil {
		s.log.Error().Err(err).Msg("error stopping feed job")
		return err
	}

	if f.Enabled {
		if err := s.startJob(f); err != nil {
			s.log.Error().Err(err).Msg("error starting feed job")
			return err
		}

		s.log.Debug().Msgf("restarted feed: %s", f.Name)
	}

	return nil
}

func (s *service) startJob(f *domain.Feed) error {
	// if it's not enabled we should not start it
	if !f.Enabled {
		return nil
	}

	// get torznab_url from settings
	if f.URL == "" {
		return errors.New("no URL provided for feed: %s", f.Name)
	}

	// cron schedule to run every X minutes
	fi := feedInstance{
		Feed:              f,
		Name:              f.Name,
		IndexerIdentifier: f.Indexer,
		Implementation:    f.Type,
		URL:               f.URL,
		ApiKey:            f.ApiKey,
		CronSchedule:      time.Duration(f.Interval) * time.Minute,
		Timeout:           time.Duration(f.Timeout) * time.Second,
	}

	var err error
	var job cron.Job

	switch fi.Implementation {
	case string(domain.FeedTypeTorznab):
		job, err = s.createTorznabJob(fi)

	case string(domain.FeedTypeNewznab):
		job, err = s.createNewznabJob(fi)

	case string(domain.FeedTypeRSS):
		job, err = s.createRSSJob(fi)

	default:
		return errors.New("unsupported feed type: %s", fi.Implementation)
	}

	if err != nil {
		s.log.Error().Err(err).Msgf("failed to initialize %s feed", fi.Implementation)
		return err
	}

	identifierKey := feedKey{f.ID}.ToString()

	// schedule job
	id, err := s.scheduler.ScheduleJob(job, fi.CronSchedule, identifierKey)
	if err != nil {
		return errors.Wrap(err, "add job %s failed", identifierKey)
	}

	// add to job map
	s.jobs[identifierKey] = id

	s.log.Debug().Msgf("successfully started feed: %s", f.Name)

	return nil
}

func (s *service) createTorznabJob(f feedInstance) (cron.Job, error) {
	s.log.Debug().Msgf("create torznab job: %s", f.Name)

	if f.URL == "" {
		return nil, errors.New("torznab feed requires URL")
	}

	//if f.CronSchedule < 5*time.Minute {
	//	f.CronSchedule = 15 * time.Minute
	//}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Logger()

	// setup torznab Client
	client := torznab.NewClient(torznab.Config{Host: f.URL, ApiKey: f.ApiKey, Timeout: f.Timeout})

	// create job
	job := NewTorznabJob(f.Feed, f.Name, f.IndexerIdentifier, l, f.URL, client, s.repo, s.cacheRepo, s.releaseSvc)

	return job, nil
}

func (s *service) createNewznabJob(f feedInstance) (cron.Job, error) {
	s.log.Debug().Msgf("add newznab job: %s", f.Name)

	if f.URL == "" {
		return nil, errors.New("newznab feed requires URL")
	}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Logger()

	// setup newznab Client
	client := newznab.NewClient(newznab.Config{Host: f.URL, ApiKey: f.ApiKey, Timeout: f.Timeout})

	// create job
	job := NewNewznabJob(f.Feed, f.Name, f.IndexerIdentifier, l, f.URL, client, s.repo, s.cacheRepo, s.releaseSvc)

	return job, nil
}

func (s *service) createRSSJob(f feedInstance) (cron.Job, error) {
	s.log.Debug().Msgf("add rss job: %s", f.Name)

	if f.URL == "" {
		return nil, errors.New("rss feed requires URL")
	}

	//if f.CronSchedule < time.Duration(5*time.Minute) {
	//	f.CronSchedule = time.Duration(15 * time.Minute)
	//}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Logger()

	// create job
	job := NewRSSJob(f.Feed, f.Name, f.IndexerIdentifier, l, f.URL, s.repo, s.cacheRepo, s.releaseSvc, f.Timeout)

	return job, nil
}

func (s *service) createCleanupJob() error {
	// setup logger
	l := s.log.With().Str("job", "feed-cache-cleanup").Logger()

	// create job
	job := NewCleanupJob(l, s.cacheRepo)

	identifierKey := "feed-cache-cleanup"

	// schedule job for every day at 03:05
	id, err := s.scheduler.AddJob(job, "5 3 * * *", identifierKey)
	if err != nil {
		return errors.Wrap(err, "add job %s failed", identifierKey)
	}

	// add to job map
	s.jobs[identifierKey] = id

	return nil
}

func (s *service) stopFeedJob(id int) error {
	// remove job from scheduler
	if err := s.scheduler.RemoveJobByIdentifier(feedKey{id}.ToString()); err != nil {
		return errors.Wrap(err, "stop job failed")
	}

	s.log.Debug().Msgf("stop feed job: %d", id)

	return nil
}

func (s *service) GetNextRun(id int) (time.Time, error) {
	return s.scheduler.GetNextRun(feedKey{id}.ToString())
}

func (s *service) GetLastRunData(ctx context.Context, id int) (string, error) {
	feed, err := s.repo.GetLastRunDataByID(ctx, id)
	if err != nil {
		return "", err
	}

	return feed, nil
}
