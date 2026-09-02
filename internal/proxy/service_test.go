// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package proxy

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubProxyRepo struct {
	proxy    *domain.Proxy
	usage    *domain.ProxyUsage
	usageErr error
}

func (r *stubProxyRepo) Store(context.Context, *domain.Proxy) error             { return nil }
func (r *stubProxyRepo) Update(context.Context, *domain.Proxy) error            { return nil }
func (r *stubProxyRepo) List(context.Context) ([]domain.Proxy, error)           { return nil, nil }
func (r *stubProxyRepo) Delete(context.Context, int64) error                    { return nil }
func (r *stubProxyRepo) FindByID(context.Context, int64) (*domain.Proxy, error) { return r.proxy, nil }
func (r *stubProxyRepo) ToggleEnabled(context.Context, int64, bool) error       { return nil }
func (r *stubProxyRepo) Usage(context.Context, int64) (*domain.ProxyUsage, error) {
	return r.usage, r.usageErr
}

type recordingProxyBus struct{ events []events.ProxyChangeEvent }

func (b *recordingProxyBus) EmitProxy(_ context.Context, event events.ProxyChangeEvent) {
	b.events = append(b.events, event)
}

func socksProxy() *domain.Proxy {
	return &domain.Proxy{ID: 1, Name: "socks", Enabled: true, Type: domain.ProxyTypeSocks5, Addr: "socks5://10.0.0.1:1080"}
}

func TestUpdate_EmitsUsage(t *testing.T) {
	t.Parallel()

	usage := &domain.ProxyUsage{IrcNetworks: []domain.ProxyUsageItem{{ID: 7, Name: "net"}}}
	bus := &recordingProxyBus{}
	s := NewService(zerolog.Nop(), bus, &stubProxyRepo{proxy: socksProxy(), usage: usage})

	require.NoError(t, s.Update(context.Background(), socksProxy()))

	require.Len(t, bus.events, 1)
	assert.Equal(t, events.ProxyUpdated, bus.events[0].Type)
	assert.Equal(t, int64(1), bus.events[0].ProxyID)
	assert.Same(t, usage, bus.events[0].Usage)
}

func TestUpdate_UsageLookupFailureDoesNotFailRequest(t *testing.T) {
	t.Parallel()

	bus := &recordingProxyBus{}
	s := NewService(zerolog.Nop(), bus, &stubProxyRepo{proxy: socksProxy(), usageErr: errors.New("db gone")})

	require.NoError(t, s.Update(context.Background(), socksProxy()))

	assert.Empty(t, bus.events, "without a snapshot there is nothing to reconcile against")
}
