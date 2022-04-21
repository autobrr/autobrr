package feed

import (
	"errors"
	"fmt"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/indexer"
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
	Name              string
	IndexerIdentifier string
	URL               string
	ApiKey            string
	Interval          string
	Implementation    string
	Cron              string
}

type service struct {
	cron  *cron.Cron
	feeds []feedInstance
	jobs  map[string]cron.EntryID

	repo       domain.FeedCacheRepo
	releaseSvc release.Service
	indexerSvc indexer.Service
}

func NewService(repo domain.FeedCacheRepo, releaseSvc release.Service, indexerSvc indexer.Service) Service {
	return &service{
		repo:       repo,
		releaseSvc: releaseSvc,
		indexerSvc: indexerSvc,
		cron: cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		)),
	}
}

func (s service) Start() error {
	// get all torznab indexer definitions
	indexers := s.indexerSvc.GetTorznabIndexers()
	for _, i := range indexers {
		if err := s.startJob(i); err != nil {
			log.Error().Err(err).Msg("failed to initialize torznab job")
			continue
		}
	}

	// start cron scheduler
	// TODO move out to main?
	s.cron.Start()

	return nil
}

func (s service) startJob(i domain.IndexerDefinition) error {
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

func (s service) Stop() {
	s.cron.Stop()
}

func (s service) AddTorznabJob(f feedInstance) error {
	if f.URL == "" {
		return errors.New("torznab feed requires url")
	}
	if f.Cron == "" {
		f.Cron = "*/15 * * * *"
	}

	// setup logger
	l := log.With().Str("feed_name", f.Name).Logger()

	// setup torznab client
	c := torznab.NewClient(f.URL, f.ApiKey)

	// create job
	job := &torznabJob{
		name:              f.Name,
		indexerIdentifier: f.IndexerIdentifier,
		client:            c,
		log:               l,
		repo:              s.repo,
		releaseSvc:        s.releaseSvc,

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

		// add to job map
		s.jobs[f.IndexerIdentifier] = id
	}

	log.Debug().Msgf("feeds.AddTorznabJob: %v", f.Name)

	return nil
}

func (s service) StopTorznabJob(indexer string) error {
	v, ok := s.jobs[indexer]
	if !ok {
		return nil
	}

	s.cron.Remove(v)

	return nil
}
