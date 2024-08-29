// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package proxy

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	netProxy "golang.org/x/net/proxy"
)

type Service interface {
	List(ctx context.Context) ([]domain.Proxy, error)
	FindByID(ctx context.Context, id int64) (*domain.Proxy, error)
	Store(ctx context.Context, p *domain.Proxy) error
	Update(ctx context.Context, p *domain.Proxy) error
	Delete(ctx context.Context, id int64) error
	Test(ctx context.Context, p *domain.Proxy) error
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
	// TODO update IRC handlers
	return s.repo.Update(ctx, proxy)
}

func (s *service) FindByID(ctx context.Context, id int64) (*domain.Proxy, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) List(ctx context.Context) ([]domain.Proxy, error) {
	return s.repo.List(ctx)
}

func (s *service) ToggleEnabled(ctx context.Context, id int64, enabled bool) error {
	// TODO update IRC handlers
	return s.repo.ToggleEnabled(ctx, id, enabled)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	// TODO update IRC handlers
	return s.repo.Delete(ctx, id)
}

func (s *service) Test(ctx context.Context, proxy *domain.Proxy) error {
	if !proxy.ValidProxyType() {
		return errors.New("invalid proxy type %s", proxy.Type)
	}

	if proxy.Addr == "" {
		return errors.New("proxy addr missing")
	}

	proxyUrl, err := url.Parse(proxy.Addr)
	if err != nil {
		return errors.Wrap(err, "could not parse proxy url: %s", proxy.Addr)
	}

	// set user and pass if not empty
	if proxy.User != "" && proxy.Pass != "" {
		proxyUrl.User = url.UserPassword(proxy.User, proxy.Pass)
	}

	switch proxy.Type {
	case domain.ProxyTypeSocks5:
		proxyDialer, err := netProxy.FromURL(proxyUrl, netProxy.Direct)
		if err != nil {
			return errors.Wrap(err, "could not create proxy dialer from url: %s", proxy.Addr)
		}

		proxyContextDialer, ok := proxyDialer.(netProxy.ContextDialer)
		if !ok {
			return errors.Wrap(err, "proxy dialer does not expose DialContext(): %v", proxyDialer)
		}

		httpClient := &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				DialContext: proxyContextDialer.DialContext,
			},
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://autobrr.com", nil)
		if err != nil {
			return errors.Wrap(err, "could not create proxy request")
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "could not connect to proxy server: %s", proxy.Addr)
		}

		if resp.StatusCode != http.StatusOK {
			return errors.New(resp.Status)
		}

	default:
		return errors.New("invalid proxy type: %s", proxy.Type)
	}

	return nil
}
