package indexer

import (
	"fmt"

	"github.com/rs/zerolog"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/mock"
	"github.com/autobrr/autobrr/pkg/btn"
	"github.com/autobrr/autobrr/pkg/ggn"
	"github.com/autobrr/autobrr/pkg/ptp"
	"github.com/autobrr/autobrr/pkg/red"
)

type APIService interface {
	TestConnection(indexer string) (bool, error)
	GetTorrentByID(indexer string, torrentID string) (*domain.TorrentBasic, error)
	AddClient(indexer string, settings map[string]string) error
	RemoveClient(indexer string) error
}

type apiClient interface {
	GetTorrentByID(torrentID string) (*domain.TorrentBasic, error)
	TestAPI() (bool, error)
}

type apiService struct {
	log        zerolog.Logger
	apiClients map[string]apiClient
}

func NewAPIService(log logger.Logger) APIService {
	return &apiService{
		log:        log.With().Str("module", "indexer-api").Logger(),
		apiClients: make(map[string]apiClient),
	}
}

func (s *apiService) GetTorrentByID(indexer string, torrentID string) (*domain.TorrentBasic, error) {
	v, ok := s.apiClients[indexer]
	if !ok {
		return nil, nil
	}

	s.log.Trace().Str("method", "GetTorrentByID").Msgf("'%v' trying to fetch torrent from api", indexer)

	t, err := v.GetTorrentByID(torrentID)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not get torrent: '%v' from: %v", torrentID, indexer)
		return nil, err
	}

	s.log.Trace().Str("method", "GetTorrentByID").Msgf("'%v' successfully fetched torrent from api: %+v", indexer, t)

	return t, nil
}

func (s *apiService) TestConnection(indexer string) (bool, error) {
	v, ok := s.apiClients[indexer]
	if !ok {
		return false, nil
	}

	t, err := v.TestAPI()
	if err != nil {
		return false, err
	}

	return t, nil
}

func (s *apiService) AddClient(indexer string, settings map[string]string) error {
	// basic validation
	if indexer == "" {
		return fmt.Errorf("api.Service.AddClient: validation falied: indexer can't be empty")
	} else if len(settings) == 0 {
		return fmt.Errorf("api.Service.AddClient: validation falied: settings can't be empty")
	}

	s.log.Trace().Msgf("api.Service.AddClient: init api client for '%v'", indexer)

	// init client
	switch indexer {
	case "btn":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return fmt.Errorf("api.Service.AddClient: could not initialize btn client: missing var 'api_key'")
		}
		s.apiClients[indexer] = btn.NewClient("", key)

	case "ptp":
		user, ok := settings["api_user"]
		if !ok || user == "" {
			return fmt.Errorf("api.Service.AddClient: could not initialize ptp client: missing var 'api_user'")
		}

		key, ok := settings["api_key"]
		if !ok || key == "" {
			return fmt.Errorf("api.Service.AddClient: could not initialize ptp client: missing var 'api_key'")
		}
		s.apiClients[indexer] = ptp.NewClient("", user, key)

	case "ggn":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return fmt.Errorf("api.Service.AddClient: could not initialize ggn client: missing var 'api_key'")
		}
		s.apiClients[indexer] = ggn.NewClient("", key)

	case "redacted":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return fmt.Errorf("api.Service.AddClient: could not initialize red client: missing var 'api_key'")
		}
		s.apiClients[indexer] = red.NewClient("", key)

	case "mock":
		s.apiClients[indexer] = mock.NewMockClient("", "mock")

	default:
		return fmt.Errorf("api.Service.AddClient: could not initialize client: unsupported indexer '%v'", indexer)

	}
	return nil
}

func (s *apiService) RemoveClient(indexer string) error {
	_, ok := s.apiClients[indexer]
	if ok {
		delete(s.apiClients, indexer)
	}

	return nil
}
