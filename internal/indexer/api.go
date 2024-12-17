// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/mock"
	"github.com/autobrr/autobrr/pkg/btn"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/ggn"
	"github.com/autobrr/autobrr/pkg/ops"
	"github.com/autobrr/autobrr/pkg/red"

	"github.com/rs/zerolog"
)

type APIService interface {
	TestConnection(ctx context.Context, req domain.IndexerTestApiRequest) (bool, error)
	GetTorrentByID(ctx context.Context, indexer string, torrentID string) (*domain.TorrentBasic, error)
	AddClient(indexer string, settings map[string]string) error
	RemoveClient(indexer string) error
}

type apiClient interface {
	GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error)
	TestAPI(ctx context.Context) (bool, error)
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

func (s *apiService) GetTorrentByID(ctx context.Context, indexer string, torrentID string) (*domain.TorrentBasic, error) {
	client, err := s.getApiClient(indexer)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not get api client for: %s", indexer)
		return nil, errors.Wrap(err, "could not get torrent via api for indexer: %s", indexer)
	}

	s.log.Trace().Str("method", "GetTorrentByID").Msgf("%s fetching torrent from api...", indexer)

	torrent, err := client.GetTorrentByID(ctx, torrentID)
	if err != nil {
		s.log.Error().Stack().Err(err).Msgf("could not get torrent: %s from: %s", torrentID, indexer)
		return nil, err
	}

	if torrent == nil {
		return nil, errors.New("could not get torrent: %s from: %s", torrentID, indexer)
	}

	s.log.Trace().Str("method", "GetTorrentByID").Msgf("%s api successfully fetched torrent: %+v", indexer, torrent)

	return torrent, nil
}

func (s *apiService) TestConnection(ctx context.Context, req domain.IndexerTestApiRequest) (bool, error) {
	client, err := s.getClientForTest(req)
	if err != nil {
		return false, errors.New("could not init api client: %s", req.Identifier)
	}

	success, err := client.TestAPI(ctx)
	if err != nil {
		s.log.Error().Err(err).Msgf("error testing connection for api: %s", req.Identifier)
		return false, err
	}

	return success, nil
}

func (s *apiService) AddClient(indexer string, settings map[string]string) error {
	s.log.Trace().Msgf("api.Service.AddClient: init api client for: %s", indexer)

	// init client
	switch indexer {
	case "btn":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return errors.New("api.Service.AddClient: could not initialize btn client: missing var 'api_key'")
		}
		s.apiClients[indexer] = btn.NewClient(key)

	case "ggn":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return errors.New("api.Service.AddClient: could not initialize ggn client: missing var 'api_key'")
		}
		s.apiClients[indexer] = ggn.NewClient(key)

	case "redacted":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return errors.New("api.Service.AddClient: could not initialize red client: missing var 'api_key'")
		}
		s.apiClients[indexer] = red.NewClient(key)

	case "ops":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return errors.New("api.Service.AddClient: could not initialize orpheus client: missing var 'api_key'")
		}
		s.apiClients[indexer] = ops.NewClient(key)

	case "mock":
		s.apiClients[indexer] = mock.NewMockClient("mock")

	default:
		return errors.New("api.Service.AddClient: could not initialize client: unsupported indexer: %s", indexer)

	}

	return nil
}

func (s *apiService) getApiClient(indexer string) (apiClient, error) {
	client, ok := s.apiClients[indexer]
	if !ok {
		return nil, errors.New("could not find api client for: %s", indexer)
	}

	return client, nil
}

func (s *apiService) getClientForTest(req domain.IndexerTestApiRequest) (apiClient, error) {
	// init client
	switch req.Identifier {
	case "btn":
		if req.ApiKey == "" {
			return nil, errors.New("api.Service.AddClient: could not initialize btn client: missing var 'api_key'")
		}
		return btn.NewClient(req.ApiKey), nil

	case "ggn":
		if req.ApiKey == "" {
			return nil, errors.New("api.Service.AddClient: could not initialize ggn client: missing var 'api_key'")
		}
		return ggn.NewClient(req.ApiKey), nil

	case "redacted":
		if req.ApiKey == "" {
			return nil, errors.New("api.Service.AddClient: could not initialize red client: missing var 'api_key'")
		}
		return red.NewClient(req.ApiKey), nil

	case "ops":
		if req.ApiKey == "" {
			return nil, errors.New("api.Service.AddClient: could not initialize orpheus client: missing var 'api_key'")
		}
		return ops.NewClient(req.ApiKey), nil

	case "mock":
		return mock.NewMockClient("mock"), nil

	default:
		return nil, errors.New("api.Service.AddClient: could not initialize client: unsupported indexer: %s", req.Identifier)

	}
}

func (s *apiService) RemoveClient(indexer string) error {
	_, ok := s.apiClients[indexer]
	if ok {
		delete(s.apiClients, indexer)
	}

	return nil
}
