package scheduler

import (
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Service interface {
	Start()
	Stop()
	AddJob(job cron.Job, interval time.Duration, identifier string) (int, error)
	RemoveJobByIdentifier(id string) error
}

type service struct {
	log             zerolog.Logger
	version         string
	notificationSvc notification.Service

	cron *cron.Cron
	jobs map[string]cron.EntryID
	m    sync.RWMutex
}

func NewService(log logger.Logger, version string, notificationSvc notification.Service) Service {
	return &service{
		log:             log.With().Str("module", "scheduler").Logger(),
		version:         version,
		notificationSvc: notificationSvc,
		cron: cron.New(cron.WithChain(
			cron.Recover(cron.DefaultLogger),
		)),
		jobs: map[string]cron.EntryID{},
	}
}

func (s *service) Start() {
	s.log.Debug().Msg("scheduler.Start")

	// start scheduler
	s.cron.Start()

	// init jobs
	go s.addAppJobs()

	return
}

func (s *service) addAppJobs() {
	time.Sleep(5 * time.Second)

	checkUpdates := &CheckUpdatesJob{
		Name:             "app-check-updates",
		Log:              s.log.With().Str("job", "app-check-updates").Logger(),
		Version:          s.version,
		NotifSvc:         s.notificationSvc,
		lastCheckVersion: "",
	}

	if id, err := s.AddJob(checkUpdates, time.Duration(36*time.Hour), "app-check-updates"); err != nil {
		s.log.Error().Err(err).Msgf("scheduler.addAppJobs: error adding job: %v", id)
	}
}

func (s *service) Stop() {
	s.log.Debug().Msg("scheduler.Stop")
	s.cron.Stop()
	return
}

func (s *service) AddJob(job cron.Job, interval time.Duration, identifier string) (int, error) {

	id := s.cron.Schedule(cron.Every(interval), cron.NewChain(
		cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job),
	)

	s.log.Debug().Msgf("scheduler.AddJob: job successfully added: %v", id)

	s.m.Lock()
	// add to job map
	s.jobs[identifier] = id
	s.m.Unlock()

	return int(id), nil
}

func (s *service) RemoveJobByIdentifier(id string) error {
	s.m.Lock()
	defer s.m.Unlock()

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

type GenericJob struct {
	Name string
	Log  zerolog.Logger

	callback func()
}

func (j *GenericJob) Run() {
	j.callback()
}
