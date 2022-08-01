package scheduler

import (
	"time"

	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Service interface {
	Start()
	Stop()
	AddJob(job cron.Job, interval time.Duration, identifier string) (int, error)
	RemoveJobByID(id cron.EntryID) error
	RemoveJobByIdentifier(id string) error
}

type service struct {
	log             zerolog.Logger
	version         string
	notificationSvc notification.Service

	cron *cron.Cron
	jobs map[string]cron.EntryID
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

	s.AddJob(checkUpdates, time.Duration(36 * time.Hour), "app-check-updates")
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

type GenericJob struct {
	Name string
	Log  zerolog.Logger

	callback func()
}

func (j *GenericJob) Run() {
	j.callback()
}
