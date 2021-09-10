package action

import (
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/download_client"
)

type Service interface {
	Store(action domain.Action) (*domain.Action, error)
	Fetch() ([]domain.Action, error)
	Delete(actionID int) error
	ToggleEnabled(actionID int) error

	RunActions(actions []domain.Action, announce domain.Announce) error
}

type service struct {
	repo      domain.ActionRepo
	clientSvc download_client.Service
}

func NewService(repo domain.ActionRepo, clientSvc download_client.Service) Service {
	return &service{repo: repo, clientSvc: clientSvc}
}

func (s *service) Store(action domain.Action) (*domain.Action, error) {
	// validate data

	a, err := s.repo.Store(action)
	if err != nil {
		return nil, err
	}

	return a, nil
}

func (s *service) Delete(actionID int) error {
	if err := s.repo.Delete(actionID); err != nil {
		return err
	}

	return nil
}

func (s *service) Fetch() ([]domain.Action, error) {
	actions, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	return actions, nil
}

func (s *service) ToggleEnabled(actionID int) error {
	if err := s.repo.ToggleEnabled(actionID); err != nil {
		return err
	}

	return nil
}
