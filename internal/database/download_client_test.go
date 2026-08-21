// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
)

func getMockDownloadClient() domain.DownloadClient {
	return domain.DownloadClient{
		Name:          "qbitorrent",
		Type:          domain.DownloadClientTypeQbittorrent,
		Enabled:       true,
		Host:          "host",
		Port:          2020,
		TLS:           true,
		TLSSkipVerify: true,
		Username:      "anime",
		Password:      "anime",
		Settings: domain.DownloadClientSettings{
			APIKey: "123",
			Basic: domain.BasicAuth{
				Auth:     true,
				Username: "username",
				Password: "password",
			},
			Rules: domain.DownloadClientRules{
				Enabled:                     true,
				MaxActiveDownloads:          10,
				IgnoreSlowTorrents:          false,
				IgnoreSlowTorrentsCondition: domain.IgnoreSlowTorrentsModeAlways,
				DownloadSpeedThreshold:      0,
				UploadSpeedThreshold:        0,
			},
			ExternalDownloadClientId: 0,
			ExternalDownloadClient:   "",
			Auth: domain.DownloadClientAuth{
				Enabled:  true,
				Type:     domain.DownloadClientAuthTypeBasic,
				Username: "username",
				Password: "password",
			},
		},
	}
}

func getMockArrList(filterID int, clientID int32) *domain.List {
	return &domain.List{
		Name:        "radarr-list",
		Type:        domain.ListTypeRadarr,
		Enabled:     true,
		ClientID:    int(clientID),
		Headers:     []string{},
		TagsInclude: []string{},
		TagsExclude: []string{},
		Filters: []domain.ListFilter{
			{
				ID:   filterID,
				Name: "filter",
			},
		},
	}
}

func TestDownloadClientRepo_List(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)
		mockData := getMockDownloadClient()

		t.Run(fmt.Sprintf("List_Succeeds_With_No_Filters [%s]", dbType), func(t *testing.T) {
			// Insert mock data
			mock := &mockData
			err := repo.Store(ctx, mock)
			clients, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.NotEmpty(t, clients)

			// Cleanup
			_ = repo.Delete(ctx, mock.ID)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Empty_Database [%s]", dbType), func(t *testing.T) {
			clients, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.Empty(t, clients)
		})

		t.Run(fmt.Sprintf("List_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
			defer cancel()
			_, err := repo.List(timeoutCtx)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Data_Integrity [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			err := repo.Store(ctx, mock)
			clients, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 1, len(clients))
			assert.Equal(t, mock.Name, clients[0].Name)

			// Cleanup
			_ = repo.Delete(ctx, mock.ID)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Boundary_Value_For_Port [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			mock.Port = 65535
			err := repo.Store(ctx, mock)
			clients, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 65535, clients[0].Port)

			// Cleanup
			_ = repo.Delete(ctx, mock.ID)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Boolean_Flags_Set_To_False [%s]", dbType), func(t *testing.T) {
			mockData.Enabled = false
			mockData.TLS = false
			mockData.TLSSkipVerify = false
			err := repo.Store(ctx, &mockData)
			clients, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.Equal(t, false, clients[0].Enabled)
			assert.Equal(t, false, clients[0].TLS)
			assert.Equal(t, false, clients[0].TLSSkipVerify)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Special_Characters_In_Name [%s]", dbType), func(t *testing.T) {
			mockData.Name = "Special$Name"
			err := repo.Store(ctx, &mockData)
			clients, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.Equal(t, "Special$Name", clients[0].Name)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})
	}
}

func TestDownloadClientRepo_FindByID(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)
		mockData := getMockDownloadClient()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			_ = repo.Store(ctx, mock)
			foundClient, err := repo.FindByID(ctx, mock.ID)
			assert.NoError(t, err)
			assert.NotNil(t, foundClient)

			// Cleanup
			_ = repo.Delete(ctx, mock.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_With_Nonexistent_ID [%s]", dbType), func(t *testing.T) {
			_, err := repo.FindByID(ctx, 9999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_With_Negative_ID [%s]", dbType), func(t *testing.T) {
			_, err := repo.FindByID(ctx, -1)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
			defer cancel()

			_, err := repo.FindByID(timeoutCtx, 1)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_After_Client_Deleted [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			_ = repo.Store(ctx, mock)
			_ = repo.Delete(ctx, mock.ID)
			_, err := repo.FindByID(ctx, mock.ID)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)

			// Cleanup
			_ = repo.Delete(ctx, mock.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Succeeds_With_Data_Integrity [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			_ = repo.Store(ctx, mock)
			foundClient, err := repo.FindByID(ctx, mock.ID)
			assert.NoError(t, err)
			assert.Equal(t, mock.Name, foundClient.Name)

			// Cleanup
			_ = repo.Delete(ctx, mock.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Succeeds_From_Cache [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			_ = repo.Store(ctx, mock)
			foundClient1, _ := repo.FindByID(ctx, mock.ID)
			foundClient2, err := repo.FindByID(ctx, mock.ID)
			assert.NoError(t, err)
			assert.Equal(t, foundClient1, foundClient2)

			// Cleanup
			_ = repo.Delete(ctx, mock.ID)
		})
	}
}

func TestDownloadClientRepo_Store(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			mockData := getMockDownloadClient()
			err := repo.Store(ctx, &mockData)
			assert.NoError(t, err)
			assert.NotNil(t, mockData)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})

		//TODO: Is this okay? Should we be able to store a client with no name (empty string)?
		t.Run(fmt.Sprintf("Store_Succeeds?_With_Missing_Required_Fields [%s]", dbType), func(t *testing.T) {
			badMockData := &domain.DownloadClient{
				Type:          "",
				Enabled:       false,
				Host:          "",
				Port:          0,
				TLS:           false,
				TLSSkipVerify: false,
				Username:      "",
				Password:      "",
				Settings:      domain.DownloadClientSettings{},
			}
			err := repo.Store(ctx, badMockData)
			assert.NoError(t, err)

			// Cleanup
			_ = repo.Delete(ctx, badMockData.ID)
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			mockData := getMockDownloadClient()
			timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
			defer cancel()
			err := repo.Store(timeoutCtx, &mockData)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Store_Succeeds_And_Caches [%s]", dbType), func(t *testing.T) {
			mockData := getMockDownloadClient()
			_ = repo.Store(ctx, &mockData)

			cachedClient, _ := repo.FindByID(ctx, mockData.ID)
			assert.Equal(t, &mockData, cachedClient)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})
	}
}

func TestDownloadClientRepo_Update(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)

		t.Run(fmt.Sprintf("Update_Successfully_Updates_Record [%s]", dbType), func(t *testing.T) {
			mockClient := getMockDownloadClient()

			_ = repo.Store(ctx, &mockClient)
			mockClient.Name = "updatedName"
			err := repo.Update(ctx, &mockClient)

			assert.NoError(t, err)
			assert.Equal(t, "updatedName", mockClient.Name)

			// Cleanup
			_ = repo.Delete(ctx, mockClient.ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_With_Missing_ID [%s]", dbType), func(t *testing.T) {
			badMockData := getMockDownloadClient()
			badMockData.ID = 0

			err := repo.Update(ctx, &badMockData)

			assert.Error(t, err)

		})

		t.Run(fmt.Sprintf("Update_Fails_With_Nonexistent_ID [%s]", dbType), func(t *testing.T) {
			badMockData := getMockDownloadClient()
			badMockData.ID = 9999

			err := repo.Update(ctx, &badMockData)

			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Update_Fails_With_Missing_Required_Fields [%s]", dbType), func(t *testing.T) {
			badMockData := domain.DownloadClient{}

			err := repo.Update(ctx, &badMockData)

			assert.Error(t, err)
		})
	}
}

func TestDownloadClientRepo_Delete(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)

		t.Run(fmt.Sprintf("Delete_Successfully_Deletes_Client [%s]", dbType), func(t *testing.T) {
			mockClient := getMockDownloadClient()
			_ = repo.Store(ctx, &mockClient)

			err := repo.Delete(ctx, mockClient.ID)
			assert.NoError(t, err)

			// Verify client was deleted
			_, err = repo.FindByID(ctx, mockClient.ID)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Delete_Fails_With_Nonexistent_Client_ID [%s]", dbType), func(t *testing.T) {
			err := repo.Delete(ctx, 9999)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Delete_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			mockClient := getMockDownloadClient()
			_ = repo.Store(ctx, &mockClient)

			timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
			defer cancel()

			err := repo.Delete(timeoutCtx, mockClient.ID)
			assert.Error(t, err)

			// Cleanup
			_ = repo.Delete(ctx, mockClient.ID)
		})

		t.Run(fmt.Sprintf("Delete_Clears_Client_From_Actions [%s]", dbType), func(t *testing.T) {
			actionRepo := NewActionRepo(log, db)
			filterRepo := NewFilterRepo(log, db)

			mockClient := getMockDownloadClient()
			err := repo.Store(ctx, &mockClient)
			assert.NoError(t, err)

			filter := getMockFilter()
			err = filterRepo.Store(ctx, filter)
			assert.NoError(t, err)

			actionWithoutFilter := getMockAction()
			actionWithoutFilter.ClientID = mockClient.ID
			actionWithoutFilter.FilterID = 0
			actionWithoutFilter.Name = "action-without-filter"

			err = actionRepo.Store(ctx, actionWithoutFilter)
			assert.NoError(t, err)

			actionWithFilter := getMockAction()
			actionWithFilter.ClientID = mockClient.ID
			actionWithFilter.FilterID = filter.ID
			actionWithFilter.Name = "action-with-filter"

			err = actionRepo.Store(ctx, actionWithFilter)
			assert.NoError(t, err)

			err = repo.Delete(ctx, mockClient.ID)
			assert.NoError(t, err)

			updatedActionWithoutFilter, err := actionRepo.Get(ctx, &domain.GetActionRequest{Id: actionWithoutFilter.ID})
			assert.NoError(t, err)
			assert.False(t, updatedActionWithoutFilter.Enabled)
			assert.Zero(t, updatedActionWithoutFilter.ClientID)
			assert.Zero(t, updatedActionWithoutFilter.FilterID)

			updatedActionWithFilter, err := actionRepo.Get(ctx, &domain.GetActionRequest{Id: actionWithFilter.ID})
			assert.NoError(t, err)
			assert.False(t, updatedActionWithFilter.Enabled)
			assert.Zero(t, updatedActionWithFilter.ClientID)
			assert.Equal(t, filter.ID, updatedActionWithFilter.FilterID)

			_, err = repo.FindByID(ctx, mockClient.ID)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)

			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionWithoutFilter.ID})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionWithFilter.ID})
			_ = filterRepo.Delete(ctx, filter.ID)
		})

		t.Run(fmt.Sprintf("Delete_Clears_Client_From_Lists [%s]", dbType), func(t *testing.T) {
			filterRepo := NewFilterRepo(log, db)
			listRepo := NewListRepo(log, db)

			mockClient := getMockDownloadClient()
			err := repo.Store(ctx, &mockClient)
			assert.NoError(t, err)

			filter := getMockFilter()
			err = filterRepo.Store(ctx, filter)
			assert.NoError(t, err)

			list := getMockArrList(filter.ID, mockClient.ID)
			err = listRepo.Store(ctx, list)
			assert.NoError(t, err)

			err = repo.Delete(ctx, mockClient.ID)
			assert.NoError(t, err)

			lists, err := listRepo.List(ctx)
			assert.NoError(t, err)

			var updatedList *domain.List
			for _, existingList := range lists {
				if existingList.ID == list.ID {
					updatedList = existingList
					break
				}
			}

			if assert.NotNil(t, updatedList) {
				assert.False(t, updatedList.Enabled)
				assert.Zero(t, updatedList.ClientID)
			}

			_, err = repo.FindByID(ctx, mockClient.ID)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)

			_ = listRepo.Delete(ctx, list.ID)
			_ = filterRepo.Delete(ctx, filter.ID)
		})
	}
}
