// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
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
	log       zerolog.Logger
	eventBus  eventBus
	config    *domain.Config
	version   string
	updateSvc updateChecker

	cron *cron.Cron
	jobs map[string]cron.EntryID
	m    sync.RWMutex
}

func NewService(log zerolog.Logger, bus eventBus, config *domain.Config, updateSvc updateChecker) *Service {
	return &Service{
		log:       log.With().Str("module", "scheduler").Logger(),
		eventBus:  bus,
		config:    config,
		updateSvc: updateSvc,
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
		updateCheckerJob := NewUpdateCheckerJob(s.log, "app-check-updates", s.version, s.updateSvc, s.eventBus)

		if id, err := s.ScheduleJob(updateCheckerJob, 2*time.Hour, "app-check-updates"); err != nil {
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
}

// registerJob replaces any entry already registered under identifier while holding the lock, so
// concurrent (re)schedules of the same job cannot leave an orphaned entry firing alongside the
// new one.
func (s *Service) registerJob(identifier string, schedule cron.Schedule, job cron.Job) int {
	s.m.Lock()
	defer s.m.Unlock()

	if old, ok := s.jobs[identifier]; ok {
		s.cron.Remove(old)
	}

	id := s.cron.Schedule(schedule, cron.NewChain(cron.SkipIfStillRunning(cron.DiscardLogger)).Then(job))
	s.jobs[identifier] = id

	return int(id)
}

// ScheduleJob takes a time duration and adds a job
func (s *Service) ScheduleJob(job cron.Job, interval time.Duration, identifier string) (int, error) {
	id := s.registerJob(identifier, cron.Every(interval), job)

	s.log.Debug().Str("job", identifier).Int("job_id", id).Msg("job scheduled")

	return id, nil
}

// ScheduleJobAnchored adds a job that runs every interval anchored to its last run: the next fire
// is lastRun+interval, or shortly after now when the job never ran or is overdue. Jobs sharing an
// interval fire on distinct identifier-derived seconds so they do not run in lockstep.
func (s *Service) ScheduleJobAnchored(job cron.Job, interval time.Duration, lastRun time.Time, identifier string) (int, error) {
	schedule := newAnchoredSchedule(interval, lastRun, time.Now(), identifier)

	id := s.registerJob(identifier, schedule, job)

	s.log.Debug().Str("job", identifier).Int("job_id", id).Dur("interval", schedule.interval).Time("first_run", schedule.first).Msg("job scheduled anchored")

	return id, nil
}

// AddJob takes a cron schedule and adds a job
func (s *Service) AddJob(job cron.Job, spec string, identifier string) (int, error) {
	schedule, err := cron.ParseStandard(spec)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse cron spec: %s", spec)
	}

	id := s.registerJob(identifier, schedule, job)

	s.log.Debug().Str("job", identifier).Int("job_id", id).Msg("job added")

	return id, nil
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
