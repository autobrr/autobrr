// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build irc_integration_test

package irc

import (
	"context"
	"io"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/test/irc/ircd"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// countingProxy forwards TCP to the test ircd and counts the connections that
// are still open, so a test can see a connection the service no longer tracks.
type countingProxy struct {
	ln   net.Listener
	live atomic.Int64

	m     sync.Mutex
	conns []net.Conn
}

func newCountingProxy(t *testing.T, target string) *countingProxy {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	p := &countingProxy{ln: ln}
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}

			p.live.Add(1)
			p.track(c)

			go func() {
				defer p.live.Add(-1)

				u, err := net.Dial("tcp", target)
				if err != nil {
					c.Close()
					return
				}
				p.track(u)

				var wg sync.WaitGroup
				wg.Go(func() { io.Copy(u, c); u.Close() })
				wg.Go(func() { io.Copy(c, u); c.Close() })
				wg.Wait()
			}()
		}
	}()
	t.Cleanup(p.close)

	return p
}

func (p *countingProxy) track(c net.Conn) {
	p.m.Lock()
	p.conns = append(p.conns, c)
	p.m.Unlock()
}

func (p *countingProxy) close() {
	p.ln.Close()

	p.m.Lock()
	defer p.m.Unlock()
	for _, c := range p.conns {
		c.Close()
	}
}

func (p *countingProxy) port() int {
	_, port, _ := net.SplitHostPort(p.ln.Addr().String())
	n, _ := strconv.Atoi(port)
	return n
}

func (p *countingProxy) waitLive(t *testing.T, want int64) {
	t.Helper()

	assert.Eventually(t, func() bool { return p.live.Load() == want }, 5*time.Second, 20*time.Millisecond, "live connections = %d, want %d", p.live.Load(), want)
}

// staysLive fails if the live count moves off want within d: a ghost shows up
// as a connection that opens after the service considers the network stopped.
func (p *countingProxy) staysLive(t *testing.T, want int64, d time.Duration) {
	t.Helper()

	assert.Never(t, func() bool { return p.live.Load() != want }, d, 20*time.Millisecond, "live connections moved off %d", want)
}

type channelRepo struct {
	*stubIrcRepo
	channels []domain.IrcChannel
}

func (r channelRepo) ListChannels(int64) ([]domain.IrcChannel, error) { return r.channels, nil }

func lifecycleNetwork(port int, nick string) domain.IrcNetwork {
	return domain.IrcNetwork{ID: 9, Name: "lifecycle", Enabled: true, Server: "127.0.0.1", Port: port, Nick: nick}
}

func newLifecycleService(repo ircRepo) *Service {
	return NewService(zerolog.Nop(), noopEventBus{}, &mockSSEServer{}, repo, nil, stubIndexerService{}, stubProxyService{})
}

func TestService_DisableAfterRestartLeavesNoConnection(t *testing.T) {
	srv := ircd.New(t)
	p := newCountingProxy(t, srv.Addr())
	network := lifecycleNetwork(p.port(), "lifecycle")

	s := newLifecycleService(newStubIrcRepo(network))
	require.NoError(t, s.UpdateNetwork(t.Context(), new(network)))
	p.waitLive(t, 1)

	changed := network
	changed.BotMode = true
	require.NoError(t, s.UpdateNetwork(t.Context(), &changed))

	disabled := changed
	disabled.Enabled = false
	require.NoError(t, s.UpdateNetwork(t.Context(), &disabled))

	_, found := s.networkHandlers.Get(network.ID)
	require.False(t, found)

	p.waitLive(t, 0)
	p.staysLive(t, 0, restartQuitTimeout+time.Second)
}

func TestService_StopHandlersDuringRestart(t *testing.T) {
	srv := ircd.New(t)
	p := newCountingProxy(t, srv.Addr())
	network := lifecycleNetwork(p.port(), "lifecycle")

	s := newLifecycleService(newStubIrcRepo(network))
	s.StartHandlers()
	p.waitLive(t, 1)

	changed := network
	changed.BotMode = true

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = s.UpdateNetwork(t.Context(), &changed)
	})

	time.Sleep(200 * time.Millisecond)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	s.StopHandlers(ctx)
	wg.Wait()

	p.waitLive(t, 0)
	p.staysLive(t, 0, restartQuitTimeout+time.Second)
}

func TestService_ConcurrentEnablesStartOneHandler(t *testing.T) {
	for i := range 10 {
		srv := ircd.New(t)
		p := newCountingProxy(t, srv.Addr())
		network := lifecycleNetwork(p.port(), "lifecycle"+strconv.Itoa(i))

		s := newLifecycleService(newStubIrcRepo(network))

		var wg sync.WaitGroup
		start := make(chan struct{})
		for range 2 {
			wg.Go(func() {
				<-start
				_ = s.UpdateNetwork(t.Context(), new(network))
			})
		}
		close(start)
		wg.Wait()

		p.waitLive(t, 1)

		require.NoError(t, s.StopAndRemoveNetwork(network.ID))
		p.waitLive(t, 0)
	}
}

func TestService_EditRestartsFatallyStoppedNetwork(t *testing.T) {
	srv := ircd.New(t)
	p := newCountingProxy(t, srv.Addr())
	network := lifecycleNetwork(p.port(), "lifecycle")

	s := newLifecycleService(newStubIrcRepo(network))
	require.NoError(t, s.UpdateNetwork(t.Context(), new(network)))
	p.waitLive(t, 1)

	h, found := s.networkHandlers.Get(network.ID)
	require.True(t, found)

	// what the TLS certificate, ban and auth failure paths do
	h.addConnectError("TLS certificate verification failed: x509: expired")
	h.stateMachine.OnError("TLS certificate verification failed")
	h.Stop()
	p.waitLive(t, 0)

	fixed := network
	fixed.TLSSkipVerify = true
	require.NoError(t, s.UpdateNetwork(t.Context(), &fixed))

	p.waitLive(t, 1)
	assert.False(t, h.Stopped())
}

func TestService_EditThenRestartKeepsChannelsJoined(t *testing.T) {
	srv := ircd.New(t)
	srv.AddChannel("#announce")

	channels := []domain.IrcChannel{{ID: 1, Name: "#announce", Enabled: true}}
	network := domain.IrcNetwork{ID: 9, Name: "lifecycle", Enabled: true, Server: srv.Host(), Port: srv.Port(), Nick: "lifecycle", Channels: channels}

	s := newLifecycleService(channelRepo{stubIrcRepo: newStubIrcRepo(network), channels: channels})
	s.StartHandlers()
	t.Cleanup(func() { s.StopHandlers(context.Background()) })

	h, found := s.networkHandlers.Get(network.ID)
	require.True(t, found)

	monitoring := func() bool {
		channel, ok := h.channels.Get("#announce")
		return ok && channel.Snapshot().Monitoring
	}
	require.Eventually(t, monitoring, 10*time.Second, 50*time.Millisecond)

	changed := network
	changed.BotMode = true

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = s.UpdateNetwork(t.Context(), &changed)
	})

	time.Sleep(100 * time.Millisecond)
	require.NoError(t, s.RestartNetwork(t.Context(), network.ID))
	wg.Wait()

	require.Eventually(t, monitoring, 10*time.Second, 50*time.Millisecond, "channel not monitored after an edit and a restart")
	assert.Never(t, func() bool { return !monitoring() }, restartQuitTimeout+time.Second, 50*time.Millisecond, "channel dropped out of monitoring after the restarts settled")
}

// staleReadRepo persists UpdateNetwork and can park one GetNetworkByID after it
// has read the row, like a request descheduled between its read and its restart.
type staleReadRepo struct {
	*stubIrcRepo
	park    atomic.Bool
	parked  chan struct{}
	release chan struct{}
}

func (r *staleReadRepo) UpdateNetwork(_ context.Context, network *domain.IrcNetwork) error {
	r.m.Lock()
	defer r.m.Unlock()

	r.network = *network
	return nil
}

func (r *staleReadRepo) GetNetworkByID(context.Context, int64) (*domain.IrcNetwork, error) {
	row := r.row()
	if r.park.CompareAndSwap(true, false) {
		close(r.parked)
		<-r.release
	}

	return row, nil
}

func TestService_RestartDoesNotReviveDisabledNetwork(t *testing.T) {
	srv := ircd.New(t)
	p := newCountingProxy(t, srv.Addr())
	network := lifecycleNetwork(p.port(), "lifecycle")

	repo := &staleReadRepo{stubIrcRepo: newStubIrcRepo(network), parked: make(chan struct{}), release: make(chan struct{})}
	release := sync.OnceFunc(func() { close(repo.release) })
	defer release()

	s := newLifecycleService(repo)
	require.NoError(t, s.UpdateNetwork(t.Context(), new(network)))
	p.waitLive(t, 1)

	repo.park.Store(true)

	var wg sync.WaitGroup
	wg.Go(func() {
		_ = s.RestartNetwork(t.Context(), network.ID)
	})
	<-repo.parked

	disabled := network
	disabled.Enabled = false
	wg.Go(func() {
		_ = s.UpdateNetwork(t.Context(), &disabled)
	})

	time.Sleep(200 * time.Millisecond)
	release()
	wg.Wait()

	_, found := s.networkHandlers.Get(network.ID)
	assert.False(t, found, "restart revived a network disabled while it was loading the config")

	p.waitLive(t, 0)
	p.staysLive(t, 0, restartQuitTimeout+time.Second)
}

// stalledChannelsRepo blocks ListChannels, which startNetwork calls while it
// holds the lifecycle lock.
type stalledChannelsRepo struct {
	*stubIrcRepo
	entered chan struct{}
	release chan struct{}
}

func (r stalledChannelsRepo) ListChannels(int64) ([]domain.IrcChannel, error) {
	close(r.entered)
	<-r.release

	return nil, nil
}

func TestService_StopHandlersKeepsDeadlineWhileLifecycleBusy(t *testing.T) {
	srv := ircd.New(t)
	p := newCountingProxy(t, srv.Addr())
	network := lifecycleNetwork(p.port(), "lifecycle")

	repo := stalledChannelsRepo{stubIrcRepo: newStubIrcRepo(network), entered: make(chan struct{}), release: make(chan struct{})}
	release := sync.OnceFunc(func() { close(repo.release) })
	defer release()

	s := newLifecycleService(repo)

	var wg sync.WaitGroup
	wg.Go(s.StartHandlers)
	<-repo.entered

	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()

	stopped := make(chan struct{})
	go func() {
		s.StopHandlers(ctx)
		close(stopped)
	}()

	select {
	case <-stopped:
	case <-time.After(5 * time.Second):
		t.Fatal("StopHandlers waited past its deadline for the lifecycle lock")
	}

	release()
	wg.Wait()

	p.staysLive(t, 0, time.Second)
}
