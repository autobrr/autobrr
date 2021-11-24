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
	s.eventbus.Subscribe("release:update-push-status", s.releaseUpdatePushStatus)
	s.eventbus.Subscribe("release:update-push-status-rejected", s.releaseUpdatePushStatusRejected)
}

func (s Subscriber) releaseUpdatePushStatus(id int64, status domain.ReleasePushStatus) {
	log.Trace().Msgf("event: 'release:update-push-status' release ID '%v' update push status: '%v'", id, status)

	err := s.releaseSvc.UpdatePushStatus(context.Background(), id, status)
	if err != nil {
		log.Error().Err(err).Msgf("events: error")
	}
}
func (s Subscriber) releaseUpdatePushStatusRejected(id int64, rejections string) {
	log.Trace().Msgf("event: 'release:update-push-status-rejected' release ID '%v' update push status rejected rejections: '%v'", id, rejections)

	err := s.releaseSvc.UpdatePushStatusRejected(context.Background(), id, rejections)
	if err != nil {
		log.Error().Err(err).Msgf("events: error")
	}
}
