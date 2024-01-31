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

func getMockRelease() *domain.Release {
	return &domain.Release{
		FilterStatus:   domain.ReleaseStatusFilterApproved,
		Rejections:     []string{"test", "not-a-match"},
		Indexer:        "BTN",
		FilterName:     "ExampleFilter",
		Protocol:       domain.ReleaseProtocolTorrent,
		Implementation: domain.ReleaseImplementationIRC,
		Timestamp:      time.Now(),
		InfoURL:        "https://example.com/info",
		DownloadURL:    "https://example.com/download",
		GroupID:        "group123",
		TorrentID:      "torrent123",
		TorrentName:    "Example.Torrent.Name",
		Size:           123456789,
		Title:          "Example Title",
		Category:       "Movie",
		Season:         1,
		Episode:        2,
		Year:           2023,
		Resolution:     "1080p",
		Source:         "BluRay",
		Codec:          []string{"H.264", "AAC"},
		Container:      "MKV",
		HDR:            []string{"HDR10", "Dolby Vision"},
		Group:          "ExampleGroup",
		Proper:         true,
		Repack:         false,
		Website:        "https://example.com",
		Type:           "Movie",
		Origin:         "P2P",
		Tags:           []string{"Action", "Adventure"},
		Uploader:       "john_doe",
		PreTime:        "10m",
		FilterID:       1,
	}
}

func getMockReleaseActionStatus() *domain.ReleaseActionStatus {
	return &domain.ReleaseActionStatus{
		ID:         0,
		Status:     domain.ReleasePushStatusApproved,
		Action:     "okay",
		ActionID:   10,
		Type:       domain.ActionTypeTest,
		Client:     "qbitorrent",
		Filter:     "Test filter",
		FilterID:   0,
		Rejections: []string{"one rejection", "two rejections"},
		ReleaseID:  0,
		Timestamp:  time.Now(),
	}
}

func TestReleaseRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("StoreReleaseActionStatus_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			createdAction, err := actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(createdAction.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Verify
			assert.NotEqual(t, int64(0), mockData.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdAction.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_StoreReleaseActionStatus(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("StoreReleaseActionStatus_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			createdAction, err := actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(createdAction.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Verify
			assert.NotEqual(t, int64(0), releaseActionMockData.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdAction.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_Find(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		//actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		//releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("FindReleases_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Search with query params
			queryParams := domain.ReleaseQueryParams{
				Limit:  10,
				Offset: 0,
				Sort: map[string]string{
					"Timestamp": "asc",
				},
				Search: "",
			}

			releases, nextCursor, total, err := repo.Find(context.Background(), queryParams)

			// Verify
			assert.NotNil(t, releases)
			assert.NotEqual(t, int64(0), total)
			assert.True(t, nextCursor >= 0)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_FindRecent(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		//actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		//releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("FindRecent_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			releases, err := repo.FindRecent(context.Background())

			// Verify
			assert.NotNil(t, releases)
			assert.Lenf(t, releases, 1, "Expected 1 release, got %d", len(releases))

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_GetIndexerOptions(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("GetIndexerOptions_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			createdAction, err := actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(createdAction.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			options, err := repo.GetIndexerOptions(context.Background())

			// Verify
			assert.NotNil(t, options)
			assert.Len(t, options, 1)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdAction.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_GetActionStatusByReleaseID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("GetActionStatusByReleaseID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			createdAction, err := actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(createdAction.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			actionStatus, err := repo.GetActionStatus(context.Background(), &domain.GetReleaseActionStatusRequest{Id: int(releaseActionMockData.ID)})

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, actionStatus)
			assert.Equal(t, releaseActionMockData.ID, actionStatus.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdAction.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_Get(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("Get_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			createdAction, err := actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(createdAction.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			release, err := repo.Get(context.Background(), &domain.GetReleaseRequest{Id: int(mockData.ID)})

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, release)
			assert.Equal(t, mockData.ID, release.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdAction.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_Stats(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("Stats_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			createdAction, err := actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(createdAction.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			stats, err := repo.Stats(context.Background())

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, stats)
			assert.Equal(t, int64(1), stats.PushApprovedCount)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdAction.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			createdAction, err := actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(createdAction.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			err = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})

			// Verify
			assert.NoError(t, err)

			// Cleanup
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdAction.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}

func TestReleaseRepo_CanDownloadShow(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdClient, err := downloadClientRepo.Store(context.Background(), getMockDownloadClient())
			assert.NoError(t, err)
			assert.NotNil(t, createdClient)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = int32(createdClient.ID)
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			createdAction, err := actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(createdAction.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			canDownload, err := repo.CanDownloadShow(context.Background(), "Example.Torrent.Name", 1, 2)

			// Verify
			assert.NoError(t, err)
			assert.True(t, canDownload)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdAction.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), createdClient.ID)
		})
	}
}
