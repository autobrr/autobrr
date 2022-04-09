package release

import (
	"context"
	"github.com/autobrr/autobrr/internal/domain"
)

type Service interface {
	Find(ctx context.Context, query domain.ReleaseQueryParams) (res []*domain.Release, nextCursor int64, count int64, err error)
	GetIndexerOptions(ctx context.Context) ([]string, error)
	Stats(ctx context.Context) (*domain.ReleaseStats, error)
	Store(ctx context.Context, release *domain.Release) error
	StoreReleaseActionStatus(ctx context.Context, actionStatus *domain.ReleaseActionStatus) error
	Delete(ctx context.Context) error
}

type service struct {
	repo domain.ReleaseRepo
}

func NewService(repo domain.ReleaseRepo) Service {
	return &service{
		repo: repo,
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

func (s *service) Delete(ctx context.Context) error {
	return s.repo.Delete(ctx)
}
