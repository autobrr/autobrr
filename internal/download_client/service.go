// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package download_client

import (
	"context"
	"log"
	"sync"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/autobrr/go-qbittorrent"
	"github.com/dcarbone/zadapters/zstdlog"
	"github.com/rs/zerolog"
)

type Service interface {
	List(ctx context.Context) ([]domain.DownloadClient, error)
	FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error)
	Store(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error)
	Update(ctx context.Context, client domain.DownloadClient) (*domain.DownloadClient, error)
	Delete(ctx context.Context, clientID int) error
	Test(ctx context.Context, client domain.DownloadClient) error

	GetCachedClient(ctx context.Context, clientId int32) *domain.DownloadClientCached
}

type service struct {
	log       zerolog.Logger
	repo      domain.DownloadClientRepo
	subLogger *log.Logger

	qbitClients map[int32]*domain.DownloadClientCached
	m           sync.RWMutex
}

func NewService(log logger.Logger, repo domain.DownloadClientRepo) Service {
	s := &service{
		log:  log.With().Str("module", "download_client").Logger(),
		repo: repo,

		qbitClients: map[int32]*domain.DownloadClientCached{},
		m:           sync.RWMutex{},
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
	// basic validation of client
	if err := client.Validate(); err != nil {
		return nil, err
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
	// basic validation of client
	if err := client.Validate(); err != nil {
		return nil, err
	}

	// update
	c, err := s.repo.Update(ctx, client)
	if err != nil {
		s.log.Error().Err(err).Msgf("could not update download client: %+v", client)
		return nil, err
	}

	if client.Type == domain.DownloadClientTypeQbittorrent {
		s.m.Lock()
		delete(s.qbitClients, int32(client.ID))
		s.m.Unlock()
	}

	return c, err
}

func (s *service) Delete(ctx context.Context, clientID int) error {
	if err := s.repo.Delete(ctx, clientID); err != nil {
		s.log.Error().Err(err).Msgf("could not delete download client: %v", clientID)
		return err
	}

	s.m.Lock()
	delete(s.qbitClients, int32(clientID))
	s.m.Unlock()

	return nil
}

func (s *service) Test(ctx context.Context, client domain.DownloadClient) error {
	// basic validation of client
	if err := client.Validate(); err != nil {
		return err
	}

	// test
	if err := s.testConnection(ctx, client); err != nil {
		s.log.Error().Err(err).Msg("client connection test error")
		return err
	}

	return nil
}

func (s *service) GetCachedClient(ctx context.Context, clientId int32) *domain.DownloadClientCached {

	// check if client exists in cache
	s.m.RLock()
	cached, ok := s.qbitClients[clientId]
	s.m.RUnlock()

	if ok {
		return cached
	}

	// get client for action
	client, err := s.FindByID(ctx, clientId)
	if err != nil {
		return nil
	}

	if client == nil {
		return nil
	}

	qbtSettings := qbittorrent.Config{
		Host:          client.BuildLegacyHost(),
		Username:      client.Username,
		Password:      client.Password,
		TLSSkipVerify: client.TLSSkipVerify,
	}

	// setup sub logger adapter which is compatible with *log.Logger
	qbtSettings.Log = zstdlog.NewStdLoggerWithLevel(s.log.With().Str("type", "qBittorrent").Str("client", client.Name).Logger(), zerolog.TraceLevel)

	// only set basic auth if enabled
	if client.Settings.Basic.Auth {
		qbtSettings.BasicUser = client.Settings.Basic.Username
		qbtSettings.BasicPass = client.Settings.Basic.Password
	}

	qc := &domain.DownloadClientCached{
		Dc:  client,
		Qbt: qbittorrent.NewClient(qbtSettings),
	}

	cached = qc

	s.m.Lock()
	s.qbitClients[clientId] = cached
	s.m.Unlock()

	return cached
}
