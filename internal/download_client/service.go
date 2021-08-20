package download_client

import (
	"errors"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/qbittorrent"
	delugeClient "github.com/gdm85/go-libdeluge"
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
	} else if client.Port == 0 {
		return nil, errors.New("validation error: no port")
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
	} else if client.Port == 0 {
		return errors.New("validation error: no port")
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

func (s *service) testConnection(client domain.DownloadClient) error {
	switch client.Type {
	case domain.DownloadClientTypeQbittorrent:
		return s.testQbittorrentConnection(client)
	case domain.DownloadClientTypeDelugeV1, domain.DownloadClientTypeDelugeV2:
		return s.testDelugeConnection(client)
	}

	return nil
}

func (s *service) testQbittorrentConnection(client domain.DownloadClient) error {
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

	return nil
}

func (s *service) testDelugeConnection(client domain.DownloadClient) error {
	var deluge delugeClient.DelugeClient

	settings := delugeClient.Settings{
		Hostname:             client.Host,
		Port:                 uint(client.Port),
		Login:                client.Username,
		Password:             client.Password,
		DebugServerResponses: true,
		ReadWriteTimeout:     time.Second * 10,
	}

	switch client.Type {
	case "DELUGE_V1":
		deluge = delugeClient.NewV1(settings)

	case "DELUGE_V2":
		deluge = delugeClient.NewV2(settings)

	default:
		deluge = delugeClient.NewV2(settings)
	}

	// perform connection to Deluge server
	err := deluge.Connect()
	if err != nil {
		log.Error().Err(err).Msgf("error logging into client: %v", client.Host)
		return err
	}

	defer deluge.Close()

	// print daemon version
	ver, err := deluge.DaemonVersion()
	if err != nil {
		log.Error().Err(err).Msgf("could not get daemon version: %v", client.Host)
		return err
	}

	log.Debug().Msgf("daemon version: %v", ver)

	return nil
}
