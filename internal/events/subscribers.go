// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package events

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/asaskevich/EventBus"
	"github.com/rs/zerolog"
)

type Subscriber struct {
	log      zerolog.Logger
	eventbus EventBus.Bus

	feedSvc         feed.Service
	notificationSvc notification.Sender
	releaseSvc      release.Service
}

func NewSubscribers(log logger.Logger, eventbus EventBus.Bus, feedSvc feed.Service, notificationSvc notification.Sender, releaseSvc release.Service) Subscriber {
	s := Subscriber{
		log:             log.With().Str("module", "events").Logger(),
		eventbus:        eventbus,
		feedSvc:         feedSvc,
		notificationSvc: notificationSvc,
		releaseSvc:      releaseSvc,
	}

	s.Register()

	return s
}

func (s Subscriber) Register() {
	s.eventbus.Subscribe(domain.EventReleaseStoreActionStatus, s.handleReleaseActionStatus)
	s.eventbus.Subscribe(domain.EventReleasePushStatus, s.handleReleasePushStatus)
	s.eventbus.Subscribe(domain.EventNotificationSend, s.handleSendNotification)
	s.eventbus.Subscribe(domain.EventIndexerDelete, s.handleIndexerDelete)
}

func (s Subscriber) handleReleaseActionStatus(actionStatus *domain.ReleaseActionStatus) {
	s.log.Trace().Str("event", domain.EventReleaseStoreActionStatus).Msgf("store action status: '%+v'", actionStatus)

	err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus)
	if err != nil {
		s.log.Error().Err(err).Msgf("events: 'release:store-action-status' error")
	}
}

func (s Subscriber) handleReleasePushStatus(actionStatus *domain.ReleaseActionStatus) {
	s.log.Trace().Str("event", domain.EventReleasePushStatus).Msgf("events: 'release:push' '%+v'", actionStatus)

	if err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus); err != nil {
		s.log.Error().Err(err).Msgf("events: 'release:push' error")
	}
}

func (s Subscriber) handleSendNotification(event *domain.NotificationEvent, payload *domain.NotificationPayload) {
	s.log.Trace().Str("event", domain.EventNotificationSend).Msgf("send notification events: '%v' '%+v'", *event, payload)

	s.notificationSvc.Send(*event, *payload)
}

// handleIndexerDelete handle feed cleanup via event because feed service can't be imported in indexer service
func (s Subscriber) handleIndexerDelete(indexer *domain.Indexer) {
	s.log.Trace().Str("event", domain.EventIndexerDelete).Msgf("events: 'indexer:delete' '%d'", indexer.ID)

	ctx := context.Background()

	if indexer.ImplementationIsFeed() {
		feedItem, err := s.feedSvc.FindOne(ctx, domain.FindOneParams{IndexerID: int(indexer.ID)})
		if err != nil {
			if errors.Is(err, domain.ErrRecordNotFound) {
				return
			}

			s.log.Error().Err(err).Msgf("events: 'indexer:delete' error, could not find feed with indexer id: %d", indexer.ID)
			return
		}

		if err := s.feedSvc.Delete(ctx, feedItem.ID); err != nil {
			s.log.Error().Err(err).Msgf("events: 'indexer:delete' error, could not delete feed with id: %d", feedItem.ID)
		}

		s.log.Debug().Msgf("successfully removed feed: %s", feedItem.Name)
	}
}
