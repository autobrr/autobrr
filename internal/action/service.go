package action

import (
	"context"

	"github.com/asaskevich/EventBus"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/download_client"
	"github.com/autobrr/autobrr/internal/logger"
)

type Service interface {
	Store(ctx context.Context, action domain.Action) (*domain.Action, error)
	List(ctx context.Context) ([]domain.Action, error)
	Delete(actionID int) error
	DeleteByFilterID(ctx context.Context, filterID int) error
	ToggleEnabled(actionID int) error

	RunAction(action *domain.Action, release domain.Release) ([]string, error)
	CheckCanDownload(actions []domain.Action) bool
}

type service struct {
	log       logger.Logger
	repo      domain.ActionRepo
	clientSvc download_client.Service
	bus       EventBus.Bus
}

func NewService(log logger.Logger, repo domain.ActionRepo, clientSvc download_client.Service, bus EventBus.Bus) Service {
	return &service{
		log:       log,
		repo:      repo,
		clientSvc: clientSvc,
		bus:       bus,
	}
}

func (s *service) Store(ctx context.Context, action domain.Action) (*domain.Action, error) {
	return s.repo.Store(ctx, action)
}

func (s *service) Delete(actionID int) error {
	return s.repo.Delete(actionID)
}

func (s *service) DeleteByFilterID(ctx context.Context, filterID int) error {
	return s.repo.DeleteByFilterID(ctx, filterID)
}

func (s *service) List(ctx context.Context) ([]domain.Action, error) {
	return s.repo.List(ctx)
}

func (s *service) ToggleEnabled(actionID int) error {
	return s.repo.ToggleEnabled(actionID)
}
