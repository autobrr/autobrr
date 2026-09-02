// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package proxy

import (
	"context"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"
	"github.com/autobrr/autobrr/pkg/sharedhttp"

	"github.com/rs/zerolog"
	netProxy "golang.org/x/net/proxy"
)

type proxyRepo interface {
	Store(ctx context.Context, p *domain.Proxy) error
	Update(ctx context.Context, p *domain.Proxy) error
	List(ctx context.Context) ([]domain.Proxy, error)
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*domain.Proxy, error)
	ToggleEnabled(ctx context.Context, id int64, enabled bool) error
	Usage(ctx context.Context, id int64) (*domain.ProxyUsage, error)
}

type eventBus interface {
	EmitProxy(ctx context.Context, event events.ProxyChangeEvent)
}

type Service struct {
	log      zerolog.Logger
	eventBus eventBus

	repo proxyRepo

	// cache is read from irc handler, feed and download goroutines while http handlers write it
	m     sync.RWMutex
	cache map[int64]*domain.Proxy
}

func NewService(log zerolog.Logger, eventBus eventBus, repo proxyRepo) *Service {
	return &Service{
		log:      log.With().Str("module", "proxy").Logger(),
		eventBus: eventBus,
		repo:     repo,
		cache:    make(map[int64]*domain.Proxy),
	}
}

func (s *Service) Store(ctx context.Context, proxy *domain.Proxy) error {
	if err := proxy.Validate(); err != nil {
		return errors.Wrap(err, "validation error")
	}

	err := s.repo.Store(ctx, proxy)
	if err != nil {
		return err
	}

	s.setCached(proxy)

	return nil
}

func (s *Service) Update(ctx context.Context, proxy *domain.Proxy) error {
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

	s.setCached(proxy)

	usage, err := s.repo.Usage(ctx, proxy.ID)
	if err != nil {
		return err
	}

	s.publishEventProxy(ctx, events.ProxyUpdated, proxy.ID, usage)

	return nil
}

func (s *Service) FindByID(ctx context.Context, id int64) (*domain.Proxy, error) {
	if proxy, ok := s.cached(id); ok {
		return proxy, nil
	}

	return s.repo.FindByID(ctx, id)
}

func (s *Service) cached(id int64) (*domain.Proxy, bool) {
	s.m.RLock()
	defer s.m.RUnlock()

	proxy, ok := s.cache[id]
	return proxy, ok
}

func (s *Service) setCached(proxy *domain.Proxy) {
	s.m.Lock()
	s.cache[proxy.ID] = proxy
	s.m.Unlock()
}

func (s *Service) evict(id int64) {
	s.m.Lock()
	delete(s.cache, id)
	s.m.Unlock()
}

func (s *Service) List(ctx context.Context) ([]domain.Proxy, error) {
	return s.repo.List(ctx)
}

func (s *Service) Usage(ctx context.Context, id int64) (*domain.ProxyUsage, error) {
	return s.repo.Usage(ctx, id)
}

func (s *Service) ToggleEnabled(ctx context.Context, id int64, enabled bool) error {
	if err := s.repo.ToggleEnabled(ctx, id, enabled); err != nil {
		return err
	}

	// consumers hold the cached pointer, so evict instead of mutating it in place
	s.evict(id)

	usage, err := s.repo.Usage(ctx, id)
	if err != nil {
		return err
	}

	s.publishEventProxy(ctx, events.ProxyUpdated, id, usage)

	return nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	usage, err := s.repo.Usage(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	s.evict(id)

	s.publishEventProxy(ctx, events.ProxyDeleted, id, usage)

	return nil
}

func (s *Service) publishEventProxy(ctx context.Context, eventType events.EventType, id int64, usage *domain.ProxyUsage) {
	s.eventBus.EmitProxy(ctx, events.ProxyChangeEvent{
		Type:    eventType,
		ProxyID: id,
		Usage:   usage,
	})
}

func (s *Service) Test(ctx context.Context, proxy *domain.Proxy) error {
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

	s.log.Debug().Str("proxy_addr", proxy.Addr).Msg("proxy test ok")

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
