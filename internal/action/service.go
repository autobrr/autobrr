package action

import (
	"context"

	"github.com/asaskevich/EventBus"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/download_client"
)

type Service interface {
	Store(ctx context.Context, action domain.Action) (*domain.Action, error)
	Fetch() ([]domain.Action, error)
	Delete(actionID int) error
	DeleteByFilterID(ctx context.Context, filterID int) error
	ToggleEnabled(actionID int) error

	RunActions(actions []domain.Action, release domain.Release) error
	CheckCanDownload(actions []domain.Action) bool
}

type service struct {
	repo      domain.ActionRepo
	clientSvc download_client.Service
	bus       EventBus.Bus
}

func NewService(repo domain.ActionRepo, clientSvc download_client.Service, bus EventBus.Bus) Service {
	return &service{repo: repo, clientSvc: clientSvc, bus: bus}
}

func (s *service) Store(ctx context.Context, action domain.Action) (*domain.Action, error) {
	// validate data

	a, err := s.repo.Store(ctx, action)
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

func (s *service) DeleteByFilterID(ctx context.Context, filterID int) error {
	if err := s.repo.DeleteByFilterID(ctx, filterID); err != nil {
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
