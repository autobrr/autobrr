package download_client

import (
	"context"
	"errors"
	"log"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/rs/zerolog"
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
	log       zerolog.Logger
	repo      domain.DownloadClientRepo
	subLogger *log.Logger
}

func NewService(log logger.Logger, repo domain.DownloadClientRepo) Service {
	s := &service{
		log:  log.With().Str("module", "download_client").Logger(),
		repo: repo,
	}

	s.subLogger = zstdlog.NewStdLoggerWithLevel(s.log.With().Logger(), zerolog.TraceLevel)

	return s
}

func (s *service) List(ctx context.Context) ([]domain.DownloadClient, error) {
	clients, err := s.repo.List(ctx)
	if err != nil {
		s.log.Error().Err(err).Msg("could not list download clients")
		return nil, err
	}

	return clients, nil
}

func (s *service) FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error) {
	client, err := s.repo.FindByID(ctx, id)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not find download client by id: %v", id)
		return nil, err
	}

	return client, nil
}

func (s *service) Store(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error) {
	// validate data
	if client.Host == "" {
		return nil, errors.New("validation error: no host")
	} else if client.Type == "" {
		return nil, errors.New("validation error: no type")
	}

	// store
	c, err := s.repo.Store(ctx, client)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not store download client: %+v", client)
		return nil, err
	}

	return c, err
}

func (s *service) Update(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error) {
	// validate data
	if client.Host == "" {
		return nil, errors.New("validation error: no host")
	} else if client.Type == "" {
		return nil, errors.New("validation error: no type")
	}

	// update
	c, err := s.repo.Update(ctx, client)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not update download client: %+v", client)
		return nil, err
	}

	return c, err
}

func (s *service) Delete(ctx context.Context, clientID int) error {
	err := s.repo.Delete(ctx, clientID)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not delete download client: %v", clientID)
		return err
	}
	return nil
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
