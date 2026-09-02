// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package irc

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubIrcRepo serves a single network row.
type stubIrcRepo struct {
	m       sync.Mutex
	network domain.IrcNetwork
}

func newStubIrcRepo(network domain.IrcNetwork) *stubIrcRepo {
	return &stubIrcRepo{network: network}
}

func (r *stubIrcRepo) row() *domain.IrcNetwork {
	r.m.Lock()
	defer r.m.Unlock()

	cp := r.network
	return &cp
}

func (r *stubIrcRepo) StoreNetwork(context.Context, *domain.IrcNetwork) error        { return nil }
func (r *stubIrcRepo) UpdateNetwork(context.Context, *domain.IrcNetwork) error       { return nil }
func (r *stubIrcRepo) StoreChannel(context.Context, int64, *domain.IrcChannel) error { return nil }
func (r *stubIrcRepo) UpdateChannel(*domain.IrcChannel) error                        { return nil }
func (r *stubIrcRepo) UpdateInviteCommand(int64, string) error                       { return nil }
func (r *stubIrcRepo) StoreNetworkChannels(context.Context, int64, []domain.IrcChannel) error {
	return nil
}
func (r *stubIrcRepo) CheckExistingNetwork(context.Context, *domain.IrcNetwork) (*domain.IrcNetwork, error) {
	return r.row(), nil
}
func (r *stubIrcRepo) FindActiveNetworks(context.Context) ([]domain.IrcNetwork, error) {
	return nil, nil
}
func (r *stubIrcRepo) ListNetworks(context.Context) ([]domain.IrcNetwork, error) { return nil, nil }
func (r *stubIrcRepo) ListChannels(int64) ([]domain.IrcChannel, error)           { return nil, nil }
func (r *stubIrcRepo) GetNetworkByID(context.Context, int64) (*domain.IrcNetwork, error) {
	return r.row(), nil
}
func (r *stubIrcRepo) DeleteNetwork(context.Context, int64) error { return nil }

type stubIndexerService struct{}

func (stubIndexerService) GetIndexersByIRCNetwork(string) []*domain.IndexerDefinition { return nil }

type stubProxyService struct{ proxy *domain.Proxy }

func (s stubProxyService) FindByID(context.Context, int64) (*domain.Proxy, error) {
	return s.proxy, nil
}

func proxiedNetwork() domain.IrcNetwork {
	return domain.IrcNetwork{ID: 7, Name: "net", Server: "irc.example.test", Port: 6697, Nick: "bot", Enabled: true, UseProxy: true, ProxyId: 1}
}

// runningHandler registers a handler that looks connected through the given proxy, the way
// StartHandlers leaves it.
func runningHandler(s *Service, network domain.IrcNetwork, p *domain.Proxy) *Handler {
	network.Proxy = p

	h := NewHandler(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, network, nil, nil)
	h.m.Lock()
	h.clientState = ircLive
	h.m.Unlock()

	s.networkHandlers.Set(network.ID, h)

	return h
}

func TestStoreNetwork_ExistingNetworkKeepsProxy(t *testing.T) {
	t.Parallel()

	socks := &domain.Proxy{ID: 1, Name: "socks", Enabled: true, Type: domain.ProxyTypeSocks5, Addr: "socks5://10.0.0.1:1080"}
	s := NewService(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, newStubIrcRepo(proxiedNetwork()), nil, stubIndexerService{}, stubProxyService{proxy: socks})
	h := runningHandler(s, proxiedNetwork(), socks)

	// a second indexer on the same server merges into the existing network
	err := s.StoreNetwork(context.Background(), &domain.IrcNetwork{Server: "irc.example.test", Port: 6697, Nick: "bot", Enabled: true})
	require.NoError(t, err)

	assert.Same(t, socks, h.GetNetwork().Proxy)
	assert.Never(t, h.Stopped, 200*time.Millisecond, 20*time.Millisecond, "an unchanged proxy must not restart the network")
}

func TestRestartNetwork_RefusesDisabledProxy(t *testing.T) {
	t.Parallel()

	disabled := &domain.Proxy{ID: 1, Name: "socks", Enabled: false, Type: domain.ProxyTypeSocks5, Addr: "socks5://10.0.0.1:1080"}
	s := NewService(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, newStubIrcRepo(proxiedNetwork()), nil, stubIndexerService{}, stubProxyService{proxy: disabled})

	require.NoError(t, s.RestartNetwork(context.Background(), 7))

	h, found := s.networkHandlers.Get(int64(7))
	require.True(t, found)
	assert.Same(t, disabled, h.GetNetwork().Proxy, "RestartNetwork must attach the proxy before building the handler")

	assert.Eventually(t, func() bool {
		h.m.RLock()
		defer h.m.RUnlock()

		return h.clientState == ircStopped && len(h.connectionErrors) == 1 && strings.Contains(h.connectionErrors[0], "is disabled")
	}, time.Second, 10*time.Millisecond, "handler must refuse to dial and surface the reason")
}
