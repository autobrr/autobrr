// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/newznab"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	errors2 "gitlab.com/tozd/go/errors"
)

type feedRepo interface {
	FindOne(ctx context.Context, params domain.FindOneParams) (*domain.Feed, error)
	FindByID(ctx context.Context, feedID int) (*domain.Feed, error)
	Find(ctx context.Context) ([]domain.Feed, error)
	GetLastRunDataByID(ctx context.Context, feedID int) (string, error)
	Store(ctx context.Context, feed *domain.Feed) error
	Update(ctx context.Context, feed *domain.Feed) error
	UpdateLastRun(ctx context.Context, feedID int) error
	UpdateLastRunWithData(ctx context.Context, feedID int, data string) error
	UpdateCapabilities(ctx context.Context, feedID int, caps *domain.FeedCapabilities) error
	ToggleEnabled(ctx context.Context, id int, enabled bool) error
	Delete(ctx context.Context, id int) error
}

type feedCacheRepo interface {
	Get(feedId int, key string) ([]byte, error)
	GetByFeed(ctx context.Context, feedId int) ([]domain.FeedCacheItem, error)
	GetCountByFeed(ctx context.Context, feedId int) (int, error)
	Exists(feedId int, key string) (bool, error)
	ExistingItems(ctx context.Context, feedId int, keys []string) (map[string]bool, error)
	Put(feedId int, key string, val []byte, ttl time.Time) error
	PutMany(ctx context.Context, items []domain.FeedCacheItem) error
	Delete(ctx context.Context, feedId int, key string) error
	DeleteByFeed(ctx context.Context, feedId int) error
	DeleteStale(ctx context.Context) error
	DeleteOrphaned(ctx context.Context) error
}

type schedulerService interface {
	ScheduleJobAnchored(job cron.Job, interval time.Duration, lastRun time.Time, identifier string) (int, error)
	AddJob(job cron.Job, spec string, identifier string) (int, error)
	RemoveJobByIdentifier(id string) error
	GetNextRun(id string) (time.Time, error)
}

type proxyService interface {
	FindByID(ctx context.Context, id int64) (*domain.Proxy, error)
}

type releaseService interface {
	ProcessMultipleFromIndexer(ctx context.Context, releases []*domain.Release, indexer domain.IndexerMinimal) error
}

type eventBus interface {
	OnIndexerDeleted(handler func(ctx context.Context, event events.IndexerChangeEvent) errors2.E) func()
	OnIndexerToggled(handler func(ctx context.Context, event events.IndexerChangeEvent) errors2.E) func()
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

// guardedJob wraps a scheduled feed job with the feed's run guard so it cannot overlap a force
// run; an overlapping fire is skipped, the next interval catches up.
type guardedJob struct {
	guard *sync.Mutex
	log   zerolog.Logger
	job   cron.Job
}

func (g *guardedJob) Run() {
	if !g.guard.TryLock() {
		g.log.Debug().Msg("feed refresh already running, skipping scheduled run")
		return
	}
	defer g.guard.Unlock()

	g.job.Run()
}

// feedKey creates a unique identifier to be used for controlling jobs in the scheduler
type feedKey struct {
	id int
}

// ToString creates a string of the unique id to be used for controlling jobs in the scheduler
func (k feedKey) ToString() string {
	return fmt.Sprintf("feed-%d", k.id)
}

type Service struct {
	log      zerolog.Logger
	eventBus eventBus

	// guards holds per-feed mutexes: run keeps a force run and a scheduled run from fetching
	// and processing the same feed concurrently (cron's SkipIfStillRunning only covers a single
	// entry and force runs use their own job instance); schedule serializes syncFeedJob so two
	// reconciliations cannot apply their start/stop in inverted order
	guards sync.Map

	repo       feedRepo
	cacheRepo  feedCacheRepo
	releaseSvc releaseService
	proxySvc   proxyService
	scheduler  schedulerService
}

type feedGuards struct {
	schedule sync.Mutex
	run      sync.Mutex
}

func NewService(log zerolog.Logger, eventBus eventBus, repo feedRepo, cacheRepo feedCacheRepo, releaseSvc releaseService, proxySvc proxyService, scheduler schedulerService) *Service {
	s := &Service{
		log:        log.With().Str("module", "feed").Logger(),
		eventBus:   eventBus,
		repo:       repo,
		cacheRepo:  cacheRepo,
		releaseSvc: releaseSvc,
		proxySvc:   proxySvc,
		scheduler:  scheduler,
	}

	s.setupEventListeners()

	return s
}

func (s *Service) setupEventListeners() {
	s.eventBus.OnIndexerToggled(func(ctx context.Context, event events.IndexerChangeEvent) errors2.E {
		indexer := event.Indexer
		s.log.Trace().Str("event", string(event.Type)).Int("indexer_id", int(indexer.ID)).Bool("enabled", indexer.Enabled).Msg("indexer toggle enabled event")

		if !indexer.ImplementationIsFeed() {
			return nil
		}

		if err := s.ToggleIndexerEnabled(ctx, int(indexer.ID)); err != nil {
			s.log.Error().Err(err).Int("indexer_id", int(indexer.ID)).Msg("could not toggle feed job for indexer")
		}

		return nil
	})

	s.eventBus.OnIndexerDeleted(func(ctx context.Context, event events.IndexerChangeEvent) errors2.E {
		indexer := event.Indexer

		s.log.Trace().Str("event", string(event.Type)).Int("indexer_id", int(indexer.ID)).Msg("indexer delete event")

		//ctx := context.Background()

		if indexer.ImplementationIsFeed() {
			feedItem, err := s.FindOne(ctx, domain.FindOneParams{IndexerID: int(indexer.ID)})
			if err != nil {
				if errors.Is(err, domain.ErrRecordNotFound) {
					return errors2.Wrap(err, "could not find feed item")
				}

				s.log.Error().Err(err).Int("indexer_id", int(indexer.ID)).Msg("indexer delete could not find feed")
				return errors2.Wrap(err, "could not find feed item")
			}

			if err := s.Delete(ctx, feedItem.ID); err != nil {
				s.log.Error().Err(err).Int("feed_id", feedItem.ID).Msg("indexer delete could not delete feed")
			}

			s.log.Debug().Str("feed_name", feedItem.Name).Msg("removed feed")
		}

		return nil
	})
}

func (s *Service) FindOne(ctx context.Context, params domain.FindOneParams) (*domain.Feed, error) {
	return s.repo.FindOne(ctx, params)
}

func (s *Service) FindByID(ctx context.Context, feedID int) (*domain.Feed, error) {
	return s.repo.FindByID(ctx, feedID)
}

func (s *Service) Find(ctx context.Context) ([]domain.Feed, error) {
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

func (s *Service) GetCacheByID(ctx context.Context, feedID int) ([]domain.FeedCacheItem, error) {
	return s.cacheRepo.GetByFeed(ctx, feedID)
}

func (s *Service) Store(ctx context.Context, feed *domain.Feed) error {
	if err := feed.Validate(); err != nil {
		return err
	}

	return s.repo.Store(ctx, feed)
}

func (s *Service) Update(ctx context.Context, feed *domain.Feed) error {
	return s.update(ctx, feed)
}

func (s *Service) Delete(ctx context.Context, feedID int) error {
	return s.delete(ctx, feedID)
}

func (s *Service) DeleteFeedCache(ctx context.Context, feedCacheID int) error {
	if _, err := s.repo.FindByID(ctx, feedCacheID); err != nil {
		return err
	}

	if err := s.cacheRepo.DeleteByFeed(ctx, feedCacheID); err != nil {
		return err
	}

	return nil
}

func (s *Service) DeleteFeedCacheStale(ctx context.Context) error {
	return s.cacheRepo.DeleteStale(ctx)
}

func (s *Service) ToggleEnabled(ctx context.Context, feedID int, enabled bool) error {
	return s.toggleEnabled(ctx, feedID, enabled)
}

func (s *Service) Test(ctx context.Context, feed *domain.Feed) error {
	return s.test(ctx, feed)
}

// ToggleIndexerEnabled stops or starts the feed job when its indexer is toggled; the feed's own
// enabled flag is left untouched so the job comes back when the indexer does.
// ToggleIndexerEnabled reconciles the feed job after its indexer was toggled. The persisted
// state is re-read instead of trusting the event: racing toggles can deliver events in a
// different order than their writes committed.
func (s *Service) ToggleIndexerEnabled(ctx context.Context, indexerID int) error {
	feed, err := s.repo.FindOne(ctx, domain.FindOneParams{IndexerID: indexerID})
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return nil
		}

		return errors.Wrap(err, "could not find feed for indexer %d", indexerID)
	}

	return s.syncFeedJob(ctx, feed.ID)
}

func (s *Service) Start() error {
	return s.start()
}

func (s *Service) update(ctx context.Context, feed *domain.Feed) error {
	if err := feed.Validate(); err != nil {
		return err
	}

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

	if err := s.syncFeedJob(ctx, feed.ID); err != nil {
		s.log.Error().Err(err).Msg("error restarting feed")
		return err
	}

	return nil
}

func (s *Service) delete(ctx context.Context, feedID int) error {
	f, err := s.repo.FindOne(ctx, domain.FindOneParams{FeedID: feedID})
	if err != nil {
		s.log.Error().Err(err).Msg("error finding feed")
		return err
	}

	s.log.Debug().Str("feed", f.Name).Msg("stopping and removing feed")

	if err := s.stopFeedJob(f.ID); err != nil {
		s.log.Error().Err(err).Str("feed", f.Name).Int("feed_id", f.ID).Msg("error stopping rss job")
		return err
	}

	// delete feed and cascade delete feed_cache by fk
	if err := s.repo.Delete(ctx, f.ID); err != nil {
		s.log.Error().Err(err).Str("feed", f.Name).Msg("error deleting feed")
		return err
	}

	// if foreign keys are not enforced in SQLite clear feed cache explicitly
	if err := s.cacheRepo.DeleteByFeed(ctx, feedID); err != nil {
		s.log.Error().Err(err).Str("feed", f.Name).Msg("error deleting feed cache")
	}

	s.guards.Delete(feedID)

	return nil
}

func (s *Service) toggleEnabled(ctx context.Context, feedID int, enabled bool) error {
	f, err := s.repo.FindOne(ctx, domain.FindOneParams{FeedID: feedID})
	if err != nil {
		s.log.Error().Err(err).Msg("error finding feed")
		return err
	}

	if err := s.repo.ToggleEnabled(ctx, feedID, enabled); err != nil {
		s.log.Error().Err(err).Msg("error feed toggle enabled")
		return err
	}

	if f.Enabled != enabled {
		s.log.Debug().Str("feed", f.Name).Bool("enabled", enabled).Msg("feed toggled")
	}

	return s.syncFeedJob(ctx, feedID)
}

func (s *Service) test(ctx context.Context, feed *domain.Feed) error {
	if err := feed.Validate(); err != nil {
		return err
	}

	existingFeed, err := s.repo.FindOne(ctx, domain.FindOneParams{FeedID: feed.ID})
	if err != nil {
		s.log.Error().Err(err).Int("feed_id", feed.ID).Msg("could not find feed")
		return err
	}

	if domain.IsRedactedString(feed.ApiKey) {
		feed.ApiKey = existingFeed.ApiKey
	}
	if domain.IsRedactedString(feed.Cookie) {
		feed.Cookie = existingFeed.Cookie
	}

	// add proxy conf
	if existingFeed.UseProxy {
		proxyConf, err := s.proxySvc.FindByID(ctx, feed.ProxyID)
		if err != nil {
			return errors.Wrap(err, "could not find proxy for indexer feed")
		}

		if proxyConf.Enabled {
			feed.Proxy = proxyConf
		}
	}

	// create sub logger
	subLogger := s.log.With().Str("feed", feed.Name).Logger()

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

	s.log.Info().Str("feed", feed.Name).Str("url", feed.URL).Msg("feed test successful")

	return nil
}

func (s *Service) testRSS(ctx context.Context, feed *domain.Feed) error {
	feedParser := NewFeedParser(time.Duration(feed.Timeout)*time.Second, feed.Cookie, feed.UserAgent, feed.TLSSkipVerify)

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

		s.log.Debug().Str("proxy", feed.Proxy.Name).Str("feed", feed.Name).Msg("using proxy for feed")
	}

	feedResponse, err := feedParser.ParseURLWithContext(ctx, feed.URL)
	if err != nil {
		s.log.Error().Err(err).Msg("error fetching rss feed items")
		return errors.Wrap(err, "error fetching rss feed items")
	}

	s.log.Info().Str("feed", feed.Name).Int("items_count", len(feedResponse.Items)).Msg("refreshing rss feed found items")

	return nil
}

func (s *Service) testTorznab(ctx context.Context, feed *domain.Feed, subLogger zerolog.Logger) error {
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

		s.log.Debug().Str("proxy", feed.Proxy.Name).Str("feed", feed.Name).Msg("using proxy for feed")
	}

	items, err := c.FetchFeed(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("error getting torznab feed")
		return err
	}

	s.log.Info().Str("feed", feed.Name).Int("items_count", len(items.Channel.Items)).Msg("refreshing torznab feed found items")

	return nil
}

func (s *Service) testNewznab(ctx context.Context, feed *domain.Feed, subLogger zerolog.Logger) error {
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

		s.log.Debug().Str("proxy", feed.Proxy.Name).Str("feed", feed.Name).Msg("using proxy for feed")
	}

	items, err := c.GetFeed(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("error getting newznab feed")
		return err
	}

	s.log.Info().Str("feed", feed.Name).Int("items_count", len(items.Channel.Items)).Msg("refreshing newznab feed found items")

	return nil
}

func (s *Service) start() error {
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

	s.log.Debug().Int("count", len(feeds)).Msg("preparing staggered start of feeds")

	// start in background to not block startup and signal.Notify signals until all feeds are started
	go func(feeds []domain.Feed) {
		for _, feed := range feeds {
			if !feed.Enabled {
				s.log.Trace().Str("feed", feed.Name).Msg("feed disabled, skipping")
				continue
			}

			if !feed.IndexerEnabled {
				s.log.Trace().Str("feed", feed.Name).Msg("indexer disabled, skipping feed")
				continue
			}

			// syncFeedJob re-fetches: the staggered sleeps make this snapshot minutes old by
			// the tail, and a feed or indexer toggled during the window must not be started
			// from stale state
			if err := s.syncFeedJob(context.TODO(), feed.ID); err != nil {
				s.log.Error().Err(err).Str("feed", feed.Name).Msg("failed to initialize feed job")
				continue
			}

			// add sleep for the next iteration to start staggered which should mitigate sqlite BUSY errors
			time.Sleep(time.Second * 5)
		}
	}(feeds)

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

func (s *Service) initializeFeedJob(fi feedInstance) (RefreshFeedJob, error) {
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
		s.log.Error().Err(err).Str("implementation", fi.Implementation).Msg("failed to initialize feed")
		return nil, err
	}

	return job, nil
}

func (s *Service) startJob(f *domain.Feed) error {
	// if it's not enabled we should not start it
	if !f.Enabled {
		return errors.New("feed %s not enabled", f.Name)
	}

	if !f.IndexerEnabled {
		return errors.New("indexer for feed %s not enabled", f.Name)
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

	s.log.Debug().Str("feed", f.Name).Msg("successfully started feed")

	return nil
}

func (s *Service) scheduleJob(fi feedInstance, job cron.Job) error {
	identifierKey := feedKey{fi.Feed.ID}.ToString()

	guarded := &guardedJob{
		guard: &s.feedGuard(fi.Feed.ID).run,
		log:   s.log.With().Str("feed", fi.Name).Int("feed_id", fi.Feed.ID).Logger(),
		job:   job,
	}

	if _, err := s.scheduler.ScheduleJobAnchored(guarded, fi.CronSchedule, fi.Feed.LastRun, identifierKey); err != nil {
		return errors.Wrap(err, "add job %s failed", identifierKey)
	}

	return nil
}

func (s *Service) feedGuard(feedID int) *feedGuards {
	guard, _ := s.guards.LoadOrStore(feedID, &feedGuards{})
	return guard.(*feedGuards)
}

// syncFeedJob converges the scheduled job with the persisted feed and indexer state. Every
// start/stop decision goes through here on a fresh fetch: enabled flags carried by events or
// pre-write snapshots can be stale when toggles race, and acting on them can strand an enabled
// feed without a job. The per-feed lock keeps racing reconciliations from applying out of order.
func (s *Service) syncFeedJob(ctx context.Context, feedID int) error {
	guard := s.feedGuard(feedID)
	guard.schedule.Lock()
	defer guard.schedule.Unlock()

	feed, err := s.repo.FindOne(ctx, domain.FindOneParams{FeedID: feedID})
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			return s.stopFeedJob(feedID)
		}

		return errors.Wrap(err, "could not find feed %d", feedID)
	}

	if !feed.Enabled || !feed.IndexerEnabled {
		return s.stopFeedJob(feed.ID)
	}

	return s.startJob(feed)
}

func (s *Service) createTorznabJob(f feedInstance) (RefreshFeedJob, error) {
	s.log.Debug().Str("feed", f.Name).Msg("create torznab job")

	if f.URL == "" {
		return nil, errors.New("torznab feed requires URL")
	}

	//if f.CronSchedule < 5*time.Minute {
	//	f.CronSchedule = 15 * time.Minute
	//}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Int("feed_id", f.Feed.ID).Str("implementation", f.Implementation).Logger()

	// setup torznab Client
	client := torznab.NewClient(torznab.Config{Host: f.URL, ApiKey: f.ApiKey, Timeout: f.Timeout, TLSSkipVerify: f.Feed.TLSSkipVerify})

	// create job
	job := NewTorznabJob(f.Feed, f.Name, l, f.URL, client, s.repo, s.cacheRepo, s.releaseSvc)

	return job, nil
}

func (s *Service) createNewznabJob(f feedInstance) (RefreshFeedJob, error) {
	s.log.Debug().Str("feed", f.Name).Msg("create newznab job")

	if f.URL == "" {
		return nil, errors.New("newznab feed requires URL")
	}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Int("feed_id", f.Feed.ID).Str("implementation", f.Implementation).Logger()

	// setup newznab Client
	client := newznab.NewClient(newznab.Config{Host: f.URL, ApiKey: f.ApiKey, Timeout: f.Timeout, TLSSkipVerify: f.Feed.TLSSkipVerify})

	// create job
	job := NewNewznabJob(f.Feed, f.Name, l, f.URL, client, s.repo, s.cacheRepo, s.releaseSvc)

	return job, nil
}

func (s *Service) createRSSJob(f feedInstance) (RefreshFeedJob, error) {
	s.log.Debug().Str("feed", f.Name).Msg("create rss job")

	if f.URL == "" {
		return nil, errors.New("rss feed requires URL")
	}

	//if f.CronSchedule < time.Duration(5*time.Minute) {
	//	f.CronSchedule = time.Duration(15 * time.Minute)
	//}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Int("feed_id", f.Feed.ID).Str("implementation", f.Implementation).Logger()

	// create job
	job := NewRSSJob(f.Feed, f.Name, l, f.URL, s.repo, s.cacheRepo, s.releaseSvc, f.Timeout)

	return job, nil
}

func (s *Service) createCleanupJob() error {
	// setup logger
	l := s.log.With().Str("job", "feed-cache-cleanup").Logger()

	// create job
	job := NewCleanupJob(l, s.cacheRepo)

	identifierKey := "feed-cache-cleanup"

	// schedule job for every day at 03:05
	if _, err := s.scheduler.AddJob(job, "5 3 * * *", identifierKey); err != nil {
		return errors.Wrap(err, "add job %s failed", identifierKey)
	}

	return nil
}

func (s *Service) stopFeedJob(id int) error {
	jobKey := feedKey{id}.ToString()
	// remove job from scheduler
	if err := s.scheduler.RemoveJobByIdentifier(jobKey); err != nil {
		return errors.Wrap(err, "stop job failed")
	}

	s.log.Debug().Int("feed_id", id).Str("job", jobKey).Msg("stop feed job")

	return nil
}

func (s *Service) GetNextRun(id int) (time.Time, error) {
	return s.scheduler.GetNextRun(feedKey{id}.ToString())
}

func (s *Service) GetLastRunData(ctx context.Context, id int) (string, error) {
	feed, err := s.repo.GetLastRunDataByID(ctx, id)
	if err != nil {
		return "", err
	}

	return feed, nil
}

func (s *Service) ForceRun(ctx context.Context, id int) error {
	feed, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if feed.UseProxy {
		proxyConf, err := s.proxySvc.FindByID(ctx, feed.ProxyID)
		if err != nil {
			return errors.Wrap(err, "could not find proxy for indexer feed")
		}

		if proxyConf.Enabled {
			feed.Proxy = proxyConf
		}
	}

	fi := newFeedInstance(feed)

	job, err := s.initializeFeedJob(fi)
	if err != nil {
		s.log.Error().Err(err).Msg("failed to initialize feed job")
		return err
	}

	guard := &s.feedGuard(feed.ID).run
	if !guard.TryLock() {
		return errors.New("feed %s is already running", feed.Name)
	}
	defer guard.Unlock()

	if err := job.RunE(ctx); err != nil {
		s.log.Error().Err(err).Msg("failed to refresh feed")
		return err
	}

	// the schedule anchors to last_run, which the run just updated; reschedule so the next
	// scheduled run is a full interval from now instead of from the previous run. Detached
	// from ctx so a client disconnect after the fetch cannot leave the stale schedule behind
	if err := s.syncFeedJob(context.WithoutCancel(ctx), id); err != nil {
		s.log.Error().Err(err).Int("feed_id", id).Msg("could not reschedule feed after force run")
	}

	return nil
}

func (s *Service) FetchCaps(ctx context.Context, feed *domain.Feed) (*domain.FeedCapabilities, error) {
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

func (s *Service) FetchCapsByID(ctx context.Context, id int) (*domain.FeedCapabilities, error) {
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
