package scheduler

import (
	"fmt"

	"github.com/autobrr/autobrr/internal/logger"

	"github.com/robfig/cron/v3"
)

type Service interface {
	Start()
	Stop()
	AddJob(job cron.Job, interval string, identifier string) (int, error)
	RemoveJobByID(id cron.EntryID) error
	RemoveJobByIdentifier(id string) error
}

type service struct {
	log  logger.Logger
	cron *cron.Cron

	jobs map[string]cron.EntryID
}

func NewService(log logger.Logger) Service {
	return &service{
		log: log,
		cron: cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		)),
		jobs: map[string]cron.EntryID{},
	}
}

func (s *service) Start() {
	s.log.Debug().Msg("scheduler.Start")

	s.cron.Start()
	return
}

func (s *service) Stop() {
	s.log.Debug().Msg("scheduler.Stop")
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

	s.log.Debug().Msgf("scheduler.AddJob: job successfully added: %v", id)

	// add to job map
	s.jobs[identifier] = id

	return int(id), nil
}

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

	s.log.Debug().Msgf("scheduler.Remove: removing job: %v", id)

	// remove from cron
	s.cron.Remove(v)

	// remove from jobs map
	delete(s.jobs, id)

	return nil
}
