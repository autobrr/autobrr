package download_client

import (
	"context"
	"errors"

	"github.com/rs/zerolog"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
)

type Service interface {
	List(ctx context.Context) ([]domain.DownloadClient, error)
	FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error)
	Store(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error)
	Update(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error)
	Delete(ctx context.Context, clientID int) error
	Test(client domain.DownloadClient) error
}

type service struct {
	log  zerolog.Logger
	repo domain.DownloadClientRepo
}

func NewService(log logger.Logger, repo domain.DownloadClientRepo) Service {
	return &service{
		log:  log.With().Str("module", "download_client").Logger(),
		repo: repo,
	}
}

func (s *service) List(ctx context.Context) ([]domain.DownloadClient, error) {
	return s.repo.List(ctx)
}

func (s *service) FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Store(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error) {
	// validate data
	if client.Host == "" {
		return nil, errors.New("validation error: no host")
	} else if client.Type == "" {
		return nil, errors.New("validation error: no type")
	}

	// store
	return s.repo.Store(ctx, client)
}

func (s *service) Update(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error) {
	// validate data
	if client.Host == "" {
		return nil, errors.New("validation error: no host")
	} else if client.Type == "" {
		return nil, errors.New("validation error: no type")
	}

	// store
	return s.repo.Update(ctx, client)
}

func (s *service) Delete(ctx context.Context, clientID int) error {
	return s.repo.Delete(ctx, clientID)
}

func (s *service) Test(client domain.DownloadClient) error {
	// basic validation of client
	if client.Host == "" {
		return errors.New("validation error: no host")
	} else if client.Type == "" {
		return errors.New("validation error: no type")
	}

	// test
	if err := s.testConnection(client); err != nil {
		s.log.Err(err).Msg("client connection test error")
		return err
	}

	return nil
}
