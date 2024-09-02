// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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

func TestDownloadClientRepo_List(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)
		mockData := getMockDownloadClient()

		t.Run(fmt.Sprintf("List_Succeeds_With_No_Filters [%s]", dbType), func(t *testing.T) {
			// Insert mock data
			mock := &mockData
			err := repo.Store(context.Background(), mock)
			clients, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.NotEmpty(t, clients)

			// Cleanup
			_ = repo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Empty_Database [%s]", dbType), func(t *testing.T) {
			clients, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.Empty(t, clients)
		})

		t.Run(fmt.Sprintf("List_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()
			_, err := repo.List(ctx)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Data_Integrity [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			err := repo.Store(context.Background(), mock)
			clients, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, 1, len(clients))
			assert.Equal(t, mock.Name, clients[0].Name)

			// Cleanup
			_ = repo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Boundary_Value_For_Port [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			mock.Port = 65535
			err := repo.Store(context.Background(), mock)
			clients, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, 65535, clients[0].Port)

			// Cleanup
			_ = repo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Boolean_Flags_Set_To_False [%s]", dbType), func(t *testing.T) {
			mockData.Enabled = false
			mockData.TLS = false
			mockData.TLSSkipVerify = false
			err := repo.Store(context.Background(), &mockData)
			clients, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, false, clients[0].Enabled)
			assert.Equal(t, false, clients[0].TLS)
			assert.Equal(t, false, clients[0].TLSSkipVerify)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("List_Succeeds_With_Special_Characters_In_Name [%s]", dbType), func(t *testing.T) {
			mockData.Name = "Special$Name"
			err := repo.Store(context.Background(), &mockData)
			clients, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, "Special$Name", clients[0].Name)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})
	}
}

func TestDownloadClientRepo_FindByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)
		mockData := getMockDownloadClient()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			_ = repo.Store(context.Background(), mock)
			foundClient, err := repo.FindByID(context.Background(), mock.ID)
			assert.NoError(t, err)
			assert.NotNil(t, foundClient)

			// Cleanup
			_ = repo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_With_Nonexistent_ID [%s]", dbType), func(t *testing.T) {
			_, err := repo.FindByID(context.Background(), 9999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_With_Negative_ID [%s]", dbType), func(t *testing.T) {
			_, err := repo.FindByID(context.Background(), -1)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			_, err := repo.FindByID(ctx, 1)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_After_Client_Deleted [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			_ = repo.Store(context.Background(), mock)
			_ = repo.Delete(context.Background(), mock.ID)
			_, err := repo.FindByID(context.Background(), mock.ID)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)

			// Cleanup
			_ = repo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Succeeds_With_Data_Integrity [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			_ = repo.Store(context.Background(), mock)
			foundClient, err := repo.FindByID(context.Background(), mock.ID)
			assert.NoError(t, err)
			assert.Equal(t, mock.Name, foundClient.Name)

			// Cleanup
			_ = repo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Succeeds_From_Cache [%s]", dbType), func(t *testing.T) {
			mock := &mockData
			_ = repo.Store(context.Background(), mock)
			foundClient1, _ := repo.FindByID(context.Background(), mock.ID)
			foundClient2, err := repo.FindByID(context.Background(), mock.ID)
			assert.NoError(t, err)
			assert.Equal(t, foundClient1, foundClient2)

			// Cleanup
			_ = repo.Delete(context.Background(), mock.ID)
		})
	}
}

func TestDownloadClientRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			mockData := getMockDownloadClient()
			err := repo.Store(context.Background(), &mockData)
			assert.NoError(t, err)
			assert.NotNil(t, mockData)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
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
			err := repo.Store(context.Background(), badMockData)
			assert.NoError(t, err)

			// Cleanup
			_ = repo.Delete(context.Background(), badMockData.ID)
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			mockData := getMockDownloadClient()
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()
			err := repo.Store(ctx, &mockData)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Store_Succeeds_And_Caches [%s]", dbType), func(t *testing.T) {
			mockData := getMockDownloadClient()
			_ = repo.Store(context.Background(), &mockData)

			cachedClient, _ := repo.FindByID(context.Background(), mockData.ID)
			assert.Equal(t, &mockData, cachedClient)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})
	}
}

func TestDownloadClientRepo_Update(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)

		t.Run(fmt.Sprintf("Update_Successfully_Updates_Record [%s]", dbType), func(t *testing.T) {
			mockClient := getMockDownloadClient()

			_ = repo.Store(context.Background(), &mockClient)
			mockClient.Name = "updatedName"
			err := repo.Update(context.Background(), &mockClient)

			assert.NoError(t, err)
			assert.Equal(t, "updatedName", mockClient.Name)

			// Cleanup
			_ = repo.Delete(context.Background(), mockClient.ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_With_Missing_ID [%s]", dbType), func(t *testing.T) {
			badMockData := getMockDownloadClient()
			badMockData.ID = 0

			err := repo.Update(context.Background(), &badMockData)

			assert.Error(t, err)

		})

		t.Run(fmt.Sprintf("Update_Fails_With_Nonexistent_ID [%s]", dbType), func(t *testing.T) {
			badMockData := getMockDownloadClient()
			badMockData.ID = 9999

			err := repo.Update(context.Background(), &badMockData)

			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Update_Fails_With_Missing_Required_Fields [%s]", dbType), func(t *testing.T) {
			badMockData := domain.DownloadClient{}

			err := repo.Update(context.Background(), &badMockData)

			assert.Error(t, err)
		})
	}
}

func TestDownloadClientRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewDownloadClientRepo(log, db)

		t.Run(fmt.Sprintf("Delete_Successfully_Deletes_Client [%s]", dbType), func(t *testing.T) {
			mockClient := getMockDownloadClient()
			_ = repo.Store(context.Background(), &mockClient)

			err := repo.Delete(context.Background(), mockClient.ID)
			assert.NoError(t, err)

			// Verify client was deleted
			_, err = repo.FindByID(context.Background(), mockClient.ID)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Delete_Fails_With_Nonexistent_Client_ID [%s]", dbType), func(t *testing.T) {
			err := repo.Delete(context.Background(), 9999)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Delete_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			mockClient := getMockDownloadClient()
			_ = repo.Store(context.Background(), &mockClient)

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			err := repo.Delete(ctx, mockClient.ID)
			assert.Error(t, err)

			// Cleanup
			_ = repo.Delete(context.Background(), mockClient.ID)
		})
	}
}
