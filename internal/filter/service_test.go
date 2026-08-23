// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package filter

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type indexerSvcStub struct {
	indexers  []domain.Indexer
	connected []domain.Indexer
}

func (s *indexerSvcStub) FindByFilterID(_ context.Context, _ int) ([]domain.Indexer, error) {
	return s.connected, nil
}

func (s *indexerSvcStub) List(_ context.Context) ([]domain.Indexer, error) {
	return s.indexers, nil
}

// filterRepoStub answers the existence check; every other repo method panics
// through the embedded nil interface, so reaching persistence fails the test.
type filterRepoStub struct {
	filterRepo
	filter  *domain.Filter
	updated *domain.Filter
}

func (s *filterRepoStub) FindByID(_ context.Context, filterID int) (*domain.Filter, error) {
	if s.filter == nil || s.filter.ID != filterID {
		return nil, domain.ErrRecordNotFound
	}
	return s.filter, nil
}

func (s *filterRepoStub) Update(_ context.Context, filter *domain.Filter) error {
	s.updated = filter
	return nil
}

func (s *filterRepoStub) StoreIndexerConnections(context.Context, int, []domain.Indexer) error {
	return nil
}

func (s *filterRepoStub) StoreFilterExternal(context.Context, int, []domain.FilterExternal) error {
	return nil
}

type actionSvcStub struct {
	actionService
}

func (s *actionSvcStub) StoreFilterActions(_ context.Context, _ int64, actions []*domain.Action) ([]*domain.Action, error) {
	return actions, nil
}

type notificationSvcStub struct {
	notificationService
	notifications       []domain.Notification
	storedFilterID      int
	storedNotifications []domain.FilterNotification
}

func (s *notificationSvcStub) Find(_ context.Context, _ domain.NotificationQueryParams) ([]domain.Notification, int, error) {
	return s.notifications, len(s.notifications), nil
}

func (s *notificationSvcStub) StoreFilterNotifications(_ context.Context, filterID int, notifications []domain.FilterNotification) error {
	s.storedFilterID = filterID
	s.storedNotifications = notifications
	return nil
}

func TestService_validateIndexers(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}, {ID: 2}}},
	}

	t.Run("accepts existing indexers", func(t *testing.T) {
		assert.NoError(t, svc.validateIndexers(t.Context(), 1, []domain.Indexer{{ID: 1}, {ID: 2}}))
	})

	t.Run("accepts empty selection", func(t *testing.T) {
		assert.NoError(t, svc.validateIndexers(t.Context(), 1, nil))
	})

	t.Run("rejects unknown indexer", func(t *testing.T) {
		err := svc.validateIndexers(t.Context(), 1, []domain.Indexer{{ID: 1}, {ID: 99}})
		assert.ErrorIs(t, err, domain.ErrIndexerNotFound)
	})

	t.Run("rejects newly added archived indexer", func(t *testing.T) {
		svc.indexerSvc = &indexerSvcStub{indexers: []domain.Indexer{{ID: 1, Archived: true}}}
		err := svc.validateIndexers(t.Context(), 1, []domain.Indexer{{ID: 1, Archived: false}})
		assert.ErrorIs(t, err, domain.ErrIndexerArchived)
	})

	t.Run("rejects archived indexer on new filter", func(t *testing.T) {
		svc.indexerSvc = &indexerSvcStub{
			indexers:  []domain.Indexer{{ID: 1, Archived: true}},
			connected: []domain.Indexer{{ID: 1, Archived: true}},
		}
		err := svc.validateIndexers(t.Context(), 0, []domain.Indexer{{ID: 1}})
		assert.ErrorIs(t, err, domain.ErrIndexerArchived)
	})

	t.Run("keeps archived indexer already connected to the filter", func(t *testing.T) {
		svc.indexerSvc = &indexerSvcStub{
			indexers:  []domain.Indexer{{ID: 1, Archived: true}, {ID: 2}},
			connected: []domain.Indexer{{ID: 1, Archived: true}},
		}
		assert.NoError(t, svc.validateIndexers(t.Context(), 1, []domain.Indexer{{ID: 1}, {ID: 2}}))
	})
}

func TestService_Update_ValidatesIndexersBeforePersisting(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		repo:       &filterRepoStub{filter: &domain.Filter{ID: 1}},
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}}},
	}

	err := svc.Update(t.Context(), &domain.Filter{
		ID:       1,
		Name:     "filter",
		Indexers: []domain.Indexer{{ID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrIndexerNotFound)
}

func TestService_Update_MissingFilterWinsOverUnknownIndexer(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		repo:       &filterRepoStub{},
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}}},
	}

	err := svc.Update(t.Context(), &domain.Filter{
		ID:       99,
		Name:     "filter",
		Indexers: []domain.Indexer{{ID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrRecordNotFound)
}

func TestService_UpdatePartial_ValidatesIndexersBeforePersisting(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		repo:       &filterRepoStub{filter: &domain.Filter{ID: 1}},
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}}},
	}

	err := svc.UpdatePartial(t.Context(), domain.FilterUpdate{
		ID:       1,
		Indexers: []domain.Indexer{{ID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrIndexerNotFound)
}

func TestService_UpdatePartial_MissingFilterWinsOverUnknownIndexer(t *testing.T) {
	svc := &Service{
		log:        zerolog.Nop(),
		repo:       &filterRepoStub{},
		indexerSvc: &indexerSvcStub{indexers: []domain.Indexer{{ID: 1}}},
	}

	err := svc.UpdatePartial(t.Context(), domain.FilterUpdate{
		ID:       99,
		Indexers: []domain.Indexer{{ID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrRecordNotFound)
}

func TestService_Update_ValidatesNotificationsBeforePersisting(t *testing.T) {
	svc := &Service{
		log:             zerolog.Nop(),
		repo:            &filterRepoStub{filter: &domain.Filter{ID: 1}},
		indexerSvc:      &indexerSvcStub{},
		notificationSvc: &notificationSvcStub{notifications: []domain.Notification{{ID: 1}}},
	}

	err := svc.Update(t.Context(), &domain.Filter{
		ID:            1,
		Name:          "filter",
		Notifications: []domain.FilterNotification{{NotificationID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
}

func TestService_Update_StoresNotifications(t *testing.T) {
	notificationSvc := &notificationSvcStub{notifications: []domain.Notification{{ID: 1}}}
	repo := &filterRepoStub{filter: &domain.Filter{ID: 7}}
	svc := &Service{
		log:             zerolog.Nop(),
		repo:            repo,
		actionService:   &actionSvcStub{},
		indexerSvc:      &indexerSvcStub{},
		notificationSvc: notificationSvc,
	}
	notifications := []domain.FilterNotification{{NotificationID: 1, Events: []string{"PUSH_APPROVED"}}}
	filter := &domain.Filter{ID: 7, Name: "filter", Notifications: notifications}

	err := svc.Update(t.Context(), filter)

	assert.NoError(t, err)
	assert.Same(t, filter, repo.updated)
	assert.Equal(t, 7, notificationSvc.storedFilterID)
	assert.Equal(t, notifications, notificationSvc.storedNotifications)
}

func TestService_UpdatePartial_ValidatesNotificationsBeforePersisting(t *testing.T) {
	svc := &Service{
		log:             zerolog.Nop(),
		repo:            &filterRepoStub{filter: &domain.Filter{ID: 1}},
		indexerSvc:      &indexerSvcStub{},
		notificationSvc: &notificationSvcStub{notifications: []domain.Notification{{ID: 1}}},
	}

	err := svc.UpdatePartial(t.Context(), domain.FilterUpdate{
		ID:            1,
		Notifications: []domain.FilterNotification{{NotificationID: 99}},
	})
	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
}

func TestService_UpdateNotifications(t *testing.T) {
	notificationSvc := &notificationSvcStub{notifications: []domain.Notification{{ID: 1}}}
	svc := &Service{
		log:             zerolog.Nop(),
		repo:            &filterRepoStub{filter: &domain.Filter{ID: 7}},
		notificationSvc: notificationSvc,
	}
	notifications := []domain.FilterNotification{{NotificationID: 1, Events: []string{"PUSH_APPROVED"}}}

	err := svc.UpdateNotifications(t.Context(), 7, notifications)

	assert.NoError(t, err)
	assert.Equal(t, 7, notificationSvc.storedFilterID)
	assert.Equal(t, notifications, notificationSvc.storedNotifications)
}

func TestService_UpdateNotifications_ValidatesBeforePersisting(t *testing.T) {
	notificationSvc := &notificationSvcStub{notifications: []domain.Notification{{ID: 1}}}
	svc := &Service{
		log:             zerolog.Nop(),
		repo:            &filterRepoStub{filter: &domain.Filter{ID: 7}},
		notificationSvc: notificationSvc,
	}

	err := svc.UpdateNotifications(t.Context(), 7, []domain.FilterNotification{{NotificationID: 99}})

	assert.ErrorIs(t, err, domain.ErrNotificationNotFound)
	assert.Zero(t, notificationSvc.storedFilterID)
}
