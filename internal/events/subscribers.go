package events

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/notification"
	"github.com/autobrr/autobrr/internal/release"

	"github.com/asaskevich/EventBus"
)

type Subscriber struct {
	log             logger.Logger
	eventbus        EventBus.Bus
	notificationSvc notification.Service
	releaseSvc      release.Service
}

func NewSubscribers(log logger.Logger, eventbus EventBus.Bus, notificationSvc notification.Service, releaseSvc release.Service) Subscriber {
	s := Subscriber{
		log:             log,
		eventbus:        eventbus,
		notificationSvc: notificationSvc,
		releaseSvc:      releaseSvc,
	}

	s.Register()

	return s
}

func (s Subscriber) Register() {
	s.eventbus.Subscribe("release:store-action-status", s.releaseActionStatus)
	s.eventbus.Subscribe("release:push", s.releasePushStatus)
	s.eventbus.Subscribe("events:notification", s.sendNotification)
}

func (s Subscriber) releaseActionStatus(actionStatus *domain.ReleaseActionStatus) {
	s.log.Trace().Msgf("events: 'release:store-action-status' '%+v'", actionStatus)

	err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus)
	if err != nil {
		s.log.Error().Err(err).Msgf("events: 'release:store-action-status' error")
	}
}

func (s Subscriber) releasePushStatus(actionStatus *domain.ReleaseActionStatus) {
	s.log.Trace().Msgf("events: 'release:push' '%+v'", actionStatus)

	if err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus); err != nil {
		s.log.Error().Err(err).Msgf("events: 'release:push' error")
	}
}

func (s Subscriber) sendNotification(event domain.NotificationEvent, payload domain.NotificationPayload) {
	s.log.Trace().Msgf("events: '%v' '%+v'", event, payload)

	if err := s.notificationSvc.Send(event, payload); err != nil {
		s.log.Error().Err(err).Msgf("events: '%v' error sending notification", event)
	}
}
