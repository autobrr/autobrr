// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"fmt"
	"log"
	"strconv"
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
	"github.com/rs/zerolog"
)

type Service interface {
	FindByID(ctx context.Context, id int) (*domain.Feed, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) (*domain.Feed, error)
	Find(ctx context.Context) ([]domain.Feed, error)
	GetCacheByID(ctx context.Context, bucket string) ([]domain.FeedCacheItem, error)
	Store(ctx context.Context, feed *domain.Feed) error
	Update(ctx context.Context, feed *domain.Feed) error
	Test(ctx context.Context, feed *domain.Feed) error
	ToggleEnabled(ctx context.Context, id int, enabled bool) error
	Delete(ctx context.Context, id int) error
	GetLastRunData(ctx context.Context, id int) (string, error)

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

type feedKey struct {
	id      int
	indexer string
	name    string
}

func (k feedKey) ToString() string {
	return fmt.Sprintf("%v+%v+%v", k.id, k.indexer, k.name)
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
	return s.repo.Find(ctx)
}

func (s *service) GetCacheByID(ctx context.Context, bucket string) ([]domain.FeedCacheItem, error) {
	id, _ := strconv.Atoi(bucket)

	feed, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find feed by id: %v", id)
		return nil, err
	}

	data, err := s.cacheRepo.GetByBucket(ctx, feed.Name)
	if err != nil {
		s.log.Error().Err(err).Msg("could not get feed cache")
		return nil, err
	}

	return data, err
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

	s.log.Debug().Msgf("stopping and removing feed: %v", f.Name)

	identifierKey := feedKey{f.ID, f.Indexer, f.Name}.ToString()

	if err := s.stopFeedJob(identifierKey); err != nil {
		s.log.Error().Err(err).Msg("error stopping rss job")
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		s.log.Error().Err(err).Msg("error deleting feed")
		return err
	}

	if err := s.cacheRepo.DeleteBucket(ctx, f.Name); err != nil {
		s.log.Error().Err(err).Msgf("could not delete feedCache bucket by id: %v", id)
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

			s.log.Debug().Msgf("feed started: %v", f.Name)

			return nil
		} else {
			s.log.Debug().Msgf("stopping feed: %v", f.Name)

			identifierKey := feedKey{f.ID, f.Indexer, f.Name}.ToString()

			if err := s.stopFeedJob(identifierKey); err != nil {
				s.log.Error().Err(err).Msg("error stopping feed job")
				return err
			}

			s.log.Debug().Msgf("feed stopped: %v", f.Name)

			return nil
		}
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

	s.log.Info().Msgf("refreshing rss feed: %v, found (%d) items", feed.Name, len(f.Items))

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

	s.log.Info().Msgf("refreshing torznab feed: %v, found (%d) items", feed.Name, len(items.Channel.Items))

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

	s.log.Info().Msgf("refreshing newznab feed: %v, found (%d) items", feed.Name, len(items.Channel.Items))

	return nil
}

func (s *service) start() error {
	// get all torznab indexer definitions
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
	s.log.Debug().Msgf("stopping feed: %v", f.Name)

	identifierKey := feedKey{f.ID, f.Indexer, f.Name}.ToString()

	// stop feed job
	if err := s.stopFeedJob(identifierKey); err != nil {
		s.log.Error().Err(err).Msg("error stopping feed job")
		return err
	}

	if f.Enabled {
		if err := s.startJob(f); err != nil {
			s.log.Error().Err(err).Msg("error starting feed job")
			return err
		}

		s.log.Debug().Msgf("restarted feed: %v", f.Name)
	}

	return nil
}

func (s *service) startJob(f *domain.Feed) error {
	// get all torznab indexer definitions
	if !f.Enabled {
		return nil
	}

	// get torznab_url from settings
	if f.URL == "" {
		return errors.New("no URL provided for feed: %v", f.Name)
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

	switch fi.Implementation {
	case string(domain.FeedTypeTorznab):
		if err := s.addTorznabJob(fi); err != nil {
			s.log.Error().Err(err).Msg("failed to initialize torznab feed")
			return err
		}

	case string(domain.FeedTypeNewznab):
		if err := s.addNewznabJob(fi); err != nil {
			s.log.Error().Err(err).Msg("failed to initialize newznab feed")
			return err
		}

	case string(domain.FeedTypeRSS):
		if err := s.addRSSJob(fi); err != nil {
			s.log.Error().Err(err).Msg("failed to initialize rss feed")
			return err
		}
	}

	return nil
}

func (s *service) addTorznabJob(f feedInstance) error {
	if f.URL == "" {
		return errors.New("torznab feed requires URL")
	}

	//if f.CronSchedule < 5*time.Minute {
	//	f.CronSchedule = 15 * time.Minute
	//}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Logger()

	// setup torznab Client
	c := torznab.NewClient(torznab.Config{Host: f.URL, ApiKey: f.ApiKey, Timeout: f.Timeout})

	// create job
	job := NewTorznabJob(f.Feed, f.Name, f.IndexerIdentifier, l, f.URL, c, s.repo, s.cacheRepo, s.releaseSvc)

	identifierKey := feedKey{f.Feed.ID, f.Feed.Indexer, f.Feed.Name}.ToString()

	// schedule job
	id, err := s.scheduler.AddJob(job, f.CronSchedule, identifierKey)
	if err != nil {
		return errors.Wrap(err, "feed.AddTorznabJob: add job failed")
	}
	job.JobID = id

	// add to job map
	s.jobs[identifierKey] = id

	s.log.Debug().Msgf("add torznab job: %v", f.Name)

	return nil
}

func (s *service) addNewznabJob(f feedInstance) error {
	if f.URL == "" {
		return errors.New("newznab feed requires URL")
	}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Logger()

	// setup newznab Client
	c := newznab.NewClient(newznab.Config{Host: f.URL, ApiKey: f.ApiKey, Timeout: f.Timeout})

	// create job
	job := NewNewznabJob(f.Feed, f.Name, f.IndexerIdentifier, l, f.URL, c, s.repo, s.cacheRepo, s.releaseSvc)

	identifierKey := feedKey{f.Feed.ID, f.Feed.Indexer, f.Feed.Name}.ToString()

	// schedule job
	id, err := s.scheduler.AddJob(job, f.CronSchedule, identifierKey)
	if err != nil {
		return errors.Wrap(err, "feed.AddNewznabJob: add job failed")
	}
	job.JobID = id

	// add to job map
	s.jobs[identifierKey] = id

	s.log.Debug().Msgf("add newznab job: %v", f.Name)

	return nil
}

func (s *service) addRSSJob(f feedInstance) error {
	if f.URL == "" {
		return errors.New("rss feed requires URL")
	}

	//if f.CronSchedule < time.Duration(5*time.Minute) {
	//	f.CronSchedule = time.Duration(15 * time.Minute)
	//}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Logger()

	// create job
	job := NewRSSJob(f.Feed, f.Name, f.IndexerIdentifier, l, f.URL, s.repo, s.cacheRepo, s.releaseSvc, f.Timeout)

	identifierKey := feedKey{f.Feed.ID, f.Feed.Indexer, f.Feed.Name}.ToString()

	// schedule job
	id, err := s.scheduler.AddJob(job, f.CronSchedule, identifierKey)
	if err != nil {
		return errors.Wrap(err, "feed.AddRSSJob: add job failed")
	}
	job.JobID = id

	// add to job map
	s.jobs[identifierKey] = id

	s.log.Debug().Msgf("add rss job: %v", f.Name)

	return nil
}

func (s *service) stopFeedJob(indexer string) error {
	// remove job from scheduler
	if err := s.scheduler.RemoveJobByIdentifier(indexer); err != nil {
		return errors.Wrap(err, "stop job failed")
	}

	s.log.Debug().Msgf("stop feed job: %v", indexer)

	return nil
}

func (s *service) GetNextRun(indexer string) (time.Time, error) {
	return s.scheduler.GetNextRun(indexer)
}

func (s *service) GetLastRunData(ctx context.Context, id int) (string, error) {
	feed, err := s.repo.GetLastRunDataByID(ctx, id)
	if err != nil {
		return "", err
	}

	return feed, nil
}
