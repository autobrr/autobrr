package action

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

func (s *service) porla(action domain.Action, release domain.Release) ([]string, error) {
	s.log.Debug().Msgf("action Porla: %v", action.Name)

	client, err := s.clientSvc.FindByID(context.TODO(), action.ClientID)
	if err != nil {
		return nil, errors.Wrap(err, "error finding client: %v", action.ClientID)
	}

	if client == nil {
		return nil, errors.New("could not find client by id: %v", action.ClientID)
	}

	return nil, nil
}
