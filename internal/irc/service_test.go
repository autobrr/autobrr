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
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubIrcRepo serves a single network row and records enabled toggles.
type stubIrcRepo struct {
	m       sync.Mutex
	network domain.IrcNetwork
	toggled map[int64]bool

	lastChannelID int64
	inviteErr     error
}

func newStubIrcRepo(network domain.IrcNetwork) *stubIrcRepo {
	return &stubIrcRepo{network: network, toggled: map[int64]bool{}}
}

func (r *stubIrcRepo) row() *domain.IrcNetwork {
	r.m.Lock()
	defer r.m.Unlock()

	cp := r.network
	return &cp
}

func (r *stubIrcRepo) toggledTo(id int64) (bool, bool) {
	r.m.Lock()
	defer r.m.Unlock()

	enabled, ok := r.toggled[id]
	return enabled, ok
}

func (r *stubIrcRepo) StoreNetwork(context.Context, *domain.IrcNetwork) error  { return nil }
func (r *stubIrcRepo) UpdateNetwork(context.Context, *domain.IrcNetwork) error { return nil }
func (r *stubIrcRepo) ToggleNetworkEnabled(_ context.Context, id int64, enabled bool) error {
	r.m.Lock()
	defer r.m.Unlock()

	r.toggled[id] = enabled
	r.network.Enabled = enabled
	return nil
}
func (r *stubIrcRepo) StoreChannel(_ context.Context, _ int64, channel *domain.IrcChannel) error {
	r.m.Lock()
	defer r.m.Unlock()

	if channel.ID == 0 {
		r.lastChannelID++
		channel.ID = r.lastChannelID
	}

	return nil
}
func (r *stubIrcRepo) UpdateChannel(*domain.IrcChannel) error  { return nil }
func (r *stubIrcRepo) UpdateInviteCommand(int64, string) error { return r.inviteErr }
func (r *stubIrcRepo) StoreNetworkChannels(context.Context, int64, []domain.IrcChannel) error {
	return nil
}
func (r *stubIrcRepo) CheckExistingNetwork(context.Context, *domain.IrcNetwork) (*domain.IrcNetwork, error) {
	return r.row(), nil
}
func (r *stubIrcRepo) FindActiveNetworks(context.Context) ([]domain.IrcNetwork, error) {
	return []domain.IrcNetwork{*r.row()}, nil
}
func (r *stubIrcRepo) ListNetworks(context.Context) ([]domain.IrcNetwork, error) { return nil, nil }
func (r *stubIrcRepo) ListChannels(int64) ([]domain.IrcChannel, error)           { return nil, nil }
func (r *stubIrcRepo) GetNetworkByID(context.Context, int64) (*domain.IrcNetwork, error) {
	return r.row(), nil
}
func (r *stubIrcRepo) DeleteNetwork(context.Context, int64) error { return nil }

type stubIndexerService struct{}

func (stubIndexerService) GetIndexersByIRCNetwork(string) []*domain.IndexerDefinition { return nil }

type stubDefinitionsIndexerService struct {
	definitions []*domain.IndexerDefinition
}

func (s stubDefinitionsIndexerService) GetIndexersByIRCNetwork(string) []*domain.IndexerDefinition {
	return s.definitions
}

type stubProxyService struct {
	proxy *domain.Proxy
	err   error
}

func (s stubProxyService) FindByID(context.Context, int64) (*domain.Proxy, error) {
	return s.proxy, s.err
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

func TestOnProxyChanged_DisabledProxyDisablesNetwork(t *testing.T) {
	t.Parallel()

	socks := &domain.Proxy{ID: 1, Name: "socks", Enabled: true, Type: domain.ProxyTypeSocks5, Addr: "socks5://10.0.0.1:1080"}
	disabled := *socks
	disabled.Enabled = false

	repo := newStubIrcRepo(proxiedNetwork())
	s := NewService(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, repo, nil, stubIndexerService{}, stubProxyService{proxy: &disabled})
	h := runningHandler(s, proxiedNetwork(), socks)

	err := s.onProxyChanged(context.Background(), events.ProxyChangeEvent{
		Type:    events.ProxyUpdated,
		ProxyID: 1,
		Usage:   &domain.ProxyUsage{IrcNetworks: []domain.ProxyUsageItem{{ID: 7, Name: "net"}}},
	})
	require.NoError(t, err)

	assert.True(t, h.Stopped())

	_, found := s.networkHandlers.Get(int64(7))
	assert.False(t, found, "handler must be removed, not left to reconnect direct")

	enabled, ok := repo.toggledTo(7)
	require.True(t, ok, "network must be persisted as disabled")
	assert.False(t, enabled)
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

func TestCheckIfNetworkRestartNeeded_EquivalentChannelNamesStayDistinct(t *testing.T) {
	t.Parallel()

	network := domain.IrcNetwork{ID: 7, Name: "net", Server: "irc.example.test", Port: 6697, Nick: "bot", Enabled: true}
	network.Channels = []domain.IrcChannel{{ID: 1, Name: "#chan[", Enabled: true, Password: "one"}}

	s := NewService(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, newStubIrcRepo(network), nil, stubIndexerService{}, stubProxyService{})
	h := runningHandler(s, network, nil)

	updated := network
	updated.Channels = []domain.IrcChannel{
		{ID: 1, Name: "#chan[", Enabled: true, Password: "one"},
		{ID: 2, Name: "#chan{", Enabled: true, Password: "two"},
	}
	require.NoError(t, s.checkIfNetworkRestartNeeded(&updated))

	first, found := h.channels.Get("#chan[")
	require.True(t, found, "existing channel lost its runtime entry")
	second, found := h.channels.Get("#chan{")
	require.True(t, found, "added channel has no runtime entry")

	assert.Equal(t, int64(1), first.Snapshot().ID)
	assert.Equal(t, "one", first.GetPassword())
	assert.Equal(t, int64(2), second.Snapshot().ID)
	assert.Equal(t, "two", second.GetPassword())
	assert.NotEqual(t, ChannelStateIdle, second.StateMachine().CurrentState(), "the added channel was never started")
}

// Adding a sister site merges its channel and invite command into the existing
// network. Before the fix the handler kept its old channel set, so the new
// channel was stored but never joined until autobrr restarted.
func TestStoreNetwork_ExistingNetworkRegistersNewChannel(t *testing.T) {
	t.Parallel()

	existing := domain.IrcNetwork{ID: 7, Name: "DigitalCore", Server: "irc.example.test", Port: 7000, Nick: "bot", Enabled: true, InviteCommand: "ENDOR !invite user dckey"}
	definitions := []*domain.IndexerDefinition{
		defWithInvite("digitalcore", "irc.example.test", "#announce", "ENDOR", "ENDOR"),
		defWithInvite("dirtybytes", "irc.example.test", "#dirtybytes-announce", "DirtyBytes", "DirtyBytes"),
	}

	s := NewService(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, newStubIrcRepo(existing), nil, stubDefinitionsIndexerService{definitions: definitions}, stubProxyService{})

	// a handler parked by a restart in flight, which the update must not bypass
	h := NewHandler(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, existing, definitions[:1], nil)
	s.networkHandlers.Set(existing.ID, h)

	err := s.StoreNetwork(t.Context(), &domain.IrcNetwork{
		Server:        "irc.example.test",
		Port:          7000,
		Nick:          "bot",
		InviteCommand: "DirtyBytes !invite user dbkey",
		Channels:      []domain.IrcChannel{{Name: "#dirtybytes-announce", Enabled: true}},
	})
	require.NoError(t, err)

	ch, found := h.channels.Get("#dirtybytes-announce")
	require.True(t, found, "the merged channel must be registered on the running handler")
	assert.NotNil(t, ch.announceProcessor)
	assert.Equal(t, "DirtyBytes !invite user dbkey", ch.InviteCommand())
	assert.Equal(t, int64(1), ch.Snapshot().ID, "the handler must carry the stored channel id")

	_, cached := s.networkCache.Get(0)
	assert.False(t, cached, "the merged network must be cached under the existing network id")

	cachedNetwork, cached := s.networkCache.Get(existing.ID)
	require.True(t, cached)
	require.Len(t, cachedNetwork.Channels, 1)
	assert.Equal(t, int64(1), cachedNetwork.Channels[0].ID, "the cache must carry the stored channel id")
}

// A failed write must not leave the merged network's unsaved invite command in
// the cache, which serves the network list.
func TestStoreNetwork_FailedInviteCommandUpdateKeepsCache(t *testing.T) {
	t.Parallel()

	existing := domain.IrcNetwork{ID: 7, Name: "DigitalCore", Server: "irc.example.test", Port: 7000, Nick: "bot", Enabled: true, InviteCommand: "ENDOR !invite user dckey"}

	repo := newStubIrcRepo(existing)
	repo.inviteErr = errors.New("database is locked")

	s := NewService(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, repo, nil, stubIndexerService{}, stubProxyService{})

	cachedBefore := existing
	s.networkCache.Set(existing.ID, &cachedBefore)

	err := s.StoreNetwork(t.Context(), &domain.IrcNetwork{
		Server:        "irc.example.test",
		Port:          7000,
		Nick:          "bot",
		InviteCommand: "DirtyBytes !invite user dbkey",
		Channels:      []domain.IrcChannel{{Name: "#dirtybytes-announce", Enabled: true}},
	})
	require.Error(t, err)

	cached, found := s.networkCache.Get(existing.ID)
	require.True(t, found)
	assert.Same(t, &cachedBefore, cached, "a failed store must keep the previous cache entry")
	assert.Equal(t, "ENDOR !invite user dckey", cached.InviteCommand)
	assert.Empty(t, cached.Channels)
}
