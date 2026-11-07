// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

type updateIDFeedService struct {
	feedService
	id *int64
}

func (s updateIDFeedService) Update(_ context.Context, feed *domain.Feed) error {
	*s.id = int64(feed.ID)
	return nil
}

type updateIDFilterService struct {
	filterService
	id *int64
}

func (s updateIDFilterService) Update(_ context.Context, filter *domain.Filter) error {
	*s.id = int64(filter.ID)
	return nil
}

type updateIDIndexerService struct {
	indexerService
	id *int64
}

func (s updateIDIndexerService) Update(_ context.Context, indexer *domain.Indexer) error {
	*s.id = indexer.ID
	return nil
}

type updateIDListService struct {
	listService
	id *int64
}

func (s updateIDListService) Update(_ context.Context, list *domain.List) error {
	*s.id = list.ID
	return nil
}

type updateIDProxyService struct {
	proxyService
	id *int64
}

func (s updateIDProxyService) Update(_ context.Context, proxy *domain.Proxy) error {
	*s.id = proxy.ID
	return nil
}

type updateIDIrcService struct {
	ircService
	id *int64
}

func (s updateIDIrcService) UpdateNetwork(_ context.Context, network *domain.IrcNetwork) error {
	*s.id = network.ID
	return nil
}

type updateIDNotificationService struct {
	notificationService
	id *int64
}

func (s updateIDNotificationService) Update(_ context.Context, notification *domain.Notification) error {
	*s.id = int64(notification.ID)
	return nil
}

type updateIDReleaseService struct {
	releaseService
	id *int64
}

func (s updateIDReleaseService) UpdateCleanupJob(_ context.Context, job *domain.ReleaseCleanupJob) error {
	*s.id = int64(job.ID)
	return nil
}

func TestUpdateHandlers_UsePathID(t *testing.T) {
	tests := []struct {
		name   string
		routes func(id *int64) func(chi.Router)
		path   string
	}{
		{
			name:   "feed",
			routes: func(id *int64) func(chi.Router) { return newFeedHandler(encoder{}, updateIDFeedService{id: id}).Routes },
			path:   "/1",
		},
		{
			name: "filter",
			routes: func(id *int64) func(chi.Router) {
				return newFilterHandler(encoder{}, updateIDFilterService{id: id}).Routes
			},
			path: "/1",
		},
		{
			name: "indexer",
			routes: func(id *int64) func(chi.Router) {
				return newIndexerHandler(encoder{}, updateIDIndexerService{id: id}, nil).Routes
			},
			path: "/1",
		},
		{
			name:   "list",
			routes: func(id *int64) func(chi.Router) { return newListHandler(encoder{}, updateIDListService{id: id}).Routes },
			path:   "/1",
		},
		{
			name: "proxy",
			routes: func(id *int64) func(chi.Router) {
				return newProxyHandler(encoder{}, updateIDProxyService{id: id}).Routes
			},
			path: "/1",
		},
		{
			name: "irc network",
			routes: func(id *int64) func(chi.Router) {
				return newIrcHandler(encoder{}, nil, updateIDIrcService{id: id}).Routes
			},
			path: "/network/1",
		},
		{
			name: "notification",
			routes: func(id *int64) func(chi.Router) {
				return newNotificationHandler(encoder{}, updateIDNotificationService{id: id}).Routes
			},
			path: "/1",
		},
		{
			name: "release cleanup job",
			routes: func(id *int64) func(chi.Router) {
				return newReleaseHandler(encoder{}, updateIDReleaseService{id: id}).Routes
			},
			path: "/cleanup-jobs/1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int64

			r := chi.NewRouter()
			r.Use(asCaller(testUser))
			r.Route("/", tt.routes(&got))

			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(http.MethodPut, tt.path, strings.NewReader(`{"id": 2}`)))

			assert.Less(t, rec.Code, http.StatusMultipleChoices, rec.Body.String())
			assert.Equal(t, int64(1), got)
		})
	}
}

func TestUpdateHandlers_RejectNullBody(t *testing.T) {
	var got int64

	r := chi.NewRouter()
	r.Use(asCaller(testUser))
	r.Route("/feeds", newFeedHandler(encoder{}, updateIDFeedService{id: &got}).Routes)
	r.Route("/indexer", newIndexerHandler(encoder{}, updateIDIndexerService{id: &got}, nil).Routes)
	r.Route("/proxy", newProxyHandler(encoder{}, updateIDProxyService{id: &got}).Routes)
	r.Route("/irc", newIrcHandler(encoder{}, nil, updateIDIrcService{id: &got}).Routes)
	r.Route("/lists", newListHandler(encoder{}, updateIDListService{id: &got}).Routes)
	r.Route("/notification", newNotificationHandler(encoder{}, updateIDNotificationService{id: &got}).Routes)
	r.Route("/release", newReleaseHandler(encoder{}, updateIDReleaseService{id: &got}).Routes)
	r.Route("/downloaders", newDownloaderHandler(encoder{}, nil).Routes)

	tests := []struct {
		method string
		path   string
	}{
		{method: http.MethodPut, path: "/feeds/1"},
		{method: http.MethodPut, path: "/indexer/1"},
		{method: http.MethodPut, path: "/proxy/1"},
		{method: http.MethodPut, path: "/irc/network/1"},
		{method: http.MethodPut, path: "/lists/1"},
		{method: http.MethodPut, path: "/notification/1"},
		{method: http.MethodPut, path: "/release/cleanup-jobs/1"},
		{method: http.MethodPost, path: "/downloaders/"},
		{method: http.MethodPost, path: "/downloaders/test"},
		{method: http.MethodPut, path: "/downloaders/"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, httptest.NewRequest(tt.method, tt.path, strings.NewReader("null")))

			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
		})
	}

	assert.Zero(t, got, "a null body reached a service")
}
