package events

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/release"

	"github.com/asaskevich/EventBus"
	"github.com/rs/zerolog/log"
)

type Subscriber struct {
	eventbus   EventBus.Bus
	releaseSvc release.Service
}

func NewSubscribers(eventbus EventBus.Bus, releaseSvc release.Service) Subscriber {
	s := Subscriber{eventbus: eventbus, releaseSvc: releaseSvc}

	s.Register()

	return s
}

func (s Subscriber) Register() {
	s.eventbus.Subscribe("release:store-action-status", s.releaseActionStatus)
	s.eventbus.Subscribe("release:push-rejected", s.releasePushRejected)
	s.eventbus.Subscribe("release:push-approved", s.releasePushApproved)
}

func (s Subscriber) releaseActionStatus(actionStatus *domain.ReleaseActionStatus) {
	log.Trace().Msgf("events: 'release:store-action-status' '%+v'", actionStatus)

	err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus)
	if err != nil {
		log.Error().Err(err).Msgf("events: 'release:store-action-status' error")
	}
}

func (s Subscriber) releasePushRejected(actionStatus *domain.ReleaseActionStatus) {
	log.Trace().Msgf("events: 'release:push-rejected' '%+v'", actionStatus)

	err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus)
	if err != nil {
		log.Error().Err(err).Msgf("events: 'release:push-rejected' error")
	}
}

func (s Subscriber) releasePushApproved(actionStatus *domain.ReleaseActionStatus) {
	log.Trace().Msgf("events: 'release:push-approved' '%+v'", actionStatus)

	err := s.releaseSvc.StoreReleaseActionStatus(context.Background(), actionStatus)
	if err != nil {
		log.Error().Err(err).Msgf("events: 'release:push-approved' error")
	}
}
