// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"net/http"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
)

type actionRepo interface {
	Store(ctx context.Context, action *domain.Action) error
	Update(ctx context.Context, action domain.Action) (*domain.Action, error)
	StoreFilterActions(ctx context.Context, filterID int64, actions []*domain.Action) ([]*domain.Action, error)
	FindByFilterID(ctx context.Context, filterID int, active *bool, withClient bool) ([]*domain.Action, error)
	List(ctx context.Context) ([]domain.Action, error)
	Get(ctx context.Context, req *domain.GetActionRequest) (*domain.Action, error)
	Delete(ctx context.Context, req *domain.DeleteActionRequest) error
	DeleteByFilterID(ctx context.Context, filterID int) error
	ToggleEnabled(actionID int) error
}

type downloaderService interface {
	FindByID(ctx context.Context, id int32) (*domain.Downloader, error)
	GetInstance(ctx context.Context, clientId int32) (*downloader.Instance, error)
}

type rlsDownloadService interface {
	DownloadRelease(ctx context.Context, rls *domain.Release) error
	ResolveMagnetURI(ctx context.Context, r *domain.Release) error
}

type eventBus interface {
	OnReleasePush(handler func(context.Context, events.ReleasePushEvent) error) func()
}

type Service struct {
	log            zerolog.Logger
	eventBus       eventBus
	repo           actionRepo
	downloaderSvc  downloaderService
	rlsDownloadSvc rlsDownloadService

	httpClient *http.Client
}

func NewService(log zerolog.Logger, bus eventBus, repo actionRepo, clientSvc downloaderService, downloadSvc rlsDownloadService) *Service {
	s := &Service{
		log:            log.With().Str("module", "action").Logger(),
		eventBus:       bus,
		repo:           repo,
		downloaderSvc:  clientSvc,
		rlsDownloadSvc: downloadSvc,

		httpClient: &http.Client{
			Timeout:   time.Second * 120,
			Transport: sharedhttp.TransportTLSInsecure,
		},
	}

	return s
}

func (s *Service) Store(ctx context.Context, action *domain.Action) error {
	return s.repo.Store(ctx, action)
}

func (s *Service) Update(ctx context.Context, action *domain.Action) (*domain.Action, error) {
	return s.repo.Update(ctx, *action)
}

func (s *Service) StoreFilterActions(ctx context.Context, filterID int64, actions []*domain.Action) ([]*domain.Action, error) {
	return s.repo.StoreFilterActions(ctx, filterID, actions)
}

func (s *Service) List(ctx context.Context) ([]domain.Action, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, req *domain.GetActionRequest) (*domain.Action, error) {
	a, err := s.repo.Get(ctx, req)
	if err != nil {
		return nil, err
	}

	// optionally attach download client to action
	if a.ClientID > 0 {
		client, err := s.downloaderSvc.FindByID(ctx, a.ClientID)
		if err != nil {
			return nil, err
		}

		a.Client = client
	}

	return a, nil
}

func (s *Service) FindByFilterID(ctx context.Context, filterID int, active *bool, withClient bool) ([]*domain.Action, error) {
	return s.repo.FindByFilterID(ctx, filterID, active, withClient)
}

func (s *Service) Delete(ctx context.Context, req *domain.DeleteActionRequest) error {
	return s.repo.Delete(ctx, req)
}

func (s *Service) DeleteByFilterID(ctx context.Context, filterID int) error {
	return s.repo.DeleteByFilterID(ctx, filterID)
}

func (s *Service) ToggleEnabled(actionID int) error {
	return s.repo.ToggleEnabled(actionID)
}
