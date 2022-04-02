package release

import (
	"context"
	"fmt"
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Find(ctx context.Context, query domain.ReleaseQueryParams) (res []*domain.Release, nextCursor int64, count int64, err error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Store(ctx context.Context, release *domain.Release) error
	StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error
	Process(release domain.Release) error
	Delete(ctx context.Context) error
}

type service struct {
	repo      domain.ReleaseRepo
	actionSvc action.Service
}

func NewService(repo domain.ReleaseRepo, actionService action.Service) Service {
	return &service{
		repo:      repo,
		actionSvc: actionService,
	}
}

func (s *service) Find(ctx context.Context, query domain.ReleaseQueryParams) (res []*domain.Release, nextCursor int64, count int64, err error) {
	return s.repo.Find(ctx, query)
}

func (s *service) GetIndexerOptions(ctx context.Context) ([]string, error) {
	return s.repo.GetIndexerOptions(ctx)
}

func (s *service) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	return s.repo.Stats(ctx)
}

func (s *service) Store(ctx context.Context, release *domain.Release) error {
	_, err := s.repo.Store(ctx, release)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error {
	return s.repo.StoreReleaseActionStatus(ctx, actionStatus)
}

func (s *service) Process(release domain.Release) error {
	log.Trace().Msgf("start to process release: %+v", release)

	if release.Filter.Actions == nil {
		return fmt.Errorf("no actions for filter: %v", release.Filter.Name)
	}

	// smart episode?

	// run actions (watchFolder, test, exec, qBittorrent, Deluge etc.)
	err := s.actionSvc.RunActions(release.Filter.Actions, release)
	if err != nil {
		log.Error().Stack().Err(err).Msgf("error running actions for filter: %v", release.Filter.Name)
		return err
	}

	return nil
}

func (s *service) Delete(ctx context.Context) error {
	return s.repo.Delete(ctx)
}
