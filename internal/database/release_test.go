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

	"github.com/moistari/rls"
	"github.com/stretchr/testify/assert"
)

func getMockRelease() *domain.Release {
	return &domain.Release{
		FilterStatus: domain.ReleaseStatusFilterApproved,
		Rejections:   []string{"test", "not-a-match"},
		Indexer: domain.IndexerMinimal{
			ID:         0,
			Name:       "BTN",
			Identifier: "btn",
		},
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
		Type:           rls.Movie,
		Origin:         "P2P",
		Tags:           []string{"Action", "Adventure"},
		Uploader:       "john_doe",
		PreTime:        "10m",
		FilterID:       1,
		Other:          []string{},
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
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Verify
			assert.NotEqual(t, int64(0), mockData.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
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
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			// Verify
			assert.NotEqual(t, int64(0), releaseActionMockData.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
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
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
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

			resp, err := repo.Find(context.Background(), queryParams)

			// Verify
			assert.NotNil(t, resp)
			assert.NotEqual(t, int64(0), resp.TotalCount)
			assert.True(t, resp.NextCursor >= 0)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
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
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			resp, err := repo.Find(context.Background(), domain.ReleaseQueryParams{Limit: 10})

			// Verify
			assert.NotNil(t, resp.Data)
			assert.Lenf(t, resp.Data, 1, "Expected 1 release, got %d", len(resp.Data))

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
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
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
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
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
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
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
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
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
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
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
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
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
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
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
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
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
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

		// Setup shared dependencies
		mock := getMockDownloadClient()
		err := downloadClientRepo.Store(context.Background(), &mock)
		assert.NoError(t, err)

		err = filterRepo.Store(context.Background(), getMockFilter())
		assert.NoError(t, err)

		createdFilters, err := filterRepo.ListFilters(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, createdFilters)

		actionMock := getMockAction()
		actionMock.FilterID = createdFilters[0].ID
		actionMock.ClientID = mock.ID
		err = actionRepo.Store(context.Background(), actionMock)
		assert.NoError(t, err)

		tests := []struct {
			name              string
			deleteReq         *domain.DeleteReleaseRequest
			expectedRemaining int
		}{
			{
				name:              "OlderThan_Precision_24Hours",
				deleteReq:         &domain.DeleteReleaseRequest{OlderThan: 24},
				expectedRemaining: 2,
			},
			{
				name:              "Indexer_Filter",
				deleteReq:         &domain.DeleteReleaseRequest{OlderThan: 0, Indexers: []string{"btn", "ptp"}},
				expectedRemaining: 1,
			},
			{
				name:              "Status_Filter",
				deleteReq:         &domain.DeleteReleaseRequest{OlderThan: 0, ReleaseStatuses: []string{"PUSH_REJECTED", "PUSH_ERROR"}},
				expectedRemaining: 2,
			},
			{
				name:              "Combined_Filters",
				deleteReq:         &domain.DeleteReleaseRequest{OlderThan: 24, Indexers: []string{"btn"}, ReleaseStatuses: []string{"PUSH_REJECTED"}},
				expectedRemaining: 3,
			},
			{
				name:              "Delete_All",
				deleteReq:         &domain.DeleteReleaseRequest{OlderThan: 0},
				expectedRemaining: 0,
			},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("Delete_%s [%s]", tt.name, dbType), func(t *testing.T) {
				ctx := context.Background()

				// Setup - create test-specific releases
				switch tt.name {
				case "OlderThan_Precision_24Hours":
					// Test datetime precision: create releases avoiding exact boundary
					for i, age := range []time.Duration{
						22*time.Hour + 30*time.Minute, // Should be kept (clearly younger than 24h)
						23*time.Hour + 45*time.Minute, // Should be kept (younger than 24h)
						24*time.Hour + 30*time.Minute, // Should be deleted (clearly older than 24h)
						25*time.Hour + 30*time.Minute, // Should be deleted (much older than 24h)
					} {
						mockRel := getMockRelease()
						mockRel.Timestamp = time.Now().Add(-age)
						mockRel.FilterID = createdFilters[0].ID
						err := repo.Store(ctx, mockRel)
						assert.NoError(t, err)

						ras := getMockReleaseActionStatus()
						ras.ReleaseID = mockRel.ID
						ras.ActionID = int64(actionMock.ID)
						ras.FilterID = int64(createdFilters[0].ID)
						ras.Status = domain.ReleasePushStatusApproved
						err = repo.StoreReleaseActionStatus(ctx, ras)
						assert.NoError(t, err)
						_ = i
					}

				case "Indexer_Filter":
					// Test indexer filtering: create releases from different indexers
					for _, indexer := range []string{"btn", "ptp", "hdt"} {
						mockRel := getMockRelease()
						mockRel.Indexer.Identifier = indexer
						mockRel.FilterID = createdFilters[0].ID
						err := repo.Store(ctx, mockRel)
						assert.NoError(t, err)

						ras := getMockReleaseActionStatus()
						ras.ReleaseID = mockRel.ID
						ras.ActionID = int64(actionMock.ID)
						ras.FilterID = int64(createdFilters[0].ID)
						err = repo.StoreReleaseActionStatus(ctx, ras)
						assert.NoError(t, err)
					}

				case "Status_Filter":
					// Test status filtering: create releases with all statuses including PENDING.
					// Validates that PENDING is excluded from deletion per domain.ValidDeletableReleasePushStatus.
					// Expected: PUSH_APPROVED and PUSH_PENDING remain, PUSH_REJECTED and PUSH_ERROR deleted.
					for _, status := range []domain.ReleasePushStatus{
						domain.ReleasePushStatusApproved,
						domain.ReleasePushStatusRejected,
						domain.ReleasePushStatusErr,
						domain.ReleasePushStatusPending,
					} {
						mockRel := getMockRelease()
						mockRel.FilterID = createdFilters[0].ID
						err := repo.Store(ctx, mockRel)
						assert.NoError(t, err)

						ras := getMockReleaseActionStatus()
						ras.ReleaseID = mockRel.ID
						ras.ActionID = int64(actionMock.ID)
						ras.FilterID = int64(createdFilters[0].ID)
						ras.Status = status
						err = repo.StoreReleaseActionStatus(ctx, ras)
						assert.NoError(t, err)
					}

				case "Combined_Filters":
					// Test combined filters: age + indexer + status
					testData := []struct {
						age     time.Duration
						indexer string
						status  domain.ReleasePushStatus
					}{
						{20 * time.Hour, "btn", domain.ReleasePushStatusApproved}, // Keep (age)
						{25 * time.Hour, "ptp", domain.ReleasePushStatusRejected}, // Keep (indexer)
						{25 * time.Hour, "btn", domain.ReleasePushStatusApproved}, // Keep (status)
						{25 * time.Hour, "btn", domain.ReleasePushStatusRejected}, // Delete (matches all filters)
					}

					for _, td := range testData {
						mockRel := getMockRelease()
						mockRel.Timestamp = time.Now().Add(-td.age)
						mockRel.Indexer.Identifier = td.indexer
						mockRel.FilterID = createdFilters[0].ID
						err := repo.Store(ctx, mockRel)
						assert.NoError(t, err)

						ras := getMockReleaseActionStatus()
						ras.ReleaseID = mockRel.ID
						ras.ActionID = int64(actionMock.ID)
						ras.FilterID = int64(createdFilters[0].ID)
						ras.Status = td.status
						err = repo.StoreReleaseActionStatus(ctx, ras)
						assert.NoError(t, err)
					}

				case "Delete_All":
					// Test delete all: create 3 releases with any variation
					for i := 0; i < 3; i++ {
						mockRel := getMockRelease()
						mockRel.FilterID = createdFilters[0].ID
						err := repo.Store(ctx, mockRel)
						assert.NoError(t, err)

						ras := getMockReleaseActionStatus()
						ras.ReleaseID = mockRel.ID
						ras.ActionID = int64(actionMock.ID)
						ras.FilterID = int64(createdFilters[0].ID)
						err = repo.StoreReleaseActionStatus(ctx, ras)
						assert.NoError(t, err)
					}
				}

				// Execute
				err := repo.Delete(ctx, tt.deleteReq)
				assert.NoError(t, err)

				// Verify
				releases, err := repo.Find(ctx, domain.ReleaseQueryParams{})
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRemaining, len(releases.Data), "Expected %d releases to remain, got %d", tt.expectedRemaining, len(releases.Data))

				// Cleanup
				_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			})
		}

		// Cleanup shared resources
		_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMock.ID})
		_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
		_ = downloadClientRepo.Delete(context.Background(), mock.ID)
	}
}

func TestReleaseRepo_CheckSmartEpisodeCanDownloadShow(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("Check_Smart_Episode_Can_Download [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloadClient()
			err := downloadClientRepo.Store(context.Background(), &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(context.Background(), actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(context.Background(), releaseActionMockData)
			assert.NoError(t, err)

			params := &domain.SmartEpisodeParams{
				Title:   "Example.Torrent.Name",
				Season:  1,
				Episode: 2,
				Year:    0,
				Month:   0,
				Day:     0,
			}

			// Execute
			canDownload, err := repo.CheckSmartEpisodeCanDownload(context.Background(), params)

			// Verify
			assert.NoError(t, err)
			assert.True(t, canDownload)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})
	}
}

func getMockDuplicateReleaseProfileTV() *domain.DuplicateReleaseProfile {
	return &domain.DuplicateReleaseProfile{
		ID:           0,
		Name:         "TV",
		Protocol:     false,
		ReleaseName:  false,
		Hash:         false,
		Title:        true,
		SubTitle:     false,
		Year:         false,
		Month:        false,
		Day:          false,
		Source:       false,
		Resolution:   false,
		Codec:        false,
		Container:    false,
		DynamicRange: false,
		Audio:        false,
		Group:        false,
		Season:       true,
		Episode:      true,
		Website:      false,
		Proper:       false,
		Repack:       false,
		Edition:      false,
		Language:     false,
	}
}

func getMockDuplicateReleaseProfileTVDaily() *domain.DuplicateReleaseProfile {
	return &domain.DuplicateReleaseProfile{
		ID:           0,
		Name:         "TV",
		Protocol:     false,
		ReleaseName:  false,
		Hash:         false,
		Title:        true,
		SubTitle:     false,
		Year:         true,
		Month:        true,
		Day:          true,
		Source:       false,
		Resolution:   false,
		Codec:        false,
		Container:    false,
		DynamicRange: false,
		Audio:        false,
		Group:        false,
		Season:       false,
		Episode:      false,
		Website:      false,
		Proper:       false,
		Repack:       false,
		Edition:      false,
		Language:     false,
	}
}

func getMockFilterDuplicates() *domain.Filter {
	return &domain.Filter{
		Name:                 "New Filter",
		Enabled:              true,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
		MinSize:              "10mb",
		MaxSize:              "20mb",
		Delay:                60,
		Priority:             1,
		MaxDownloads:         100,
		MaxDownloadsUnit:     domain.FilterMaxDownloadsHour,
		MatchReleases:        "BRRip",
		ExceptReleases:       "BRRip",
		UseRegex:             false,
		MatchReleaseGroups:   "AMIABLE",
		ExceptReleaseGroups:  "NTb",
		Scene:                false,
		Origins:              nil,
		ExceptOrigins:        nil,
		Bonus:                nil,
		Freeleech:            false,
		FreeleechPercent:     "100%",
		SmartEpisode:         false,
		Shows:                "Is It Wrong to Try to Pick Up Girls in a Dungeon?",
		Seasons:              "4",
		Episodes:             "500",
		Resolutions:          []string{"1080p"},
		Codecs:               []string{"x264"},
		Sources:              []string{"BluRay"},
		Containers:           []string{"mkv"},
		MatchHDR:             []string{"HDR10"},
		ExceptHDR:            []string{"HDR10"},
		MatchOther:           []string{"Atmos"},
		ExceptOther:          []string{"Atmos"},
		Years:                "2023",
		Months:               "",
		Days:                 "",
		Artists:              "",
		Albums:               "",
		MatchReleaseTypes:    []string{"Remux"},
		ExceptReleaseTypes:   "Remux",
		Formats:              []string{"FLAC"},
		Quality:              []string{"Lossless"},
		Media:                []string{"CD"},
		PerfectFlac:          true,
		Cue:                  true,
		Log:                  true,
		LogScore:             100,
		MatchCategories:      "Anime",
		ExceptCategories:     "Anime",
		MatchUploaders:       "SubsPlease",
		ExceptUploaders:      "SubsPlease",
		MatchLanguage:        []string{"English", "Japanese"},
		ExceptLanguage:       []string{"English", "Japanese"},
		Tags:                 "Anime, x264",
		ExceptTags:           "Anime, x264",
		TagsAny:              "Anime, x264",
		ExceptTagsAny:        "Anime, x264",
		TagsMatchLogic:       "AND",
		ExceptTagsMatchLogic: "AND",
		MatchReleaseTags:     "Anime, x264",
		ExceptReleaseTags:    "Anime, x264",
		UseRegexReleaseTags:  true,
		MatchDescription:     "Anime, x264",
		ExceptDescription:    "Anime, x264",
		UseRegexDescription:  true,
	}
}

func TestReleaseRepo_CheckIsDuplicateRelease(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)
		releaseRepo := NewReleaseRepo(log, db)

		// reset
		//db.Handler.Exec("DELETE FROM release")
		//db.Handler.Exec("DELETE FROM action")
		//db.Handler.Exec("DELETE FROM release_action_status")

		mockIndexer := domain.IndexerMinimal{ID: 0, Name: "Mock", Identifier: "mock", IdentifierExternal: "Mock"}
		actionMock := &domain.Action{Name: "Test", Type: domain.ActionTypeTest, Enabled: true}
		filterMock := getMockFilterDuplicates()

		// Setup
		err := filterRepo.Store(context.Background(), filterMock)
		assert.NoError(t, err)

		createdFilters, err := filterRepo.ListFilters(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, createdFilters)

		actionMock.FilterID = filterMock.ID

		err = actionRepo.Store(context.Background(), actionMock)
		assert.NoError(t, err)

		type fields struct {
			releaseTitles []string
			releaseTitle  string
			profile       *domain.DuplicateReleaseProfile
		}

		tests := []struct {
			name        string
			fields      fields
			isDuplicate bool
		}{
			{
				name: "1",
				fields: fields{
					releaseTitles: []string{
						"Inkheart 2008 BluRay 1080p DD5.1 x264-BADGROUP",
					},
					releaseTitle: "Inkheart 2008 BluRay 1080p DD5.1 x264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "2",
				fields: fields{
					releaseTitles: []string{
						"That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP",
						"That.Movie.2023.BluRay.720p.x265.DTS-HD-GROUP",
						"That.Movie.2023.WEB.2160p.x265.DTS-HD-GROUP",
					},
					releaseTitle: "That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP1",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Source: true, Resolution: true},
				},
				isDuplicate: true,
			},
			{
				name: "3",
				fields: fields{
					releaseTitles: []string{
						"That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP",
						"That.Movie.2023.BluRay.720p.x265.DTS-HD-GROUP",
						"That.Movie.2023.WEB.2160p.x265.DTS-HD-GROUP",
					},
					releaseTitle: "That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP1",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true},
				},
				isDuplicate: true,
			},
			{
				name: "4",
				fields: fields{
					releaseTitles: []string{
						"That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP",
						"That.Movie.2023.BluRay.720p.x265.DTS-HD-GROUP",
						"That.Movie.2023.WEB.2160p.x265.DTS-HD-GROUP",
					},
					releaseTitle: "That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP1",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "5",
				fields: fields{
					releaseTitles: []string{
						"That.Tv.Show.2023.S01E01.BluRay.2160p.x265.DTS-HD-GROUP",
					},
					releaseTitle: "That.Tv.Show.2023.S01E01.BluRay.2160p.x265.DTS-HD-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Group: true},
				},
				isDuplicate: true,
			},
			{
				name: "6",
				fields: fields{
					releaseTitles: []string{
						"That.Tv.Show.2023.S01E01.BluRay.2160p.x265.DTS-HD-GROUP",
					},
					releaseTitle: "That.Tv.Show.2023.S01E02.BluRay.2160p.x265.DTS-HD-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "7",
				fields: fields{
					releaseTitles: []string{
						"That.Tv.Show.2023.S01.BluRay.2160p.x265.DTS-HD-GROUP",
					},
					releaseTitle: "That.Tv.Show.2023.S01.BluRay.2160p.x265.DTS-HD-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Group: true},
				},
				isDuplicate: true,
			},
			{
				name: "8",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p AMZN WEB-DL DDP 5.1 SDR H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 1080p AMZN WEB-DL DDP 5.1 SDR H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Website: true, Group: true},
				},
				isDuplicate: true,
			},
			{
				name: "9",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.HULU.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 1080p AMZN WEB-DL DDP 5.1 SDR H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Website: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "10",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 1080p AMZN WEB-DL DDP 5.1 H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Website: true, DynamicRange: true, Group: true},
				},
				isDuplicate: true,
			},
			{
				name: "11",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
						"The Best Show 2020 S04E10 1080p amzn web-dl ddp 5.1 hdr dv h.264-group",
					},
					releaseTitle: "The Best Show 2020 S04E10 1080p AMZN WEB-DL DDP 5.1 HDR DV H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Website: true, DynamicRange: true},
				},
				isDuplicate: true,
			},
			{
				name: "12",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 1080p AMZN WEB-DL DDP 5.1 DV H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Website: true, DynamicRange: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "13",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 Episode Title 1080p AMZN WEB-DL DDP 5.1 HDR DV H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, SubTitle: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Website: true, DynamicRange: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "14",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The.Best.Show.2020.S04E10.Episode.Title.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 Episode Title 1080p AMZN WEB-DL DDP 5.1 HDR DV H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, SubTitle: true, Year: true, Season: true, Episode: true, Source: true, Codec: true, Resolution: true, Website: true, DynamicRange: true, Group: true},
				},
				isDuplicate: true,
			},
			{
				name: "15",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The.Best.Show.2020.S04E10.Episode.Title.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 Episode Title 1080p AMZN WEB-DL DDP 5.1 HDR DV H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, SubTitle: true, Season: true, Episode: true, DynamicRange: true},
				},
				isDuplicate: true,
			},
			{
				name: "16",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The.Best.Show.2020.S04E10.Episode.Title.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E11 Episode Title 1080p AMZN WEB-DL DDP 5.1 HDR DV H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, SubTitle: true, Season: true, Episode: true, DynamicRange: true},
				},
				isDuplicate: false,
			},
			{
				name: "17",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The.Best.Show.2020.S04E10.Episode.Title.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 Episode Title REPACK 1080p AMZN WEB-DL DDP 5.1 HDR DV H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, SubTitle: true, Season: true, Episode: true, DynamicRange: true},
				},
				isDuplicate: true,
			},
			{
				name: "18",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The.Best.Show.2020.S04E10.Episode.Title.REPACK.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 Episode Title REPACK 1080p AMZN WEB-DL DDP 5.1 DV H.264-OTHERGROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Season: true, Episode: true, Repack: true},
				},
				isDuplicate: false, // not a match because REPACK checks for the same group
			},
			{
				name: "18_1",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-OTHERGROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Season: true, Episode: true, Repack: true},
				},
				isDuplicate: true, // Repack should not force group match when release isn't a repack
			},
			{
				name: "18_2",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 REPACK 1080p HULU WEB-DL DDP 5.1 SDR H.264-OTHERGROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Season: true, Episode: true, Repack: true},
				},
				isDuplicate: false, // Repack should not force group match when release isn't a repack
			},
			{
				name: "18_3",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 REPACK 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Season: true, Episode: true, Repack: true},
				},
				isDuplicate: false, // Repack with same group is not a duplicate
			},
			{
				name: "19",
				fields: fields{
					releaseTitles: []string{
						"The Daily Show 2024-09-21 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The Daily Show 2024-09-21.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The Daily Show 2024-09-21.Guest.1080p.AMZN.WEB-DL.DDP.5.1.H.264-GROUP1",
					},
					releaseTitle: "The Daily Show 2024-09-21.Other.Guest.1080p.AMZN.WEB-DL.DDP.5.1.H.264-GROUP1",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Season: true, Episode: true, Year: true, Month: true, Day: true},
				},
				isDuplicate: true,
			},
			{
				name: "20",
				fields: fields{
					releaseTitles: []string{
						"The Daily Show 2024-09-21 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The Daily Show 2024-09-21.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The Daily Show 2024-09-21.Guest.1080p.AMZN.WEB-DL.DDP.5.1.H.264-GROUP1",
					},
					releaseTitle: "The Daily Show 2024-09-21 Other Guest 1080p AMZN WEB-DL DDP 5.1 H.264-GROUP1",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Season: true, Episode: true, Year: true, Month: true, Day: true, SubTitle: true},
				},
				isDuplicate: false,
			},
			{
				name: "21",
				fields: fields{
					releaseTitles: []string{
						"The Daily Show 2024-09-21 1080p HULU WEB-DL DDP 5.1 SDR H.264-GROUP",
						"The Daily Show 2024-09-21.1080p.AMZN.WEB-DL.DDP.5.1.SDR.H.264-GROUP",
						"The Daily Show 2024-09-21.Guest.1080p.AMZN.WEB-DL.DDP.5.1.H.264-GROUP1",
					},
					releaseTitle: "The Daily Show 2024-09-22 Other Guest 1080p AMZN WEB-DL DDP 5.1 H.264-GROUP1",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Season: true, Episode: true, Year: true, Month: true, Day: true, SubTitle: true},
				},
				isDuplicate: false,
			},
			{
				name: "22",
				fields: fields{
					releaseTitles: []string{
						"That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP",
						"That.Movie.2023.BluRay.720p.x265.DTS-HD-GROUP",
						"That.Movie.2023.2160p.BluRay.DTS-HD.5.1.x265-GROUP",
					},
					releaseTitle: "That.Movie.2023.2160p.BluRay.DD.2.0.x265-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Audio: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "23",
				fields: fields{
					releaseTitles: []string{
						"That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP",
						"That.Movie.2023.BluRay.720p.x265.DTS-HD-GROUP",
						"That.Movie.2023.2160p.BluRay.DTS-HD.5.1.x265-GROUP",
					},
					releaseTitle: "That.Movie.2023.2160p.BluRay.DTS-HD.5.1.x265-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Audio: true, Group: true},
				},
				isDuplicate: true,
			},
			{
				name: "24",
				fields: fields{
					releaseTitles: []string{
						"That.Movie.2023.BluRay.2160p.x265.DTS-HD-GROUP",
						"That.Movie.2023.BluRay.720p.x265.DTS-HD-GROUP",
						"That.Movie.2023.2160p.BluRay.DD.5.1.x265-GROUP",
					},
					releaseTitle: "That.Movie.2023.2160p.BluRay.AC3.5.1.x265-GROUP",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Audio: true, Group: true},
				},
				isDuplicate: true,
			},
			// FIXME this is now a match after we commented out the media_processing check (REMUX).
			// TODO add MediaProcessing toggle to DuplicateProfile and/or start to parse things into Quality, tbd
			//{
			//	name: "25",
			//	fields: fields{
			//		releaseTitles: []string{
			//			//"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
			//			"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC-FraMeSToR",
			//		},
			//		releaseTitle: "Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
			//		profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Audio: true, Group: true},
			//	},
			//	isDuplicate: false,
			//},
			{
				name: "26",
				fields: fields{
					releaseTitles: []string{
						//"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 Collectors Edition UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Audio: true, Group: true, Hybrid: true},
				},
				isDuplicate: false,
			},
			{
				name: "26_1",
				fields: fields{
					releaseTitles: []string{
						//"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 Collectors Edition UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Audio: true, Group: true, Hybrid: false},
				},
				isDuplicate: true,
			},
			{
				name: "27",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 Collectors Edition UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Edition: false, Source: true, Codec: true, Resolution: true, Audio: true, Group: true},
				},
				isDuplicate: true,
			},
			{
				name: "28",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
						"Despicable Me 4 2024 Collectors Edition UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 Collectors Edition UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Edition: true, Source: true, Codec: true, Resolution: true, Audio: true, Group: true},
				},
				isDuplicate: true,
			},
			{
				name: "29",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR10 HEVC REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR HEVC REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR DV HEVC REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, DynamicRange: true, Audio: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "30",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR10 HEVC REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR HEVC REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR DV HEVC REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, DynamicRange: true, Audio: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "31",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR10 HEVC REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR HEVC REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HEVC REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, DynamicRange: true, Audio: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "32",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR10 HEVC REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HDR HEVC REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HEVC REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, DynamicRange: true, Audio: true, Group: true},
				},
				isDuplicate: false,
			},
			{
				name: "33",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 FRENCH UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 GERMAN UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, DynamicRange: true, Audio: true, Group: true, Language: true},
				},
				isDuplicate: false,
			},
			{
				name: "34",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 FRENCH UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 GERMAN UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 GERMAN UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, DynamicRange: true, Audio: true, Group: true, Language: true},
				},
				isDuplicate: true,
			},
			{
				name: "35",
				fields: fields{
					releaseTitles: []string{
						"Despicable Me 4 2024 FRENCH UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
						"Despicable Me 4 2024 GERMAN UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 HEVC DV REMUX Hybrid-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, DynamicRange: true, Audio: true, Group: true, Language: true},
				},
				isDuplicate: false,
			},
			{
				name: "36",
				fields: fields{
					releaseTitles: []string{
						"Road House 1989 1080p GER Blu-ray AVC LPCM 2.0-MONUMENT",
					},
					releaseTitle: "Road House 1989 1080p Blu-ray AVC LPCM 2.0-MONUMENT",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Group: true, Language: true},
				},
				isDuplicate: false,
			},
			{
				name: "37",
				fields: fields{
					releaseTitles: []string{
						"Road House 1989 1080p ITA Blu-ray AVC LPCM 2.0-MONUMENT",
						"Road House 1989 1080p GER Blu-ray AVC LPCM 2.0-MONUMENT",
					},
					releaseTitle: "Road House 1989 1080p NOR Blu-ray AVC LPCM 2.0-MONUMENT",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Group: true, Language: true},
				},
				isDuplicate: false,
			},
			{
				name: "38",
				fields: fields{
					releaseTitles: []string{
						"Road House 1989 1080p GER Blu-ray AVC LPCM 2.0-MONUMENT",
					},
					releaseTitle: "Road House 1989 1080p GER Blu-ray AVC LPCM 2.0-MONUMENT",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Group: true, Language: true},
				},
				isDuplicate: true,
			},
			{
				name: "39",
				fields: fields{
					releaseTitles: []string{
						"The Best Show 2020 S04E10 1080p AMZN WEB-DL DDP 5.1 H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.H.264-GROUP",
						"The.Best.Show.2020.S04E10.1080p.AMZN.WEB-DL.DDP.5.1.HDR.DV.H.264-GROUP",
					},
					releaseTitle: "The Best Show 2020 S04E10 1080p AMZN WEB-DL DDP 5.1 H.264-GROUP",
					profile:      &domain.DuplicateReleaseProfile{ReleaseName: true},
				},
				isDuplicate: true,
			},

			{
				name: "40",
				fields: fields{
					releaseTitles: []string{
						"Sisu: Road to Revenge 2025 Hybrid 2160p UHD BluRay REMUX DV HDR10+ HEVC Dubbed TrueHD 7 1 Atmos-HDT",
						//"Sisu: Road to Revenge 2025 2160p UHD Blu-ray DV HDR HEVC TrueHD 7 1 Atmos-MTeam",
						//"Sisu Road to Revenge 2025 MULTi iNTERNAL UHD BluRay 2160p TrueHD Atmos 7 1 DV HDR10 REMUX-seedpool",
					},
					releaseTitle: "Sisu Road to Revenge 2025 MULTi iNTERNAL UHD BluRay 2160p TrueHD Atmos 7 1 DV HDR10 REMUX-seedpool",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Resolution: true},
				},
				isDuplicate: true,
			},
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("Check_Is_Duplicate_Release %s [%s]", tt.name, dbType), func(t *testing.T) {
				ctx := context.Background()

				// Setup
				for _, rel := range tt.fields.releaseTitles {
					mockRel := domain.NewRelease(mockIndexer)
					mockRel.ParseString(rel)

					mockRel.FilterID = filterMock.ID

					err = releaseRepo.Store(ctx, mockRel)
					assert.NoError(t, err)

					ras := &domain.ReleaseActionStatus{
						ID:         0,
						Status:     domain.ReleasePushStatusApproved,
						Action:     "test",
						ActionID:   int64(actionMock.ID),
						Type:       domain.ActionTypeTest,
						Client:     "",
						Filter:     "Test filter",
						FilterID:   int64(filterMock.ID),
						Rejections: []string{},
						ReleaseID:  mockRel.ID,
						Timestamp:  time.Now(),
					}

					err = releaseRepo.StoreReleaseActionStatus(ctx, ras)
					assert.NoError(t, err)
				}

				releases, err := releaseRepo.Find(ctx, domain.ReleaseQueryParams{})
				assert.NoError(t, err)
				assert.Len(t, releases.Data, len(tt.fields.releaseTitles))

				compareRel := domain.NewRelease(mockIndexer)
				compareRel.ParseString(tt.fields.releaseTitle)

				// Execute
				isDuplicate, err := releaseRepo.CheckIsDuplicateRelease(ctx, tt.fields.profile, compareRel)

				// Verify
				assert.NoError(t, err)
				assert.Equal(t, tt.isDuplicate, isDuplicate)

				// Cleanup
				_ = releaseRepo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			})
		}

		// Cleanup
		//_ = releaseRepo.Delete(context.Background(), &domain.DeleteReleaseRequest{OlderThan: 0})
		_ = actionRepo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: actionMock.ID})
		_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
	}
}

func getMockReleaseCleanupJob() *domain.ReleaseCleanupJob {
	return &domain.ReleaseCleanupJob{
		Name:          "Test Cleanup Job",
		Enabled:       true,
		Schedule:      "0 3 * * *",
		OlderThan:     720,
		Indexers:      "btn,ptp",
		Statuses:      "PUSH_REJECTED,PUSH_ERROR",
		LastRun:       time.Now(),
		LastRunStatus: domain.ReleaseCleanupStatusSuccess,
		LastRunData:   `{"deleted": 10}`,
	}
}

func TestReleaseCleanupJobRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.StoreCleanupJob(context.Background(), mockData)
			assert.NoError(t, err)
			assert.NotZero(t, mockData.ID)

			// Verify
			job, err := repo.FindCleanupJobByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, job.Name)
			assert.Equal(t, mockData.Enabled, job.Enabled)
			assert.Equal(t, mockData.Schedule, job.Schedule)
			assert.Equal(t, mockData.OlderThan, job.OlderThan)
			assert.Equal(t, mockData.Indexers, job.Indexers)
			assert.Equal(t, mockData.Statuses, job.Statuses)

			// Cleanup
			_ = repo.DeleteCleanupJob(context.Background(), mockData.ID)
		})
	}
}

func TestReleaseCleanupJobRepo_FindByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreCleanupJob(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute
			job, err := repo.FindCleanupJobByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, job.Name)
			assert.Equal(t, mockData.ID, job.ID)

			// Cleanup
			_ = repo.DeleteCleanupJob(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_Not_Found [%s]", dbType), func(t *testing.T) {
			// Execute
			_, err := repo.FindCleanupJobByID(context.Background(), 99999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_List(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)

		t.Run(fmt.Sprintf("List_Returns_All_Jobs [%s]", dbType), func(t *testing.T) {
			// Setup - create multiple jobs
			job1 := getMockReleaseCleanupJob()
			job1.Name = "Job 1"
			job2 := getMockReleaseCleanupJob()
			job2.Name = "Job 2"
			job3 := getMockReleaseCleanupJob()
			job3.Name = "Job 3"

			err := repo.StoreCleanupJob(context.Background(), job1)
			assert.NoError(t, err)
			err = repo.StoreCleanupJob(context.Background(), job2)
			assert.NoError(t, err)
			err = repo.StoreCleanupJob(context.Background(), job3)
			assert.NoError(t, err)

			// Execute
			jobs, err := repo.ListCleanupJobs(context.Background())
			assert.NoError(t, err)
			assert.GreaterOrEqual(t, len(jobs), 3)

			// Verify - find our test jobs
			foundJobs := 0
			for _, job := range jobs {
				if job.Name == "Job 1" || job.Name == "Job 2" || job.Name == "Job 3" {
					foundJobs++
				}
			}
			assert.Equal(t, 3, foundJobs)

			// Cleanup
			_ = repo.DeleteCleanupJob(context.Background(), job1.ID)
			_ = repo.DeleteCleanupJob(context.Background(), job2.ID)
			_ = repo.DeleteCleanupJob(context.Background(), job3.ID)
		})

		t.Run(fmt.Sprintf("List_Empty_Table [%s]", dbType), func(t *testing.T) {
			// Execute
			jobs, err := repo.ListCleanupJobs(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, jobs)
		})
	}
}

func TestReleaseCleanupJobRepo_Update(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreCleanupJob(context.Background(), mockData)
			assert.NoError(t, err)

			// Update data
			mockData.Name = "Updated Name"
			mockData.Schedule = "0 4 * * *"
			mockData.OlderThan = 168
			mockData.Enabled = false
			mockData.Indexers = "hdt,blu"
			mockData.Statuses = "PUSH_APPROVED"

			// Execute
			err = repo.UpdateCleanupJob(context.Background(), mockData)
			assert.NoError(t, err)

			// Verify
			updatedJob, err := repo.FindCleanupJobByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, "Updated Name", updatedJob.Name)
			assert.Equal(t, "0 4 * * *", updatedJob.Schedule)
			assert.Equal(t, 168, updatedJob.OlderThan)
			assert.Equal(t, false, updatedJob.Enabled)
			assert.Equal(t, "hdt,blu", updatedJob.Indexers)
			assert.Equal(t, "PUSH_APPROVED", updatedJob.Statuses)

			// Cleanup
			_ = repo.DeleteCleanupJob(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Setup
			nonExistingJob := getMockReleaseCleanupJob()
			nonExistingJob.ID = 99999

			// Execute
			err := repo.UpdateCleanupJob(context.Background(), nonExistingJob)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_UpdateLastRun(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("UpdateLastRun_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreCleanupJob(context.Background(), mockData)
			assert.NoError(t, err)

			// Update last run data
			newLastRun := time.Now().Add(-1 * time.Hour)
			mockData.LastRun = newLastRun
			mockData.LastRunStatus = domain.ReleaseCleanupStatusError
			mockData.LastRunData = `{"error": "test error"}`

			// Execute
			err = repo.UpdateCleanupJobLastRun(context.Background(), mockData)
			assert.NoError(t, err)

			// Verify
			updatedJob, err := repo.FindCleanupJobByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, domain.ReleaseCleanupStatusError, updatedJob.LastRunStatus)
			assert.Equal(t, `{"error": "test error"}`, updatedJob.LastRunData)
			assert.WithinDuration(t, newLastRun, updatedJob.LastRun, 2*time.Second)

			// Cleanup
			_ = repo.DeleteCleanupJob(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("UpdateLastRun_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Setup
			nonExistingJob := getMockReleaseCleanupJob()
			nonExistingJob.ID = 99999

			// Execute
			err := repo.UpdateCleanupJobLastRun(context.Background(), nonExistingJob)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_ToggleEnabled(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData.Enabled = true
			err := repo.StoreCleanupJob(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute - disable
			err = repo.CleanupJobToggleEnabled(context.Background(), mockData.ID, false)
			assert.NoError(t, err)

			// Verify
			job, err := repo.FindCleanupJobByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.False(t, job.Enabled)

			// Execute - enable
			err = repo.CleanupJobToggleEnabled(context.Background(), mockData.ID, true)
			assert.NoError(t, err)

			// Verify
			job, err = repo.FindCleanupJobByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.True(t, job.Enabled)

			// Cleanup
			_ = repo.DeleteCleanupJob(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.CleanupJobToggleEnabled(context.Background(), 99999, false)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreCleanupJob(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute
			err = repo.DeleteCleanupJob(context.Background(), mockData.ID)
			assert.NoError(t, err)

			// Verify
			_, err = repo.FindCleanupJobByID(context.Background(), mockData.ID)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})

		t.Run(fmt.Sprintf("Delete_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.DeleteCleanupJob(context.Background(), 99999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}
