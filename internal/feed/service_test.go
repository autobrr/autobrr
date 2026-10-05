// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package feed_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/feed"
	"github.com/autobrr/autobrr/internal/proxy"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testFeedRepo struct {
	*database.FeedRepo
	item domain.Feed
}

func (r *testFeedRepo) FindOne(context.Context, domain.FindOneParams) (*domain.Feed, error) {
	item := r.item
	return &item, nil
}

type testProxyRepo struct {
	*database.ProxyRepo
	item domain.Proxy
}

func (r *testProxyRepo) FindByID(context.Context, int64) (*domain.Proxy, error) {
	item := r.item
	return &item, nil
}

func TestServiceTestTimeout(t *testing.T) {
	t.Parallel()

	for _, feedType := range []domain.FeedType{domain.FeedTypeTorznab, domain.FeedTypeNewznab} {
		t.Run(string(feedType), func(t *testing.T) {
			for _, route := range []string{"direct", "proxy"} {
				t.Run(route, func(t *testing.T) {
					t.Parallel()

					srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.URL.Query().Get("t") == "caps" {
							_, _ = w.Write([]byte(`<caps/>`))
							return
						}
						<-r.Context().Done()
					}))
					defer srv.Close()

					f := domain.Feed{ID: 1, Type: string(feedType), URL: srv.URL, Timeout: 1}
					log := zerolog.Nop()
					bus := events.NewEventBus(log)
					proxyRepo := &testProxyRepo{item: domain.Proxy{ID: 1, Enabled: true, Type: domain.ProxyTypeHTTP, Addr: srv.URL}}
					proxyService := proxy.NewService(log, bus, proxyRepo)
					if route == "proxy" {
						f.UseProxy = true
						f.ProxyID = 1
						f.URL = "http://feed.invalid/api"
					}
					service := feed.NewService(log, bus, &testFeedRepo{item: f}, nil, nil, proxyService, nil)
					ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
					defer cancel()

					err := service.Test(ctx, &f)
					require.ErrorIs(t, err, context.DeadlineExceeded)
					assert.NoError(t, ctx.Err(), "the feed timeout must expire before the caller deadline")
				})
			}
		})
	}
}
