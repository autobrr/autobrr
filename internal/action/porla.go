package action

import "github.com/autobrr/autobrr/internal/domain"

func (s *service) porla(action domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action Porla: %v", action.Name)
	return nil, nil
}
