// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/newznab"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Service interface {
	FindOne(ctx context.Context, params domain.FindOneParams) (*domain.Feed, error)
	FindByID(ctx context.Context, id int) (*domain.Feed, error)
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
	ForceRun(ctx context.Context, id int) error
	FetchCaps(ctx context.Context, feed *domain.Feed) (*domain.FeedCapabilities, error)
	FetchCapsByID(ctx context.Context, id int) (*domain.FeedCapabilities, error)

	Start() error
}

type feedInstance struct {
	Feed           *domain.Feed
	Name           string
	Indexer        domain.IndexerMinimal
	URL            string
	ApiKey         string
	Implementation string
	CronSchedule   time.Duration
	Timeout        time.Duration
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
	proxySvc   proxy.Service
	scheduler  scheduler.Service
}

func NewService(log logger.Logger, repo domain.FeedRepo, cacheRepo domain.FeedCacheRepo, releaseSvc release.Service, proxySvc proxy.Service, scheduler scheduler.Service) Service {
	return &service{
		log:        log.With().Str("module", "feed").Logger(),
		jobs:       map[string]int{},
		repo:       repo,
		cacheRepo:  cacheRepo,
		releaseSvc: releaseSvc,
		proxySvc:   proxySvc,
		scheduler:  scheduler,
	}
}

func (s *service) FindOne(ctx context.Context, params domain.FindOneParams) (*domain.Feed, error) {
	return s.repo.FindOne(ctx, params)
}

func (s *service) FindByID(ctx context.Context, id int) (*domain.Feed, error) {
	return s.repo.FindByID(ctx, id)
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
	existingFeed, err := s.repo.FindOne(ctx, domain.FindOneParams{FeedID: feed.ID})
	if err != nil {
		s.log.Error().Err(err).Msg("could not find feed")
		return err
	}

	if domain.IsRedactedString(feed.ApiKey) {
		feed.ApiKey = existingFeed.ApiKey
	}
	if domain.IsRedactedString(feed.Cookie) {
		feed.Cookie = existingFeed.Cookie
	}

	if err := s.repo.Update(ctx, feed); err != nil {
		s.log.Error().Err(err).Msg("error updating feed")
		return err
	}

	// get Feed again for ProxyID and UseProxy to be correctly populated
	feed, err = s.repo.FindOne(ctx, domain.FindOneParams{FeedID: feed.ID})
	if err != nil {
		s.log.Error().Err(err).Msg("error finding feed")
		return err
	}

	if err := s.restartJob(feed); err != nil {
		s.log.Error().Err(err).Msg("error restarting feed")
		return err
	}

	return nil
}

func (s *service) delete(ctx context.Context, id int) error {
	f, err := s.repo.FindOne(ctx, domain.FindOneParams{FeedID: id})
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

	// if foreign keys are not enforced in SQLite clear feed cache explicitly
	if err := s.cacheRepo.DeleteByFeed(ctx, id); err != nil {
		s.log.Error().Err(err).Msgf("error deleting feed cache: %s", f.Name)
	}

	return nil
}

func (s *service) toggleEnabled(ctx context.Context, id int, enabled bool) error {
	f, err := s.repo.FindOne(ctx, domain.FindOneParams{FeedID: id})
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

	// add proxy conf
	if feed.UseProxy {
		proxyConf, err := s.proxySvc.FindByID(ctx, feed.ProxyID)
		if err != nil {
			return errors.Wrap(err, "could not find proxy for indexer feed")
		}

		if proxyConf.Enabled {
			feed.Proxy = proxyConf
		}
	}

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
	feedParser := NewFeedParser(time.Duration(feed.Timeout)*time.Second, feed.Cookie, feed.TLSSkipVerify)

	// add proxy if enabled and exists
	if feed.UseProxy && feed.Proxy != nil {
		proxyClient, err := proxy.GetProxiedHTTPClient(feed.Proxy)
		if err != nil {
			return errors.Wrap(err, "could not get proxy client")
		}

		if feed.TLSSkipVerify {
			if t, ok := proxyClient.Transport.(*http.Transport); ok {
				t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}
		}

		feedParser.WithHTTPClient(proxyClient)

		s.log.Debug().Msgf("using proxy %s for feed %s", feed.Proxy.Name, feed.Name)
	}

	feedResponse, err := feedParser.ParseURLWithContext(ctx, feed.URL)
	if err != nil {
		s.log.Error().Err(err).Msgf("error fetching rss feed items")
		return errors.Wrap(err, "error fetching rss feed items")
	}

	s.log.Info().Msgf("refreshing rss feed: %s, found (%d) items", feed.Name, len(feedResponse.Items))

	return nil
}

func (s *service) testTorznab(ctx context.Context, feed *domain.Feed, subLogger *log.Logger) error {
	// setup torznab Client
	c := torznab.NewClient(torznab.Config{Host: feed.URL, ApiKey: feed.ApiKey, TLSSkipVerify: feed.TLSSkipVerify, Log: subLogger})

	// add proxy if enabled and exists
	if feed.UseProxy && feed.Proxy != nil {
		proxyClient, err := proxy.GetProxiedHTTPClient(feed.Proxy)
		if err != nil {
			return errors.Wrap(err, "could not get proxy client")
		}

		if feed.TLSSkipVerify {
			if t, ok := proxyClient.Transport.(*http.Transport); ok {
				t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}
		}

		c.WithHTTPClient(proxyClient)

		s.log.Debug().Msgf("using proxy %s for feed %s", feed.Proxy.Name, feed.Name)
	}

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
	c := newznab.NewClient(newznab.Config{Host: feed.URL, ApiKey: feed.ApiKey, TLSSkipVerify: feed.TLSSkipVerify, Log: subLogger})

	// add proxy if enabled and exists
	if feed.UseProxy && feed.Proxy != nil {
		proxyClient, err := proxy.GetProxiedHTTPClient(feed.Proxy)
		if err != nil {
			return errors.Wrap(err, "could not get proxy client")
		}

		if feed.TLSSkipVerify {
			if t, ok := proxyClient.Transport.(*http.Transport); ok {
				t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
			}
		}

		c.WithHTTPClient(proxyClient)

		s.log.Debug().Msgf("using proxy %s for feed %s", feed.Proxy.Name, feed.Name)
	}

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

	if len(feeds) == 0 {
		s.log.Debug().Msg("found 0 feeds to start")
		return nil
	}

	s.log.Debug().Msgf("preparing staggered start of %d feeds", len(feeds))

	// start in background to not block startup and signal.Notify signals until all feeds are started
	go func(feeds []domain.Feed) {
		for _, feed := range feeds {
			if !feed.Enabled {
				s.log.Trace().Msgf("feed disabled, skipping... %s", feed.Name)
				continue
			}

			if err := s.startJob(&feed); err != nil {
				s.log.Error().Err(err).Msgf("failed to initialize feed job: %s", feed.Name)
				continue
			}

			// add sleep for the next iteration to start staggered which should mitigate sqlite BUSY errors
			time.Sleep(time.Second * 5)
		}
	}(feeds)

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
func newFeedInstance(f *domain.Feed) feedInstance {
	// cron schedule to run every X minutes
	fi := feedInstance{
		Feed:           f,
		Name:           f.Name,
		Indexer:        f.Indexer,
		Implementation: f.Type,
		URL:            f.URL,
		ApiKey:         f.ApiKey,
		CronSchedule:   time.Duration(f.Interval) * time.Minute,
		Timeout:        time.Duration(f.Timeout) * time.Second,
	}

	return fi
}

func (s *service) initializeFeedJob(fi feedInstance) (RefreshFeedJob, error) {
	var err error
	var job RefreshFeedJob

	switch fi.Implementation {
	case string(domain.FeedTypeTorznab):
		job, err = s.createTorznabJob(fi)

	case string(domain.FeedTypeNewznab):
		job, err = s.createNewznabJob(fi)

	case string(domain.FeedTypeRSS):
		job, err = s.createRSSJob(fi)

	default:
		return nil, errors.New("unsupported feed type: %s", fi.Implementation)
	}

	if err != nil {
		s.log.Error().Err(err).Msgf("failed to initialize %s feed", fi.Implementation)
		return nil, err
	}

	return job, nil
}

func (s *service) startJob(f *domain.Feed) error {
	// if it's not enabled we should not start it
	if !f.Enabled {
		return errors.New("feed %s not enabled", f.Name)
	}

	// get url from settings
	if f.URL == "" {
		return errors.New("no URL provided for feed: %s", f.Name)
	}

	// add proxy conf
	if f.UseProxy {
		proxyConf, err := s.proxySvc.FindByID(context.Background(), f.ProxyID)
		if err != nil {
			return errors.Wrap(err, "could not find proxy for indexer feed")
		}

		if proxyConf.Enabled {
			f.Proxy = proxyConf
		}
	}

	fi := newFeedInstance(f)

	job, err := s.initializeFeedJob(fi)
	if err != nil {
		return errors.Wrap(err, "initialize job %s failed", f.Name)
	}

	if err := s.scheduleJob(fi, job); err != nil {
		return errors.Wrap(err, "schedule job %s failed", f.Name)
	}

	s.log.Debug().Msgf("successfully started feed: %s", f.Name)

	return nil
}

func (s *service) scheduleJob(fi feedInstance, job cron.Job) error {
	identifierKey := feedKey{fi.Feed.ID}.ToString()

	// schedule job
	id, err := s.scheduler.ScheduleJob(job, fi.CronSchedule, identifierKey)
	if err != nil {
		return errors.Wrap(err, "add job %s failed", identifierKey)
	}

	// add to job map
	s.jobs[identifierKey] = id

	return nil
}

func (s *service) createTorznabJob(f feedInstance) (RefreshFeedJob, error) {
	s.log.Debug().Msgf("create torznab job: %s", f.Name)

	if f.URL == "" {
		return nil, errors.New("torznab feed requires URL")
	}

	//if f.CronSchedule < 5*time.Minute {
	//	f.CronSchedule = 15 * time.Minute
	//}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Str("implementation", f.Implementation).Logger()

	// setup torznab Client
	client := torznab.NewClient(torznab.Config{Host: f.URL, ApiKey: f.ApiKey, Timeout: f.Timeout, TLSSkipVerify: f.Feed.TLSSkipVerify})

	// create job
	job := NewTorznabJob(f.Feed, f.Name, l, f.URL, client, s.repo, s.cacheRepo, s.releaseSvc)

	return job, nil
}

func (s *service) createNewznabJob(f feedInstance) (RefreshFeedJob, error) {
	s.log.Debug().Msgf("create newznab job: %s", f.Name)

	if f.URL == "" {
		return nil, errors.New("newznab feed requires URL")
	}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Str("implementation", f.Implementation).Logger()

	// setup newznab Client
	client := newznab.NewClient(newznab.Config{Host: f.URL, ApiKey: f.ApiKey, Timeout: f.Timeout, TLSSkipVerify: f.Feed.TLSSkipVerify})

	// create job
	job := NewNewznabJob(f.Feed, f.Name, l, f.URL, client, s.repo, s.cacheRepo, s.releaseSvc)

	return job, nil
}

func (s *service) createRSSJob(f feedInstance) (RefreshFeedJob, error) {
	s.log.Debug().Msgf("create rss job: %s", f.Name)

	if f.URL == "" {
		return nil, errors.New("rss feed requires URL")
	}

	//if f.CronSchedule < time.Duration(5*time.Minute) {
	//	f.CronSchedule = time.Duration(15 * time.Minute)
	//}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Str("implementation", f.Implementation).Logger()

	// create job
	job := NewRSSJob(f.Feed, f.Name, l, f.URL, s.repo, s.cacheRepo, s.releaseSvc, f.Timeout)

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

func (s *service) ForceRun(ctx context.Context, id int) error {
	feed, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}

	fi := newFeedInstance(feed)

	job, err := s.initializeFeedJob(fi)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to initialize feed job")
		return err
	}

	if err := job.RunE(ctx); err != nil {
		s.log.Error().Err(err).Msg("failed to refresh feed")
		return err
	}

	return nil
}

func (s *service) FetchCaps(ctx context.Context, feed *domain.Feed) (*domain.FeedCapabilities, error) {
	if feed == nil {
		return nil, errors.New("feed is required")
	}

	if feed.URL == "" {
		return nil, errors.New("feed URL is required")
	}

	if feed.Timeout == 0 {
		feed.Timeout = 60
	}

	if feed.UseProxy {
		proxyConf, err := s.proxySvc.FindByID(ctx, feed.ProxyID)
		if err != nil {
			return nil, errors.Wrap(err, "could not find proxy for indexer feed")
		}

		if proxyConf.Enabled {
			feed.Proxy = proxyConf
		}
	}

	switch feed.Type {
	case string(domain.FeedTypeTorznab):
		client := torznab.NewClient(torznab.Config{Host: feed.URL, ApiKey: feed.ApiKey, Timeout: time.Duration(feed.Timeout) * time.Second, TLSSkipVerify: feed.TLSSkipVerify})

		if feed.UseProxy && feed.Proxy != nil {
			proxyClient, err := proxy.GetProxiedHTTPClient(feed.Proxy)
			if err != nil {
				return nil, errors.Wrap(err, "could not get proxy client")
			}

			if feed.TLSSkipVerify {
				if t, ok := proxyClient.Transport.(*http.Transport); ok {
					t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				}
			}

			client.WithHTTPClient(proxyClient)
		}

		caps, err := client.FetchCaps(ctx)
		if err != nil {
			return nil, err
		}

		unifiedCaps := domain.NewFeedCapabilitiesFromTorznab(caps)

		return unifiedCaps, nil

	case string(domain.FeedTypeNewznab):
		client := newznab.NewClient(newznab.Config{Host: feed.URL, ApiKey: feed.ApiKey, Timeout: time.Duration(feed.Timeout) * time.Second, TLSSkipVerify: feed.TLSSkipVerify})

		if feed.UseProxy && feed.Proxy != nil {
			proxyClient, err := proxy.GetProxiedHTTPClient(feed.Proxy)
			if err != nil {
				return nil, errors.Wrap(err, "could not get proxy client")
			}

			if feed.TLSSkipVerify {
				if t, ok := proxyClient.Transport.(*http.Transport); ok {
					t.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				}
			}

			client.WithHTTPClient(proxyClient)
		}

		caps, err := client.GetCaps(ctx)
		if err != nil {
			return nil, err
		}

		unifiedCaps := domain.NewFeedCapabilitiesFromNewznab(caps)

		return unifiedCaps, nil
	default:
		return nil, errors.New("unsupported feed type: %s", feed.Type)
	}
}

func (s *service) FetchCapsByID(ctx context.Context, id int) (*domain.FeedCapabilities, error) {
	feed, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	caps, err := s.FetchCaps(ctx, feed)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateCapabilities(ctx, feed.ID, caps); err != nil {
		return nil, err
	}

	return caps, nil
}
