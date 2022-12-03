package action

import (
	"context"
	"log"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/download_client"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/qbittorrent"

	"github.com/asaskevich/EventBus"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/rs/zerolog"
)

type Service interface {
	Store(ctx context.Context, action domain.Action) (*domain.Action, error)
	List(ctx context.Context) ([]domain.Action, error)
	Delete(actionID int) error
	DeleteByFilterID(ctx context.Context, filterID int) error
	ToggleEnabled(actionID int) error

	RunAction(ctx context.Context, action *domain.Action, release domain.Release) ([]string, error)
}

type qbitKey struct {
	I int    // type
	N string // name
}

type service struct {
	log       zerolog.Logger
	subLogger *log.Logger
	repo      domain.ActionRepo
	clientSvc download_client.Service
	bus       EventBus.Bus

	qbitClients map[qbitKey]qbittorrent.Client
}

func NewService(log logger.Logger, repo domain.ActionRepo, clientSvc download_client.Service, bus EventBus.Bus) Service {
	s := &service{
		log:         log.With().Str("module", "action").Logger(),
		repo:        repo,
		clientSvc:   clientSvc,
		bus:         bus,
		qbitClients: map[qbitKey]qbittorrent.Client{},
	}

	s.subLogger = zstdlog.NewStdLoggerWithLevel(s.log.With().Logger(), zerolog.TraceLevel)

	return s
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
