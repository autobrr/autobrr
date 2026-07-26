// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

type notificationSender interface {
	Send(event domain.NotificationEvent, payload domain.NotificationPayload)
}

type updateChecker interface {
	CheckUpdateAvailable(ctx context.Context) (*version.Release, error)
}

type Service struct {
	log             zerolog.Logger
	config          *domain.Config
	version         string
	notificationSvc notificationSender
	updateSvc       updateChecker

	cron *cron.Cron
	jobs map[string]cron.EntryID
	m    sync.RWMutex
}

func NewService(log logger.Logger, config *domain.Config, notificationSvc notificationSender, updateSvc updateChecker) *Service {
	return &Service{
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

func (s *Service) Start() error {
	s.log.Debug().Msg("scheduler.Start")

	// start scheduler
	s.cron.Start()

	// init jobs
	go s.addAppJobs()

	return nil
}

func (s *Service) addAppJobs() {
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
			s.log.Error().Err(err).Int("job_id", id).Msg("error adding job")
		}
	}

	tempDirCleanup := NewTempDirCleanupJob(s.log.With().Str("job", "temp-dir-cleanup").Logger())

	if id, err := s.AddJob(tempDirCleanup, "0 4 * * *", "temp-dir-cleanup"); err != nil {
		s.log.Error().Err(err).Int("job_id", id).Msg("error adding temp dir cleanup job")
	}
}

func (s *Service) Stop() {
	s.log.Debug().Msg("scheduler.Stop")
	s.cron.Stop()
	return
}

// ScheduleJob takes a time duration and adds a job
func (s *Service) ScheduleJob(job cron.Job, interval time.Duration, identifier string) (int, error) {
	id := s.cron.Schedule(cron.Every(interval), cron.NewChain(cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job))

	s.log.Debug().Str("job", identifier).Int("job_id", int(id)).Msg("job scheduled")

	s.m.Lock()
	// add to job map
	s.jobs[identifier] = id
	s.m.Unlock()

	return int(id), nil
}

// ScheduleJobJittered takes a time duration and adds a job, spreading jobs that share an interval
// across it so they do not all fire on the same second.
func (s *Service) ScheduleJobJittered(job cron.Job, interval time.Duration, identifier string) (int, error) {
	schedule := newJitteredSchedule(interval, identifier)

	id := s.cron.Schedule(schedule, cron.NewChain(cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job))

	s.log.Debug().Str("identifier", identifier).Int("entry_id", int(id)).Dur("interval", schedule.interval).Dur("offset", schedule.offset).Msg("scheduler.ScheduleJobJittered: job successfully added")

	s.m.Lock()
	s.jobs[identifier] = id
	s.m.Unlock()

	return int(id), nil
}

// AddJob takes a cron schedule and adds a job
func (s *Service) AddJob(job cron.Job, spec string, identifier string) (int, error) {
	id, err := s.cron.AddJob(spec, cron.NewChain(cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job))

	if err != nil {
		return 0, errors.Wrap(err, "could not add job to cron")
	}

	s.log.Debug().Str("job", identifier).Int("job_id", int(id)).Msg("job added")

	s.m.Lock()
	// add to job map
	s.jobs[identifier] = id
	s.m.Unlock()

	return int(id), nil
}

func (s *Service) RemoveJobByIdentifier(id string) error {
	s.m.Lock()
	defer s.m.Unlock()

	v, ok := s.jobs[id]
	if !ok {
		return nil
	}

	s.log.Debug().Str("job", id).Msg("removing job")

	// remove from cron
	s.cron.Remove(v)

	// remove from jobs map
	delete(s.jobs, id)

	return nil
}

func (s *Service) GetNextRun(id string) (time.Time, error) {
	entry := s.getEntryById(id)

	if !entry.Valid() {
		return time.Time{}, nil
	}

	s.log.Debug().Str("job", id).Time("next_run", entry.Next).Msg("job next run")

	return entry.Next, nil
}

func (s *Service) getEntryById(id string) cron.Entry {
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
