package feed

import (
	"context"
	"fmt"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/rs/zerolog"
)

type Service interface {
	FindByID(ctx context.Context, id int) (*domain.Feed, error)
	FindByIndexerIdentifier(ctx context.Context, indexer string) (*domain.Feed, error)
	Find(ctx context.Context) ([]domain.Feed, error)
	Store(ctx context.Context, feed *domain.Feed) error
	Update(ctx context.Context, feed *domain.Feed) error
	ToggleEnabled(ctx context.Context, id int, enabled bool) error
	Delete(ctx context.Context, id int) error

	Start() error
}

type feedInstance struct {
	Name              string
	IndexerIdentifier string
	URL               string
	ApiKey            string
	Implementation    string
	CronSchedule      string
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
	feed, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find feed by id: %v", id)
		return nil, err
	}

	return feed, nil
}

func (s *service) FindByIndexerIdentifier(ctx context.Context, indexer string) (*domain.Feed, error) {
	feed, err := s.repo.FindByIndexerIdentifier(ctx, indexer)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find feed by indexer: %v", indexer)
		return nil, err
	}

	return feed, nil
}

func (s *service) Find(ctx context.Context) ([]domain.Feed, error) {
	feeds, err := s.repo.Find(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not find feeds")
		return nil, err
	}

	return feeds, err
}

func (s *service) Store(ctx context.Context, feed *domain.Feed) error {
	if err := s.repo.Store(ctx, feed); err != nil {
		s.log.Error().Err(err).Msgf("could not store feed: %+v", feed)
		return err
	}

	s.log.Debug().Msgf("successfully added feed: %+v", feed)

	return nil
}

func (s *service) Update(ctx context.Context, feed *domain.Feed) error {
	if err := s.update(ctx, feed); err != nil {
		s.log.Error().Err(err).Msgf("could not update feed: %+v", feed)
		return err
	}

	s.log.Debug().Msgf("successfully updated feed: %+v", feed)

	return nil
}

func (s *service) Delete(ctx context.Context, id int) error {
	if err := s.delete(ctx, id); err != nil {
		s.log.Error().Err(err).Msgf("could not delete feed by id: %v", id)
		return err
	}

	return nil
}

func (s *service) ToggleEnabled(ctx context.Context, id int, enabled bool) error {
	if err := s.toggleEnabled(ctx, id, enabled); err != nil {
		s.log.Error().Err(err).Msgf("could not toggle feed by id: %v", id)
		return err
	}
	return nil
}

func (s *service) update(ctx context.Context, feed *domain.Feed) error {
	if err := s.repo.Update(ctx, feed); err != nil {
		s.log.Error().Err(err).Msg("feed.Update: error updating feed")
		return err
	}

	if err := s.restartJob(feed); err != nil {
		s.log.Error().Err(err).Msg("feed.Update: error restarting feed")
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

	if err := s.stopTorznabJob(f.Indexer); err != nil {
		s.log.Error().Err(err).Msg("error stopping torznab job")
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

	s.log.Debug().Msgf("feed.Delete: stopping and removing feed: %v", f.Name)

	return nil
}

func (s *service) toggleEnabled(ctx context.Context, id int, enabled bool) error {
	if err := s.repo.ToggleEnabled(ctx, id, enabled); err != nil {
		s.log.Error().Err(err).Msg("feed.ToggleEnabled: error toggle enabled")
		return err
	}

	f, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msg("feed.ToggleEnabled: error finding feed")
		return err
	}

	if !enabled {
		if err := s.stopTorznabJob(f.Indexer); err != nil {
			s.log.Error().Err(err).Msg("feed.ToggleEnabled: error stopping torznab job")
			return err
		}

		s.log.Debug().Msgf("feed.ToggleEnabled: stopping feed: %v", f.Name)

		return nil
	}

	if err := s.startJob(*f); err != nil {
		s.log.Error().Err(err).Msg("feed.ToggleEnabled: error starting torznab job")
		return err
	}

	s.log.Debug().Msgf("feed.ToggleEnabled: started feed: %v", f.Name)

	return nil
}

func (s *service) Start() error {
	// get all torznab indexer definitions
	feeds, err := s.repo.Find(context.TODO())
	if err != nil {
		s.log.Error().Err(err).Msg("feed.Start: error finding feeds")
		return err
	}

	for _, i := range feeds {
		if err := s.startJob(i); err != nil {
			s.log.Error().Err(err).Msg("feed.Start: failed to initialize torznab job")
			continue
		}
	}

	return nil
}

func (s *service) restartJob(f *domain.Feed) error {
	// stop feed
	if err := s.stopTorznabJob(f.Indexer); err != nil {
		s.log.Error().Err(err).Msg("feed.restartJob: error stopping torznab job")
		return err
	}

	s.log.Debug().Msgf("feed.restartJob: stopping feed: %v", f.Name)

	if f.Enabled {
		if err := s.startJob(*f); err != nil {
			s.log.Error().Err(err).Msg("feed.restartJob: error starting torznab job")
			return err
		}

		s.log.Debug().Msgf("feed.restartJob: restarted feed: %v", f.Name)
	}

	return nil
}

func (s *service) startJob(f domain.Feed) error {
	// get all torznab indexer definitions
	if !f.Enabled {
		return nil
	}

	// get torznab_url from settings
	if f.URL == "" {
		return nil
	}

	// cron schedule to run every X minutes
	schedule := fmt.Sprintf("*/%d * * * *", f.Interval)

	fi := feedInstance{
		Name:              f.Name,
		IndexerIdentifier: f.Indexer,
		Implementation:    f.Type,
		URL:               f.URL,
		ApiKey:            f.ApiKey,
		CronSchedule:      schedule,
	}

	switch fi.Implementation {
	case string(domain.FeedTypeTorznab):
		if err := s.addTorznabJob(fi); err != nil {
			s.log.Error().Err(err).Msg("feed.startJob: failed to initialize feed")
			return err
		}
		//case "rss":

	}

	return nil
}

func (s *service) addTorznabJob(f feedInstance) error {
	if f.URL == "" {
		return errors.New("torznab feed requires URL")
	}
	if f.CronSchedule == "" {
		f.CronSchedule = "*/15 * * * *"
	}

	// setup logger
	l := s.log.With().Str("feed", f.Name).Logger()

	// setup torznab Client
	c := torznab.NewClient(f.URL, f.ApiKey)

	// create job
	job := NewTorznabJob(f.Name, f.IndexerIdentifier, l, f.URL, c, s.cacheRepo, s.releaseSvc)

	// schedule job
	id, err := s.scheduler.AddJob(job, f.CronSchedule, f.IndexerIdentifier)
	if err != nil {
		return errors.Wrap(err, "feed.AddTorznabJob: add job failed")
	}
	job.JobID = id

	// add to job map
	s.jobs[f.IndexerIdentifier] = id

	s.log.Debug().Msgf("feed.AddTorznabJob: %v", f.Name)

	return nil
}

func (s *service) stopTorznabJob(indexer string) error {
	// remove job from scheduler
	if err := s.scheduler.RemoveJobByIdentifier(indexer); err != nil {
		return errors.Wrap(err, "feed.stopTorznabJob: stop job failed")
	}

	s.log.Debug().Msgf("feed.stopTorznabJob: %v", indexer)

	return nil
}
