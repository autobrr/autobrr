package release

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Find(ctx context.Context, query domain.QueryParams) (res []domain.Release, nextCursor int64, count int64, err error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Store(ctx context.Context, release *domain.Release) error
	StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error
	Process(release domain.Release) error
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

func (s *service) Find(ctx context.Context, query domain.QueryParams) (res []domain.Release, nextCursor int64, count int64, err error) {
	res, nextCursor, count, err = s.repo.Find(ctx, query)
	if err != nil {
		return
	}

	return
}

func (s *service) Stats(ctx context.Context) (*domain.ReleaseStats, error) {
	stats, err := s.repo.Stats(ctx)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func (s *service) Store(ctx context.Context, release *domain.Release) error {
	_, err := s.repo.Store(ctx, release)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error {
	err := s.repo.StoreReleaseActionStatus(ctx, actionStatus)
	if err != nil {
		return err
	}

	return nil
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
