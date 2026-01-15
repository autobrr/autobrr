// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/action"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/filter"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/asaskevich/EventBus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock objects
type mockFilterService struct {
	filter.Service
	mock.Mock
}

func (m *mockFilterService) CheckFilter(ctx context.Context, f *domain.Filter, release *domain.Release) (bool, error) {
	args := m.Called(ctx, f, release)
	return args.Bool(0), args.Error(1)
}

func (m *mockFilterService) FindByIndexerIdentifier(ctx context.Context, indexer string) ([]*domain.Filter, error) {
	args := m.Called(ctx, indexer)
	return args.Get(0).([]*domain.Filter), args.Error(1)
}

type mockActionService struct {
	action.Service
	mock.Mock
}

func (m *mockActionService) FindByFilterID(ctx context.Context, filterID int, active *bool, isTest bool) ([]*domain.Action, error) {
	args := m.Called(ctx, filterID, active, isTest)
	return args.Get(0).([]*domain.Action), args.Error(1)
}

type mockReleaseRepo struct {
	domain.ReleaseRepo
	mock.Mock
}

func (m *mockReleaseRepo) Store(ctx context.Context, release *domain.Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func TestService_Process_PublishesEvent(t *testing.T) {
	bus := EventBus.New()

	// Track if event was published
	published := false
	bus.Subscribe(domain.EventNotificationSend, func(event *domain.NotificationEvent, payload *domain.NotificationPayload) {
		if *event == domain.NotificationEventReleaseNew {
			published = true
		}
	})

	log := logger.Mock()

	// Minimal mock for FilterSvc
	filterSvc := &mockFilterService{}
	filterSvc.On("FindByIndexerIdentifier", mock.Anything, mock.Anything).Return([]*domain.Filter{{ID: 1, Name: "Test Filter", RejectReasons: domain.NewRejectionReasons()}}, nil)
	filterSvc.On("CheckFilter", mock.Anything, mock.Anything, mock.Anything).Return(true, nil)

	// Minimal mock for ActionSvc
	actionSvc := &mockActionService{}
	actionSvc.On("FindByFilterID", mock.Anything, 1, mock.Anything, false).Return([]*domain.Action{{ID: 1, Name: "Test Action"}}, nil)

	// Minimal mock for Repo
	repo := &mockReleaseRepo{}
	repo.On("Store", mock.Anything, mock.Anything).Return(nil)

	s := &service{
		log:        log.With().Logger(),
		bus:        bus,
		filterSvc:  filterSvc,
		actionSvc:  actionSvc,
		repo:       repo,
		indexerSvc: nil, // Not used in this path
	}

	release := &domain.Release{
		TorrentName: "Test.Release-Group",
		Indexer:     domain.IndexerMinimal{Name: "MockIndexer", Identifier: "mock"},
	}

	s.Process(release)

	// s.bus.Publish is synchronous in EventBus when using standard Publish
	assert.True(t, published, "RELEASE_NEW event should have been published")
}
