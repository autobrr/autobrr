package feed

import (
	"fmt"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

type Service interface {
	Start() error
	Stop()
}

type feedInstance struct {
	Name     string
	URL      string
	Interval string
	Type     string
	Cron     string
}

type service struct {
	cron  *cron.Cron
	feeds []feedInstance

	repo       domain.FeedCacheRepo
	releaseSvc release.Service
}

func NewService(repo domain.FeedCacheRepo, releaseSvc release.Service) Service {
	return &service{
		repo:       repo,
		releaseSvc: releaseSvc,
		cron: cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		)),
	}
}

func (s service) Start() error {
	feeds := []feedInstance{{
		Name:     "test",
		URL:      "",
		Interval: "",
		Type:     "torznab",
		Cron:     "*/1 * * * *",
	}}

	for _, f := range feeds {
		switch f.Type {
		case "torznab":
			if err := s.AddTorznabJob(f); err != nil {
				log.Error().Err(err).Msg("failed to initialize feed")
				return err
			}
			//case "rss":

		}
	}

	s.cron.Start()

	return nil
}

func (s service) Stop() {
	s.cron.Stop()
}

func (s service) AddTorznabJob(f feedInstance) error {
	if f.Cron == "" {
		f.Cron = "*/1 * * * *" // TODO change to 10-15+ min
	}

	// setup logger
	l := log.With().Str("feed_name", f.Name).Logger()

	// setup torznab client
	c := torznab.NewClient(f.URL)
	c.ApiKey = ""

	// create job
	job := &torznabJob{
		name:       f.Name,
		client:     c,
		log:        l,
		repo:       s.repo,
		releaseSvc: s.releaseSvc,

		url:  f.URL,
		cron: s.cron,
	}

	// schedule job
	if id, err := s.cron.AddJob(f.Cron, cron.NewChain(
		cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job),
	); err != nil {
		return fmt.Errorf("add job fialed: %w", err)
	} else {
		job.jobID = id
	}

	log.Debug().Msgf("feeds.AddTorznabJob: %v", f.Name)

	return nil
}
