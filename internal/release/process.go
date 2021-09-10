package release

import (
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Process(announce domain.Announce) error
}

type service struct {
	actionSvc action.Service
}

func NewService(actionService action.Service) Service {
	return &service{actionSvc: actionService}
}

func (s *service) Process(announce domain.Announce) error {
	log.Trace().Msgf("start to process release: %+v", announce)

	if announce.Filter.Actions == nil {
		return fmt.Errorf("no actions for filter: %v", announce.Filter.Name)
	}

	// smart episode?

	// run actions (watchFolder, test, exec, qBittorrent, Deluge etc.)
	err := s.actionSvc.RunActions(announce.Filter.Actions, announce)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error running actions for filter: %v", announce.Filter.Name)
		return err
	}

	return nil
}
