package scheduler

import (
	"fmt"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
)

type Service interface {
	Start()
	Stop()
	AddJob(job cron.Job, interval string, identifier string) (int, error)
	RemoveJobByID(id cron.EntryID) error
	RemoveJobByIdentifier(id string) error
}

type service struct {
	cron *cron.Cron

	jobs map[string]cron.EntryID
	//feedCache  domain.FeedCacheRepo
	//releaseSvc release.Service
}

func NewService() Service {
	return &service{
		cron: cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		)),
		jobs: map[string]cron.EntryID{},
		//feedCache:  feedCache,
		//releaseSvc: releaseSvc,
	}
}

func (s *service) Start() {
	log.Debug().Msg("scheduler.Start")

	s.cron.Start()
	return
}

func (s *service) Stop() {
	log.Debug().Msg("scheduler.Stop")
	s.cron.Stop()
	return
}
func (s *service) AddJob(job cron.Job, interval string, identifier string) (int, error) {

	id, err := s.cron.AddJob(interval, cron.NewChain(
		cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job),
	)
	if err != nil {
		return 0, fmt.Errorf("scheduler: add job failed: %w", err)
	}

	log.Debug().Msgf("scheduler.AddJob: job successfully added: %v", id)

	// add to job map
	s.jobs[identifier] = id

	return int(id), nil
}

//func (s *service) AddTorznabJob(indexer domain.IndexerDefinition) error {
//
//	// get all torznab indexer definitions
//	if !indexer.Enabled {
//		return nil
//	}
//
//	// get torznab_url from settings
//	url, ok := indexer.SettingsMap["torznab_url"]
//	if !ok || url == "" {
//		return nil
//	}
//
//	// get apikey if it's there from settings
//	apiKey, ok := indexer.SettingsMap["apikey"]
//	if !ok {
//		return nil
//	}
//
//	cronSchedule := "*/15 * * * *"
//	//if f.Cron == "" {
//	//	f.Cron = "*/15 * * * *"
//	//}
//
//	// setup logger
//	l := log.With().Str("feed_name", indexer.Name).Logger()
//
//	// setup torznab client
//	c := torznab.NewClient(url, apiKey)
//
//	// create job
//	job := &feed.TorznabJob{
//		Name:              indexer.Name,
//		IndexerIdentifier: indexer.Identifier,
//		Client:            c,
//		Log:               l,
//		Repo:              s.feedCache,
//		ReleaseSvc:        s.releaseSvc,
//		URL:               url,
//	}
//
//	// schedule job
//	id, err := s.AddJob(job, cronSchedule, indexer.Identifier)
//	if err != nil {
//		return fmt.Errorf("feeds,AddTorznabJob: add job failed: %w", err)
//	}
//	job.JobID = id
//
//	log.Debug().Msgf("feeds.AddTorznabJob: %v", indexer.Name)
//
//	//switch indexer.Implementation {
//	//case "torznab":
//	//	if err := s.AddTorznabJob(f); err != nil {
//	//		log.Error().Err(err).Msg("failed to initialize feed")
//	//		return err
//	//	}
//	//	//case "rss":
//	//}
//
//	return nil
//}

func (s *service) RemoveJobByID(id cron.EntryID) error {
	v, ok := s.jobs[""]
	if !ok {
		return nil
	}

	s.cron.Remove(v)
	return nil
}

func (s *service) RemoveJobByIdentifier(id string) error {
	v, ok := s.jobs[id]
	if !ok {
		return nil
	}

	log.Debug().Msgf("scheduler.Remove: removing job: %v", id)

	// remove from cron
	s.cron.Remove(v)

	// remove from jobs map
	delete(s.jobs, id)

	return nil
}
