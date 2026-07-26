// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"context"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/internal/mock"
	"github.com/autobrr/autobrr/internal/proxy"
	"github.com/autobrr/autobrr/pkg/btn"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/ggn"
	"github.com/autobrr/autobrr/pkg/ops"
	"github.com/autobrr/autobrr/pkg/red"

	"github.com/rs/zerolog"
)

type apiService interface {
	TestConnection(ctx context.Context, req domain.IndexerTestApiRequest) (bool, error)
	GetTorrentByID(ctx context.Context, indexer string, torrentID string) (*domain.TorrentBasic, error)
	AddClient(indexer string, settings map[string]string, proxyID int64, useProxy bool) error
	RemoveClient(indexer string) error
}

type apiClient interface {
	GetTorrentByID(ctx context.Context, torrentID string) (*domain.TorrentBasic, error)
	TestAPI(ctx context.Context) (bool, error)
}

type proxySvc interface {
	FindByID(ctx context.Context, proxyID int64) (*domain.Proxy, error)
}

type APIService struct {
	log        zerolog.Logger
	apiClients map[string]apiClient
	proxySvc   proxySvc
}

func NewAPIService(log logger.Logger, proxySvc proxySvc) *APIService {
	return &APIService{
		log:        log.With().Str("module", "indexer-api").Logger(),
		apiClients: make(map[string]apiClient),
		proxySvc:   proxySvc,
	}
}

func (s *APIService) GetTorrentByID(ctx context.Context, indexer string, torrentID string) (*domain.TorrentBasic, error) {
	client, err := s.getApiClient(indexer)
	if err != nil {
		s.log.Error().Err(err).Str("indexer", indexer).Msg("could not get indexer api client")
		return nil, errors.Wrap(err, "could not get torrent via api for indexer: %s", indexer)
	}

	s.log.Trace().Str("method", "GetTorrentByID").Str("indexer", indexer).Str("torrent_id", torrentID).Msg("fetching torrent from indexer api...")

	start := time.Now()

	torrent, err := client.GetTorrentByID(ctx, torrentID)
	if err != nil {
		s.log.Error().Stack().Err(err).Str("indexer", indexer).Str("torrent_id", torrentID).Msg("could not get torrent from indexer api")
		return nil, err
	}

	if torrent == nil {
		return nil, errors.New("could not get torrent: %s from: %s", torrentID, indexer)
	}

	s.log.Trace().Str("method", "GetTorrentByID").Str("indexer", indexer).Interface("torrent", torrent).Dur("duration", time.Since(start)).Str("torrent_id", torrentID).Msg("indexer api successfully fetched torrent")

	return torrent, nil
}

func (s *APIService) TestConnection(ctx context.Context, req domain.IndexerTestApiRequest) (bool, error) {
	client, err := s.getClientForTest(req)
	if err != nil {
		return false, errors.New("could not init api client: %s", req.Identifier)
	}

	success, err := client.TestAPI(ctx)
	if err != nil {
		s.log.Error().Err(err).Str("indexer", req.Identifier).Msg("error testing connection for indexer api")
		return false, err
	}

	return success, nil
}

func (s *APIService) AddClient(indexer string, settings map[string]string, proxyID int64, useProxy bool) error {
	s.log.Trace().Str("indexer", indexer).Msg("api.Service.AddClient: init api client")

	var proxyHttpClient *http.Client
	if proxyID > 0 && useProxy {
		s.log.Trace().Str("indexer", indexer).Int64("proxy_id", proxyID).Msg("api.Service.AddClient: attaching proxy to indexer api client")

		p, err := s.proxySvc.FindByID(context.Background(), proxyID)
		if err != nil {
			return err
		}

		proxyClient, err := proxy.GetProxiedHTTPClient(p)
		if err != nil {
			return errors.Wrap(err, "could not get proxy client")
		}
		proxyHttpClient = proxyClient
	}

	subLogger := s.log.With().Str("indexer", indexer).Logger()

	// init client
	switch indexer {
	case "btn":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return errors.New("api.Service.AddClient: could not initialize btn client: missing var 'api_key'")
		}
		s.apiClients[indexer] = btn.NewClient(key, btn.WithHTTPClient(proxyHttpClient), btn.WithLog(subLogger))

	case "ggn":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return errors.New("api.Service.AddClient: could not initialize ggn client: missing var 'api_key'")
		}
		s.apiClients[indexer] = ggn.NewClient(key, ggn.WithHTTPClient(proxyHttpClient), ggn.WithLog(subLogger))

	case "redacted":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return errors.New("api.Service.AddClient: could not initialize red client: missing var 'api_key'")
		}
		s.apiClients[indexer] = red.NewClient(key, red.WithHTTPClient(proxyHttpClient), red.WithLog(subLogger))

	case "ops":
		key, ok := settings["api_key"]
		if !ok || key == "" {
			return errors.New("api.Service.AddClient: could not initialize orpheus client: missing var 'api_key'")
		}
		s.apiClients[indexer] = ops.NewClient(key, ops.WithHTTPClient(proxyHttpClient), ops.WithLog(subLogger))

	case "mock":
		s.apiClients[indexer] = mock.NewMockClient("mock")

	default:
		return errors.New("api.Service.AddClient: could not initialize client: unsupported indexer: %s", indexer)
	}

	return nil
}

func (s *APIService) getApiClient(indexer string) (apiClient, error) {
	client, ok := s.apiClients[indexer]
	if !ok {
		return nil, errors.New("could not find api client for: %s", indexer)
	}

	return client, nil
}

func (s *APIService) getClientForTest(req domain.IndexerTestApiRequest) (apiClient, error) {
	var proxyHttpClient *http.Client
	if req.ProxyID > 0 && req.UseProxy {
		s.log.Trace().Str("indexer", req.Identifier).Int64("proxy_id", req.ProxyID).Msg("api.Service.AddClient: attaching proxy to indexer api client")

		p, err := s.proxySvc.FindByID(context.Background(), req.ProxyID)
		if err != nil {
			return nil, err
		}

		proxyClient, err := proxy.GetProxiedHTTPClient(p)
		if err != nil {
			return nil, errors.Wrap(err, "could not get proxy client")
		}
		proxyHttpClient = proxyClient
	}

	subLogger := s.log.With().Str("indexer", req.Identifier).Logger()

	// init client
	switch req.Identifier {
	case "btn":
		if req.ApiKey == "" {
			return nil, errors.New("api.Service.AddClient: could not initialize btn client: missing var 'api_key'")
		}
		return btn.NewClient(req.ApiKey, btn.WithHTTPClient(proxyHttpClient), btn.WithLog(subLogger)), nil

	case "ggn":
		if req.ApiKey == "" {
			return nil, errors.New("api.Service.AddClient: could not initialize ggn client: missing var 'api_key'")
		}
		return ggn.NewClient(req.ApiKey, ggn.WithHTTPClient(proxyHttpClient), ggn.WithLog(subLogger)), nil

	case "redacted":
		if req.ApiKey == "" {
			return nil, errors.New("api.Service.AddClient: could not initialize red client: missing var 'api_key'")
		}
		return red.NewClient(req.ApiKey, red.WithHTTPClient(proxyHttpClient), red.WithLog(subLogger)), nil

	case "ops":
		if req.ApiKey == "" {
			return nil, errors.New("api.Service.AddClient: could not initialize orpheus client: missing var 'api_key'")
		}
		return ops.NewClient(req.ApiKey, ops.WithHTTPClient(proxyHttpClient), ops.WithLog(subLogger)), nil

	case "mock":
		return mock.NewMockClient("mock"), nil

	default:
		return nil, errors.New("api.Service.AddClient: could not initialize client: unsupported indexer: %s", req.Identifier)

	}
}

func (s *APIService) RemoveClient(indexer string) error {
	_, ok := s.apiClients[indexer]
	if ok {
		delete(s.apiClients, indexer)
	}

	return nil
}
