package download_client

import (
	"errors"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog/log"
)

type Service interface {
	List() ([]domain.DownloadClient, error)
	FindByID(id int32) (*domain.DownloadClient, error)
	Store(client domain.DownloadClient) (*domain.DownloadClient, error)
	Delete(clientID int) error
	Test(client domain.DownloadClient) error
}

type service struct {
	repo domain.DownloadClientRepo
}

func NewService(repo domain.DownloadClientRepo) Service {
	return &service{repo: repo}
}

func (s *service) List() ([]domain.DownloadClient, error) {
	clients, err := s.repo.List()
	if err != nil {
		return nil, err
	}

	return clients, nil
}

func (s *service) FindByID(id int32) (*domain.DownloadClient, error) {
	client, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (s *service) Store(client domain.DownloadClient) (*domain.DownloadClient, error) {
	// validate data
	if client.Host == "" {
		return nil, errors.New("validation error: no host")
	} else if client.Type == "" {
		return nil, errors.New("validation error: no type")
	}

	// store
	c, err := s.repo.Store(client)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (s *service) Delete(clientID int) error {
	if err := s.repo.Delete(clientID); err != nil {
		return err
	}

	log.Debug().Msgf("delete client: %v", clientID)

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
	err := s.testConnection(client)
	if err != nil {
		return err
	}

	return nil
}
