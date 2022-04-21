package feed

import (
	"errors"
	"fmt"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/internal/scheduler"
	"github.com/autobrr/autobrr/pkg/torznab"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

type Service interface {
	Start() error
	StartJob(i domain.IndexerDefinition) error
}

type feedInstance struct {
	Name              string
	IndexerIdentifier string
	URL               string
	ApiKey            string
	Interval          string
	Implementation    string
	Cron              string
}

type service struct {
	jobs map[string]cron.EntryID

	repo       domain.FeedCacheRepo
	indexerSvc indexer.Service
	releaseSvc release.Service
	scheduler  scheduler.Service
}

func NewService(repo domain.FeedCacheRepo, indexerSvc indexer.Service, releaseSvc release.Service, scheduler scheduler.Service) Service {
	return &service{
		repo:       repo,
		indexerSvc: indexerSvc,
		releaseSvc: releaseSvc,
		scheduler:  scheduler,
	}
}

func (s service) Start() error {
	// get all torznab indexer definitions
	indexers := s.indexerSvc.GetTorznabIndexers()
	for _, i := range indexers {
		if err := s.StartJob(i); err != nil {
			log.Error().Err(err).Msg("failed to initialize torznab job")
			continue
		}
	}

	return nil
}

func (s service) StartJob(i domain.IndexerDefinition) error {
	// get all torznab indexer definitions
	if !i.Enabled {
		return nil
	}

	// get torznab_url from settings
	url, ok := i.SettingsMap["torznab_url"]
	if !ok || url == "" {
		return nil
	}

	// get apikey if it's there from settings
	apiKey, ok := i.SettingsMap["apikey"]
	if !ok {
		return nil
	}

	f := feedInstance{
		Name:              i.Name,
		IndexerIdentifier: i.Identifier,
		Implementation:    i.Implementation,
		URL:               url,
		ApiKey:            apiKey,
		Interval:          "*/15 * * * *",
		Cron:              "*/15 * * * *",
	}

	switch i.Implementation {
	case "torznab":
		if err := s.AddTorznabJob(f); err != nil {
			log.Error().Err(err).Msg("failed to initialize feed")
			return err
		}
		//case "rss":

	}

	return nil
}

func (s service) AddTorznabJob(f feedInstance) error {
	if f.URL == "" {
		return errors.New("torznab feed requires URL")
	}
	if f.Cron == "" {
		f.Cron = "*/15 * * * *"
	}

	// setup logger
	l := log.With().Str("feed_name", f.Name).Logger()

	// setup torznab Client
	c := torznab.NewClient(f.URL, f.ApiKey)

	// create job
	job := &TorznabJob{
		Name:              f.Name,
		IndexerIdentifier: f.IndexerIdentifier,
		Client:            c,
		Log:               l,
		Repo:              s.repo,
		ReleaseSvc:        s.releaseSvc,
		URL:               f.URL,
	}

	// schedule job
	id, err := s.scheduler.AddJob(job, f.Cron, f.IndexerIdentifier)
	if err != nil {
		return fmt.Errorf("feeds,AddTorznabJob: add job failed: %w", err)
	}
	job.JobID = id
	//
	//// add to job map
	//s.jobs[f.IndexerIdentifier] = id

	log.Debug().Msgf("feeds.AddTorznabJob: %v", f.Name)

	return nil
}
