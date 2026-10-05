// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package notification

import (
	"context"
	"encoding/json"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/r3labs/sse/v2"
	"github.com/rs/zerolog"
)

// InboxStreamKey is the SSE stream that inbox messages and changes are published on.
const InboxStreamKey = "notifications"

const (
	inboxMaxMessages = 1000
	inboxMaxAge      = 30 * 24 * time.Hour
	inboxCleanupJob  = "notification-inbox-cleanup"
)

type inboxRepo interface {
	Find(ctx context.Context, params domain.InboxQueryParams) (*domain.FindInboxResponse, error)
	Store(ctx context.Context, msg *domain.InboxMessage) error
	MarkRead(ctx context.Context, messageIDs []int64, readAt time.Time) error
	Delete(ctx context.Context, messageIDs []int64) error
	DeleteAll(ctx context.Context) error
	Cleanup(ctx context.Context, params domain.InboxCleanupParams) (int64, error)
}

type ssePublisher interface {
	Publish(id string, event *sse.Event)
}

type inboxWriter interface {
	StoreInboxMessage(ctx context.Context, msg *domain.InboxMessage) error
}

type builtinSender struct {
	baseSender
	inbox inboxWriter
}

func NewBuiltinSender(settings *domain.Notification, inbox inboxWriter) *builtinSender {
	return &builtinSender{
		baseSender: newBaseSender("built-in", settings),
		inbox:      inbox,
	}
}

func (s *builtinSender) Send(ctx context.Context, payload domain.NotificationPayload) error {
	msg := &domain.InboxMessage{
		Event:        payload.Event,
		Title:        BuildTitle(payload.Event),
		Message:      payload.Message,
		ReleaseName:  payload.ReleaseName,
		Indexer:      payload.Indexer,
		FilterName:   payload.Filter,
		FilterID:     payload.FilterID,
		Action:       payload.Action,
		ActionClient: payload.ActionClient,
		Rejections:   payload.Rejections,
		URL:          payload.URL,
		CreatedAt:    time.Now().UTC(),
	}

	if msg.Message == msg.ReleaseName {
		msg.Message = ""
	}

	return s.inbox.StoreInboxMessage(ctx, msg)
}

func (s *Service) FindInbox(ctx context.Context, params domain.InboxQueryParams) (*domain.FindInboxResponse, error) {
	return s.inboxRepo.Find(ctx, params)
}

// MarkInboxRead marks the messages in messageIDs as read, or the whole inbox when messageIDs is empty.
func (s *Service) MarkInboxRead(ctx context.Context, messageIDs []int64) error {
	if err := s.inboxRepo.MarkRead(ctx, messageIDs, time.Now().UTC()); err != nil {
		return err
	}

	publishInboxChanged(s.sse)

	return nil
}

func (s *Service) DeleteInboxMessages(ctx context.Context, messageIDs []int64) error {
	if err := s.inboxRepo.Delete(ctx, messageIDs); err != nil {
		return err
	}

	publishInboxChanged(s.sse)

	return nil
}

func (s *Service) DeleteInbox(ctx context.Context) error {
	if err := s.inboxRepo.DeleteAll(ctx); err != nil {
		return err
	}

	publishInboxChanged(s.sse)

	return nil
}

// publishInboxChanged tells connected web clients to refetch the inbox. It carries no ids
// because deletes shift pages and the counts depend on each client's query filters.
func publishInboxChanged(publisher ssePublisher) {
	// r3labs/sse closes the subscriber's stream on an event with empty Data.
	publisher.Publish(InboxStreamKey, &sse.Event{
		Event: []byte("INBOX_CHANGED"),
		Data:  []byte("{}"),
	})
}

// StoreInboxMessage stores msg and publishes it to connected web clients.
func (s *Service) StoreInboxMessage(ctx context.Context, msg *domain.InboxMessage) error {
	if err := s.inboxRepo.Store(ctx, msg); err != nil {
		return err
	}

	data, err := json.Marshal(msg)
	if err != nil {
		s.log.Error().Err(err).Int64("message_id", msg.ID).Msg("could not marshal inbox message")
		return nil
	}

	s.sse.Publish(InboxStreamKey, &sse.Event{
		Event: []byte("NOTIFICATION"),
		Data:  data,
	})

	return nil
}

func (s *Service) startInboxCleanupJob() error {
	job := &InboxCleanupJob{
		log:  s.log.With().Str("job", inboxCleanupJob).Logger(),
		repo: s.inboxRepo,
		sse:  s.sse,
	}

	if _, err := s.scheduler.ScheduleJob(job, 1*time.Hour, inboxCleanupJob); err != nil {
		return err
	}

	go job.Run()

	return nil
}

// InboxCleanupJob keeps the inbox within its size and age limits.
type InboxCleanupJob struct {
	log  zerolog.Logger
	repo inboxRepo
	sse  ssePublisher
}

func (j *InboxCleanupJob) Run() {
	deleted, err := j.repo.Cleanup(context.Background(), domain.InboxCleanupParams{
		MaxMessages: inboxMaxMessages,
		OlderThan:   time.Now().UTC().Add(-inboxMaxAge),
	})
	if err != nil {
		j.log.Error().Err(err).Msg("could not clean up notification inbox")
		return
	}

	j.log.Debug().Int64("deleted", deleted).Msg("notification inbox cleaned up")

	if deleted > 0 {
		publishInboxChanged(j.sse)
	}
}
