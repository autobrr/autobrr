package download_client

import (
	"github.com/rs/zerolog/log"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/qbittorrent"
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
	// test
	err := s.testConnection(client)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) testConnection(client domain.DownloadClient) error {
	if client.Type == "QBITTORRENT" {
		qbtSettings := qbittorrent.Settings{
			Hostname: client.Host,
			Port:     uint(client.Port),
			Username: client.Username,
			Password: client.Password,
			SSL:      client.SSL,
		}

		qbt := qbittorrent.NewClient(qbtSettings)
		err := qbt.Login()
		if err != nil {
			log.Error().Err(err).Msgf("error logging into client: %v", client.Host)
			return err
		}
	}

	return nil
}
