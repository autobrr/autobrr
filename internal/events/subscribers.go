// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package events

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/asaskevich/EventBus"
	"github.com/rs/zerolog"
)

type feedService interface {
	FindOne(ctx context.Context, params domain.FindOneParams) (*domain.Feed, error)
	Delete(ctx context.Context, id int) error
	ToggleIndexerEnabled(ctx context.Context, indexerID int, enabled bool) error
}

type notificationSender interface {
	Send(event domain.NotificationEvent, payload domain.NotificationPayload)
}

type releaseService interface {
	StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error
}

type Subscriber struct {
	log      zerolog.Logger
	eventbus EventBus.Bus

	feedSvc         feedService
	notificationSvc notificationSender
	releaseSvc      releaseService
}

func NewSubscribers(log zerolog.Logger, eventbus EventBus.Bus, feedSvc feedService, notificationSvc notificationSender, releaseSvc releaseService) *Subscriber {
	s := &Subscriber{
		log:             log.With().Str("module", "events").Logger(),
		eventbus:        eventbus,
		feedSvc:         feedSvc,
		notificationSvc: notificationSvc,
		releaseSvc:      releaseSvc,
	}

	s.Register()

	return s
}

func (s *Subscriber) Register() {
	s.eventbus.Subscribe(domain.EventReleaseStoreActionStatus, s.handleReleaseActionStatus)
	s.eventbus.Subscribe(domain.EventReleasePushStatus, s.handleReleasePushStatus)
	s.eventbus.Subscribe(domain.EventNotificationSend, s.handleSendNotification)
	s.eventbus.Subscribe(domain.EventIndexerDelete, s.handleIndexerDelete)
	s.eventbus.Subscribe(domain.EventIndexerToggleEnabled, s.handleIndexerToggleEnabled)
}

func (s *Subscriber) handleReleaseActionStatus(actionStatus *domain.ReleaseActionStatus) {
	s.log.Trace().Str("event", domain.EventReleaseStoreActionStatus).Interface("action_status", actionStatus).Msg("store action status")

	err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus)
	if err != nil {
		s.log.Error().Err(err).Msg("release action status store error")
	}
}

func (s *Subscriber) handleReleasePushStatus(actionStatus *domain.ReleaseActionStatus) {
	s.log.Trace().Str("event", domain.EventReleasePushStatus).Interface("action_status", actionStatus).Msg("release push")

	if err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus); err != nil {
		s.log.Error().Err(err).Msg("release push error")
	}
}

func (s *Subscriber) handleSendNotification(event *domain.NotificationEvent, payload *domain.NotificationPayload) {
	s.log.Trace().Str("event", domain.EventNotificationSend).Interface("notification_event", *event).Interface("payload", payload).Msg("send notification event")

	s.notificationSvc.Send(*event, *payload)
}

// handleIndexerDelete handle feed cleanup via event because feed service can't be imported in indexer service
func (s *Subscriber) handleIndexerDelete(indexer *domain.Indexer) {
	s.log.Trace().Str("event", domain.EventIndexerDelete).Int("indexer_id", int(indexer.ID)).Msg("indexer delete event")

	ctx := context.Background()

	if indexer.ImplementationIsFeed() {
		feedItem, err := s.feedSvc.FindOne(ctx, domain.FindOneParams{IndexerID: int(indexer.ID)})
		if err != nil {
			if errors.Is(err, domain.ErrRecordNotFound) {
				return
			}

			s.log.Error().Err(err).Int("indexer_id", int(indexer.ID)).Msg("indexer delete could not find feed")
			return
		}

		if err := s.feedSvc.Delete(ctx, feedItem.ID); err != nil {
			s.log.Error().Err(err).Int("feed_id", feedItem.ID).Msg("indexer delete could not delete feed")
		}

		s.log.Debug().Str("feed_name", feedItem.Name).Msg("removed feed")
	}
}

// handleIndexerToggleEnabled stops or starts the indexer's feed job via event because the feed
// service can't be imported in the indexer service
func (s *Subscriber) handleIndexerToggleEnabled(indexer *domain.Indexer) {
	s.log.Trace().Str("event", domain.EventIndexerToggleEnabled).Int("indexer_id", int(indexer.ID)).Bool("enabled", indexer.Enabled).Msg("indexer toggle enabled event")

	if !indexer.ImplementationIsFeed() {
		return
	}

	if err := s.feedSvc.ToggleIndexerEnabled(context.Background(), int(indexer.ID), indexer.Enabled); err != nil {
		s.log.Error().Err(err).Int("indexer_id", int(indexer.ID)).Msg("could not toggle feed job for indexer")
	}
}
