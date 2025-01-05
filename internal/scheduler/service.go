// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/update"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type Service interface {
	Start()
	Stop()
	ScheduleJob(job cron.Job, interval time.Duration, identifier string) (int, error)
	AddJob(job cron.Job, spec string, identifier string) (int, error)
	RemoveJobByIdentifier(id string) error
	GetNextRun(id string) (time.Time, error)
}

type service struct {
	log             zerolog.Logger
	config          *domain.Config
	version         string
	notificationSvc notification.Service
	updateSvc       *update.Service

	cron *cron.Cron
	jobs map[string]cron.EntryID
	m    sync.RWMutex
}

func NewService(log logger.Logger, config *domain.Config, notificationSvc notification.Service, updateSvc *update.Service) Service {
	return &service{
		log:             log.With().Str("module", "scheduler").Logger(),
		config:          config,
		notificationSvc: notificationSvc,
		updateSvc:       updateSvc,
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

	if s.config.CheckForUpdates {
		checkUpdates := &CheckUpdatesJob{
			Name:             "app-check-updates",
			Log:              s.log.With().Str("job", "app-check-updates").Logger(),
			Version:          s.version,
			NotifSvc:         s.notificationSvc,
			updateService:    s.updateSvc,
			lastCheckVersion: s.version,
		}

		if id, err := s.ScheduleJob(checkUpdates, 2*time.Hour, "app-check-updates"); err != nil {
			s.log.Error().Err(err).Msgf("scheduler.addAppJobs: error adding job: %v", id)
		}
	}
}

func (s *service) Stop() {
	s.log.Debug().Msg("scheduler.Stop")
	s.cron.Stop()
	return
}

// ScheduleJob takes a time duration and adds a job
func (s *service) ScheduleJob(job cron.Job, interval time.Duration, identifier string) (int, error) {
	id := s.cron.Schedule(cron.Every(interval), cron.NewChain(cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job))

	s.log.Debug().Msgf("scheduler.ScheduleJob: job successfully added: %s id %d", identifier, id)

	s.m.Lock()
	// add to job map
	s.jobs[identifier] = id
	s.m.Unlock()

	return int(id), nil
}

// AddJob takes a cron schedule and adds a job
func (s *service) AddJob(job cron.Job, spec string, identifier string) (int, error) {
	id, err := s.cron.AddJob(spec, cron.NewChain(cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job))

	if err != nil {
		return 0, errors.Wrap(err, "could not add job to cron")
	}

	s.log.Debug().Msgf("scheduler.AddJob: job successfully added: %s id %d", identifier, id)

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

func (s *service) GetNextRun(id string) (time.Time, error) {
	entry := s.getEntryById(id)

	if !entry.Valid() {
		return time.Time{}, nil
	}

	s.log.Debug().Msgf("scheduler.GetNextRun: %s next run: %s", id, entry.Next)

	return entry.Next, nil
}

func (s *service) getEntryById(id string) cron.Entry {
	s.m.Lock()
	defer s.m.Unlock()

	v, ok := s.jobs[id]
	if !ok {
		return cron.Entry{}
	}

	return s.cron.Entry(v)
}

type GenericJob struct {
	Name string
	Log  zerolog.Logger

	callback func()
}

func (j *GenericJob) Run() {
	j.callback()
}
