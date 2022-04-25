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
}

func NewService() Service {
	return &service{
		cron: cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		)),
		jobs: map[string]cron.EntryID{},
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
