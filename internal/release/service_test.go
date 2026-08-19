// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/events"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.com/tozd/go/errors"
)

// Mock objects
type mockFilterService struct {
	filterService
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
	actionService
	mock.Mock
}

func (m *mockActionService) FindByFilterID(ctx context.Context, filterID int, active *bool, isTest bool) ([]*domain.Action, error) {
	args := m.Called(ctx, filterID, active, isTest)
	return args.Get(0).([]*domain.Action), args.Error(1)
}

type mockReleaseRepo struct {
	releaseRepo
	mock.Mock
}

func (m *mockReleaseRepo) Store(ctx context.Context, release *domain.Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func (m *mockReleaseRepo) Update(ctx context.Context, release *domain.Release) error {
	args := m.Called(ctx, release)
	return args.Error(0)
}

func TestService_Process_PublishesEvent(t *testing.T) {
	log := logger.Mock()

	bus := events.NewEventBus(log)

	// Track if event was published
	published := false
	bus.OnReleaseNew(func(ctx context.Context, event events.ReleaseEvent) errors.E {
		published = true
		return nil
	})

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
	repo.On("Update", mock.Anything, mock.Anything).Return(nil)

	s := &Service{
		log:        log.With().Logger(),
		eventBus:   bus,
		filterSvc:  filterSvc,
		actionSvc:  actionSvc,
		repo:       repo,
		indexerSvc: nil, // Not used in this path
	}

	release := &domain.Release{
		TorrentName: "Test.Release-Group",
		Indexer:     domain.IndexerMinimal{Name: "MockIndexer", Identifier: "mock"},
	}

	s.Process(t.Context(), release)

	// s.bus.Publish is synchronous in EventBus when using standard Publish
	assert.True(t, published, "RELEASE_NEW event should have been published")
}

func TestCleanupJobKey_ToString(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		expected string
	}{
		{
			name:     "ID 1",
			id:       1,
			expected: "release-cleanup-1",
		},
		{
			name:     "ID 42",
			id:       42,
			expected: "release-cleanup-42",
		},
		{
			name:     "ID 999",
			id:       999,
			expected: "release-cleanup-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cleanupJobKey{id: tt.id}
			assert.Equal(t, tt.expected, key.ToString())
		})
	}
}
