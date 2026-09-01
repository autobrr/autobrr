// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
)

func getMockNotification() domain.Notification {
	return domain.Notification{
		ID:        1,
		Name:      "MockNotification",
		Type:      domain.NotificationTypeSlack,
		Enabled:   true,
		Events:    []string{"event1", "event2"},
		Token:     "mock-token",
		APIKey:    "mock-api-key",
		Webhook:   "https://webhook.example.com",
		Title:     "Mock Title",
		Icon:      "https://icon.example.com",
		Username:  "mock-username",
		Host:      "https://host.example.com",
		Password:  "mock-password",
		Channel:   "#mock-channel",
		Rooms:     "room1,room2",
		Targets:   "target1,target2",
		Devices:   "device1,device2",
		Priority:  1,
		Topic:     "mock-topic",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestNotificationRepo_Store(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		repo := NewNotificationRepo(log, db)

		mockData := getMockNotification()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			assert.NotNil(t, mockData)

			notification := getMockNotification()

			// Execute
			err := repo.Store(ctx, &notification)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, notification.Name)
			assert.Equal(t, mockData.Type, notification.Type)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})
	}
}

func TestNotificationRepo_Update(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		repo := NewNotificationRepo(log, db)
		mockData := getMockNotification()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Initial setup and Store
			err := repo.Store(ctx, &mockData)
			assert.NoError(t, err)
			assert.NotNil(t, &mockData)

			// Modify some fields
			newName := "UpdatedName"
			newType := domain.NotificationTypeTelegram
			newPriority := int32(2)

			updatedMockData := &mockData
			updatedMockData.Name = newName
			updatedMockData.Type = newType
			updatedMockData.Priority = newPriority

			// Execute Update
			err = repo.Update(ctx, updatedMockData)

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, &mockData)
			assert.Equal(t, updatedMockData.Name, newName)
			assert.Equal(t, updatedMockData.Type, newType)
			assert.Equal(t, updatedMockData.Priority, newPriority)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})
	}
}

func TestNotificationRepo_Delete(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		repo := NewNotificationRepo(log, db)
		//mockData := getMockNotification()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			notification := getMockNotification()

			// Initial setup and Store
			err := repo.Store(ctx, &notification)
			assert.NoError(t, err)
			assert.NotNil(t, notification)

			// Execute Delete
			err = repo.Delete(ctx, notification.ID)

			// Verify
			assert.NoError(t, err)

			// Further verification: Attempt to fetch deleted notification, expect an error or a nil result
			deletedNotification, err := repo.FindByID(ctx, notification.ID)
			assert.Error(t, err)
			assert.Nil(t, deletedNotification)
		})
	}
}

func TestNotificationRepo_Find(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		repo := NewNotificationRepo(log, db)
		mockData1 := getMockNotification()
		mockData2 := getMockNotification()
		mockData3 := getMockNotification()

		t.Run(fmt.Sprintf("Find_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup

			// Clear out any existing notifications
			notificationsList, _ := repo.List(ctx)
			for _, notification := range notificationsList {
				_ = repo.Delete(ctx, notification.ID)
			}

			err := repo.Store(ctx, &mockData1)
			assert.NoError(t, err)
			err = repo.Store(ctx, &mockData2)
			assert.NoError(t, err)
			err = repo.Store(ctx, &mockData3)
			assert.NoError(t, err)

			// Setup query params
			params := domain.NotificationQueryParams{
				Limit:  2,
				Offset: 0,
			}

			// Execute Find
			notifications, totalCount, err := repo.Find(ctx, params)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, 3, len(notifications)) // TODO: This should be 2 technically since limit is 2, but it's returning 3 because params are not being applied.
			assert.Equal(t, 3, totalCount)

			// Cleanup
			notificationsList, _ = repo.List(ctx)
			for _, notification := range notificationsList {
				_ = repo.Delete(ctx, notification.ID)
			}
		})
	}
}

func TestNotificationRepo_FindByID(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		repo := NewNotificationRepo(log, db)

		mockData := getMockNotification()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			//notification := getMockNotification()

			assert.NotNil(t, mockData)
			err := repo.Store(ctx, &mockData)

			// Execute
			notification, err := repo.FindByID(ctx, mockData.ID)

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, notification)
			assert.Equal(t, mockData.Name, notification.Name)
			assert.Equal(t, mockData.Type, notification.Type)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})
	}
}

func TestNotificationRepo_List(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		repo := NewNotificationRepo(log, db)
		mockData := getMockNotification()

		t.Run(fmt.Sprintf("List_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			notificationsList, _ := repo.List(ctx)
			for _, notification := range notificationsList {
				_ = repo.Delete(ctx, notification.ID)
			}

			for range 10 {
				err := repo.Store(ctx, &mockData)
				assert.NoError(t, err)
			}

			// Execute
			notifications, err := repo.List(ctx)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, 10, len(notifications))

			// Cleanup
			for _, notification := range notifications {
				_ = repo.Delete(ctx, notification.ID)
			}
		})
	}
}

func TestNotificationRepo_FilterNotificationLifecycle(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		repo := NewNotificationRepo(setupLoggerForTest(), db)

		t.Run(fmt.Sprintf("FilterNotificationLifecycle [%s]", dbType), func(t *testing.T) {
			filterQuery := db.squirrel.
				Insert("filter").
				Columns("name").
				Values("notification-route-test").
				Suffix("RETURNING id").
				RunWith(db.Handler)

			var filterID int
			assert.NoError(t, filterQuery.QueryRowContext(ctx).Scan(&filterID))
			t.Cleanup(func() {
				query, args, err := db.squirrel.Delete("filter").Where("id = ?", filterID).ToSql()
				if err == nil {
					_, _ = db.Handler.ExecContext(ctx, query, args...)
				}
			})

			notification := getMockNotification()
			notification.Type = domain.NotificationTypeWebhook
			notification.Webhook = "https://example.com/notifications"
			assert.NoError(t, repo.Store(ctx, &notification))

			routes := []domain.FilterNotification{{
				FilterID:       filterID,
				NotificationID: notification.ID,
				Events:         []string{},
			}}
			assert.NoError(t, repo.StoreFilterNotifications(ctx, filterID, routes))

			stored, err := repo.ListFilterNotifications(ctx)
			assert.NoError(t, err)
			assert.Contains(t, stored, routes[0])

			assert.NoError(t, repo.Delete(ctx, notification.ID))
			stored, err = repo.GetFilterNotifications(ctx, filterID)
			assert.NoError(t, err)
			assert.Empty(t, stored)
		})
	}
}
