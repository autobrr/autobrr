// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package proxy

import (
	"context"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/rs/zerolog"
)

type Service interface {
	List(ctx context.Context) ([]domain.Proxy, error)
	FindByID(ctx context.Context, id int64) (*domain.Proxy, error)
	Store(ctx context.Context, p *domain.Proxy) error
	Update(ctx context.Context, p *domain.Proxy) error
	Delete(ctx context.Context, id int64) error
}

type service struct {
	log zerolog.Logger

	repo domain.ProxyRepo
}

func NewService(log logger.Logger, repo domain.ProxyRepo) Service {
	return &service{
		log:  log.With().Str("module", "proxy").Logger(),
		repo: repo,
	}
}

func (s *service) Store(ctx context.Context, proxy *domain.Proxy) error {
	return s.repo.Store(ctx, proxy)
}

func (s *service) Update(ctx context.Context, proxy *domain.Proxy) error {
	return s.repo.Update(ctx, proxy)
}

func (s *service) FindByID(ctx context.Context, id int64) (*domain.Proxy, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) List(ctx context.Context) ([]domain.Proxy, error) {
	return s.repo.List(ctx)
}

func (s *service) ToggleEnabled(ctx context.Context, id int64, enabled bool) error {
	return s.repo.ToggleEnabled(ctx, id, enabled)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
