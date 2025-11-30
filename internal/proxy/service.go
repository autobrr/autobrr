// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
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
	"github.com/autobrr/autobrr/pkg/sharedhttp"

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

	repo  domain.ProxyRepo
	cache map[int64]*domain.Proxy
}

func NewService(log logger.Logger, repo domain.ProxyRepo) Service {
	return &service{
		log:   log.With().Str("module", "proxy").Logger(),
		repo:  repo,
		cache: make(map[int64]*domain.Proxy),
	}
}

func (s *service) Store(ctx context.Context, proxy *domain.Proxy) error {
	if err := proxy.Validate(); err != nil {
		return errors.Wrap(err, "validation error")
	}

	err := s.repo.Store(ctx, proxy)
	if err != nil {
		return err
	}

	s.cache[proxy.ID] = proxy

	return nil
}

func (s *service) Update(ctx context.Context, proxy *domain.Proxy) error {
	if err := proxy.Validate(); err != nil {
		return errors.Wrap(err, "validation error")
	}

	existingProxy, err := s.repo.FindByID(ctx, proxy.ID)
	if err != nil {
		return err
	}

	if domain.IsRedactedString(proxy.Pass) {
		proxy.Pass = existingProxy.Pass
	}

	if err := s.repo.Update(ctx, proxy); err != nil {
		return err
	}

	s.cache[proxy.ID] = proxy

	// TODO update IRC handlers

	return nil
}

func (s *service) FindByID(ctx context.Context, id int64) (*domain.Proxy, error) {
	if proxy, ok := s.cache[id]; ok {
		return proxy, nil
	}

	return s.repo.FindByID(ctx, id)
}

func (s *service) List(ctx context.Context) ([]domain.Proxy, error) {
	return s.repo.List(ctx)
}

func (s *service) ToggleEnabled(ctx context.Context, id int64, enabled bool) error {
	err := s.repo.ToggleEnabled(ctx, id, enabled)
	if err != nil {
		return err
	}

	v, ok := s.cache[id]
	if !ok {
		v.Enabled = !enabled
		s.cache[id] = v
	}

	// TODO update IRC handlers

	return nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	delete(s.cache, id)

	// TODO update IRC handlers

	return nil
}

func (s *service) Test(ctx context.Context, proxy *domain.Proxy) error {
	if !proxy.ValidProxyType() {
		return errors.New("invalid proxy type %s", proxy.Type)
	}

	if proxy.ID > 0 {
		existingProxy, err := s.repo.FindByID(ctx, proxy.ID)
		if err != nil {
			return err
		}

		if domain.IsRedactedString(proxy.Pass) {
			proxy.Pass = existingProxy.Pass
		}
	}

	if proxy.Addr == "" {
		return errors.New("proxy addr missing")
	}

	httpClient, err := GetProxiedHTTPClient(proxy)
	if err != nil {
		return errors.Wrap(err, "could not get http client")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://autobrr.com", nil)
	if err != nil {
		return errors.Wrap(err, "could not create proxy request")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "could not connect to proxy server: %s", proxy.Addr)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("got unexpected status code: %d", resp.StatusCode)
	}

	s.log.Debug().Msgf("proxy %s test OK!", proxy.Addr)

	return nil
}

func GetProxiedHTTPClient(p *domain.Proxy) (*http.Client, error) {
	proxyUrl, err := url.Parse(p.Addr)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse proxy url: %s", p.Addr)
	}

	// set user and pass if not empty
	if p.User != "" && p.Pass != "" {
		proxyUrl.User = url.UserPassword(p.User, p.Pass)
	}

	transport := sharedhttp.ProxyTransport.Clone()

	switch p.Type {
	case domain.ProxyTypeSocks5:
		proxyDialer, err := netProxy.FromURL(proxyUrl, netProxy.Direct)
		if err != nil {
			return nil, errors.Wrap(err, "could not create proxy dialer from url: %s", p.Addr)
		}

		proxyContextDialer, ok := proxyDialer.(netProxy.ContextDialer)
		if !ok {
			return nil, errors.Wrap(err, "proxy dialer does not expose DialContext(): %v", proxyDialer)
		}

		transport.Proxy = nil
		transport.DialContext = proxyContextDialer.DialContext
	case domain.ProxyTypeHTTP:
		transport.Proxy = http.ProxyURL(proxyUrl)

	default:
		return nil, errors.New("invalid proxy type: %s", p.Type)
	}

	client := &http.Client{
		Timeout:   60 * time.Second,
		Transport: transport,
	}

	return client, nil
}
