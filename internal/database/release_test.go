// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
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
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("StoreReleaseActionStatus_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(ctx, actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(ctx, releaseActionMockData)
			assert.NoError(t, err)

			// Verify
			assert.NotEqual(t, int64(0), mockData.ID)

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_StoreReleaseActionStatus(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("StoreReleaseActionStatus_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(ctx, actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(ctx, releaseActionMockData)
			assert.NoError(t, err)

			// Verify
			assert.NotEqual(t, int64(0), releaseActionMockData.ID)

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_Find(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		//actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		//releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("FindReleases_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(ctx, mockData)
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

			resp, err := repo.Find(ctx, queryParams)

			// Verify
			assert.NotNil(t, resp)
			assert.NotEqual(t, int64(0), resp.TotalCount)
			assert.True(t, resp.NextCursor >= 0)

			// Search by type
			queryParams.Search = "type:movie"
			resp, err = repo.Find(ctx, queryParams)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, uint64(1), resp.TotalCount)

			// Search by type with no matches
			queryParams.Search = "type:episode"
			resp, err = repo.Find(ctx, queryParams)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			assert.Equal(t, uint64(0), resp.TotalCount)

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_FindRecent(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		//actionRepo := NewActionRepo(log, db, downloadClientRepo)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		//releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("FindRecent_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Execute
			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)

			resp, err := repo.Find(ctx, domain.ReleaseQueryParams{Limit: 10})

			// Verify
			assert.NotNil(t, resp.Data)
			assert.Lenf(t, resp.Data, 1, "Expected 1 release, got %d", len(resp.Data))

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_GetIndexerOptions(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("GetIndexerOptions_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(ctx, actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(ctx, releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			options, err := repo.GetIndexerOptions(ctx)

			// Verify
			assert.NotNil(t, options)
			assert.Len(t, options, 1)

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_GetActionStatusByReleaseID(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("GetActionStatusByReleaseID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(ctx, actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(ctx, releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			actionStatus, err := repo.GetActionStatus(ctx, &domain.GetReleaseActionStatusRequest{Id: int(releaseActionMockData.ID)})

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, actionStatus)
			assert.Equal(t, releaseActionMockData.ID, actionStatus.ID)

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_Get(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		mockData.Title = "Troy"
		mockData.NormalizedHash = "troy-2004"
		mockData.Year = 2004
		mockData.Month = 5
		mockData.Day = 14
		mockData.Audio = []string{"DTS-HD MA", "Atmos"}
		mockData.AudioChannels = "7.1"
		mockData.Region = "US"
		mockData.Language = []string{"English", "French"}
		mockData.Cut = []string{"Director's Cut"}
		mockData.Edition = []string{"Extended"}
		mockData.Hybrid = true
		mockData.Repack = true
		mockData.MediaProcessing = "Remux"
		mockData.Other = []string{"Remastered"}
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("Get_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(ctx, actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(ctx, releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			release, err := repo.Get(ctx, &domain.GetReleaseRequest{Id: int(mockData.ID)})

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, release)
			assert.Equal(t, mockData.ID, release.ID)
			assert.Equal(t, mockData.Title, release.Title)
			assert.Equal(t, mockData.NormalizedHash, release.NormalizedHash)
			assert.Equal(t, mockData.Season, release.Season)
			assert.Equal(t, mockData.Episode, release.Episode)
			assert.Equal(t, mockData.Year, release.Year)
			assert.Equal(t, mockData.Month, release.Month)
			assert.Equal(t, mockData.Day, release.Day)
			assert.Equal(t, mockData.Resolution, release.Resolution)
			assert.Equal(t, mockData.Source, release.Source)
			assert.Equal(t, mockData.Codec, release.Codec)
			assert.Equal(t, mockData.Container, release.Container)
			assert.Equal(t, mockData.HDR, release.HDR)
			assert.Equal(t, mockData.Audio, release.Audio)
			assert.Equal(t, mockData.AudioChannels, release.AudioChannels)
			assert.Equal(t, mockData.Group, release.Group)
			assert.Equal(t, mockData.Region, release.Region)
			assert.Equal(t, mockData.Language, release.Language)
			assert.Equal(t, mockData.Cut, release.Cut)
			assert.Equal(t, mockData.Edition, release.Edition)
			assert.Equal(t, mockData.Hybrid, release.Hybrid)
			assert.Equal(t, mockData.Proper, release.Proper)
			assert.Equal(t, mockData.Repack, release.Repack)
			assert.Equal(t, mockData.Website, release.Website)
			assert.Equal(t, mockData.MediaProcessing, release.MediaProcessing)
			assert.Equal(t, mockData.Type, release.Type)
			assert.Equal(t, mockData.Origin, release.Origin)
			assert.Equal(t, mockData.Tags, release.Tags)
			assert.Equal(t, mockData.PreTime, release.PreTime)
			assert.Equal(t, mockData.Other, release.Other)

			episode := &domain.Release{
				Rejections: []string{},
				Timestamp:  time.Now(),
				Title:      "Example Show",
				Season:     4,
				Episode:    7,
				Type:       rls.Episode,
				Tags:       []string{},
				Other:      []string{},
				FilterID:   createdFilters[0].ID,
			}

			err = repo.Store(ctx, episode)
			assert.NoError(t, err)

			_, err = db.squirrel.
				Update("release").
				Set("year", nil).
				Set("resolution", nil).
				Set("codec", nil).
				Set("hdr", nil).
				Set("audio", nil).
				Set("language", nil).
				Set("cut", nil).
				Set("edition", nil).
				Set("proper", nil).
				Set("repack", nil).
				Set("hybrid", nil).
				Where("id = ?", episode.ID).
				RunWith(db.Handler).
				ExecContext(t.Context())
			assert.NoError(t, err)

			storedEpisode, err := repo.Get(ctx, &domain.GetReleaseRequest{Id: int(episode.ID)})
			assert.NoError(t, err)
			assert.NotNil(t, storedEpisode)
			assert.Equal(t, episode.Season, storedEpisode.Season)
			assert.Equal(t, episode.Episode, storedEpisode.Episode)
			assert.Equal(t, episode.Type, storedEpisode.Type)
			assert.Zero(t, storedEpisode.Year)
			assert.Empty(t, storedEpisode.Resolution)
			assert.Empty(t, storedEpisode.Codec)
			assert.Empty(t, storedEpisode.HDR)
			assert.Empty(t, storedEpisode.Audio)
			assert.Empty(t, storedEpisode.Language)
			assert.Empty(t, storedEpisode.Cut)
			assert.Empty(t, storedEpisode.Edition)
			assert.False(t, storedEpisode.Proper)
			assert.False(t, storedEpisode.Repack)
			assert.False(t, storedEpisode.Hybrid)
			assert.Empty(t, storedEpisode.Tags)
			assert.Empty(t, storedEpisode.Other)

			missing, err := repo.Get(ctx, &domain.GetReleaseRequest{Id: -1})
			assert.Nil(t, missing)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_Stats(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("Stats_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(ctx, actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(ctx, releaseActionMockData)
			assert.NoError(t, err)

			// Execute
			stats, err := repo.Stats(ctx)

			// Verify
			assert.NoError(t, err)
			assert.NotNil(t, stats)
			assert.Equal(t, int64(1), stats.TotalCount)
			assert.Equal(t, int64(1), stats.FilteredCount)
			assert.Equal(t, int64(0), stats.FilterRejectedCount)
			assert.Equal(t, int64(1), stats.PushApprovedCount)
			assert.Equal(t, int64(0), stats.PushRejectedCount)
			assert.Equal(t, int64(0), stats.PushErrorCount)

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_StatsDashboard(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("StatsDashboard_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(ctx, actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(ctx, releaseActionMockData)
			assert.NoError(t, err)

			for _, days := range []int{30, 0} {
				activity, err := repo.StatsActivity(ctx, days)
				assert.NoError(t, err)
				assert.Equal(t, days, activity.Days)
				assert.NotEmpty(t, activity.Daily)
				today := activity.Daily[len(activity.Daily)-1]
				assert.Equal(t, int64(1), today.MatchedCount)
				assert.Equal(t, int64(1), today.PushApprovedCount)
				assert.Equal(t, int64(0), today.PushRejectedCount)

				volume, err := repo.StatsVolume(ctx, days)
				assert.NoError(t, err)
				assert.NotEmpty(t, volume.Daily)

				heatmap, err := repo.StatsHeatmap(ctx, days)
				assert.NoError(t, err)
				assert.Len(t, heatmap.Heatmap, 168)
				var heatmapTotal int64
				for _, count := range heatmap.Heatmap {
					heatmapTotal += count
				}
				assert.Equal(t, int64(1), heatmapTotal)

				indexers, err := repo.StatsTopIndexers(ctx, days)
				assert.NoError(t, err)
				assert.Len(t, indexers.Top, 1)
				assert.Equal(t, int64(1), indexers.Top[0].MatchedCount)
				assert.Equal(t, int64(1), indexers.Top[0].PushApprovedCount)

				filters, err := repo.StatsTopFilters(ctx, days)
				assert.NoError(t, err)
				assert.NotEmpty(t, filters.Top)
			}

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
		})
	}
}

func TestReleaseRepo_Delete(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		// Setup shared dependencies
		mock := getMockDownloader()
		err := downloadClientRepo.Store(ctx, &mock)
		assert.NoError(t, err)

		err = filterRepo.Store(ctx, getMockFilter())
		assert.NoError(t, err)

		createdFilters, err := filterRepo.ListFilters(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, createdFilters)

		actionMock := getMockAction()
		actionMock.FilterID = createdFilters[0].ID
		actionMock.ClientID = mock.ID
		err = actionRepo.Store(ctx, actionMock)
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
					for range 3 {
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
		_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMock.ID})
		_ = filterRepo.Delete(ctx, createdFilters[0].ID)
		_ = downloadClientRepo.Delete(ctx, mock.ID)
	}
}

func TestReleaseRepo_CheckSmartEpisodeCanDownloadShow(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		downloadClientRepo := NewDownloaderRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		repo := NewReleaseRepo(log, db)

		mockData := getMockRelease()
		releaseActionMockData := getMockReleaseActionStatus()
		actionMockData := getMockAction()

		t.Run(fmt.Sprintf("Check_Smart_Episode_Can_Download [%s]", dbType), func(t *testing.T) {
			// Setup
			mock := getMockDownloader()
			err := downloadClientRepo.Store(ctx, &mock)
			assert.NoError(t, err)
			assert.NotNil(t, mock)

			err = filterRepo.Store(ctx, getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			actionMockData.FilterID = createdFilters[0].ID
			actionMockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			err = repo.Store(ctx, mockData)
			assert.NoError(t, err)
			err = actionRepo.Store(ctx, actionMockData)
			assert.NoError(t, err)

			releaseActionMockData.ReleaseID = mockData.ID
			releaseActionMockData.ActionID = int64(actionMockData.ID)
			releaseActionMockData.FilterID = int64(createdFilters[0].ID)

			err = repo.StoreReleaseActionStatus(ctx, releaseActionMockData)
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
			canDownload, err := repo.CheckSmartEpisodeCanDownload(ctx, params)

			// Verify
			assert.NoError(t, err)
			assert.True(t, canDownload)

			// Cleanup
			_ = repo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMockData.ID})
			_ = filterRepo.Delete(ctx, createdFilters[0].ID)
			_ = downloadClientRepo.Delete(ctx, mock.ID)
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
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()

		filterRepo := NewFilterRepo(log, db)
		actionRepo := NewActionRepo(log, db)
		releaseRepo := NewReleaseRepo(log, db)

		// reset
		//db.Handler.Exec("DELETE FROM release")
		//db.Handler.Exec("DELETE FROM action")
		//db.Handler.Exec("DELETE FROM release_action_status")

		mockIndexer := domain.IndexerMinimal{ID: 0, Name: "Mock", Identifier: "mock", IdentifierExternal: "Mock"}
		actionMock := &domain.Action{Name: "Test", Type: domain.ActionTypeTest, Enabled: true}
		filterMock := getMockFilterDuplicates()

		// Setup
		err := filterRepo.Store(ctx, filterMock)
		assert.NoError(t, err)

		createdFilters, err := filterRepo.ListFilters(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, createdFilters)

		actionMock.FilterID = filterMock.ID

		err = actionRepo.Store(ctx, actionMock)
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
			{
				name: "25",
				fields: fields{
					releaseTitles: []string{
						//"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
						"Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC-FraMeSToR",
					},
					releaseTitle: "Despicable Me 4 2024 UHD BluRay 2160p TrueHD Atmos 7.1 DV HEVC REMUX-FraMeSToR",
					profile:      &domain.DuplicateReleaseProfile{Title: true, Year: true, Source: true, Codec: true, Resolution: true, Audio: true, Group: true},
				},
				isDuplicate: false,
			},
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
		}

		for _, tt := range tests {
			t.Run(fmt.Sprintf("Check_Is_Duplicate_Release %s [%s]", tt.name, dbType), func(t *testing.T) {

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
		//_ = releaseRepo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
		_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: actionMock.ID})
		_ = filterRepo.Delete(ctx, createdFilters[0].ID)
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
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.StoreCleanupJob(ctx, mockData)
			assert.NoError(t, err)
			assert.NotZero(t, mockData.ID)

			// Verify
			job, err := repo.FindCleanupJobByID(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, job.Name)
			assert.Equal(t, mockData.Enabled, job.Enabled)
			assert.Equal(t, mockData.Schedule, job.Schedule)
			assert.Equal(t, mockData.OlderThan, job.OlderThan)
			assert.Equal(t, mockData.Indexers, job.Indexers)
			assert.Equal(t, mockData.Statuses, job.Statuses)

			// Cleanup
			_ = repo.DeleteCleanupJob(ctx, mockData.ID)
		})
	}
}

func TestReleaseCleanupJobRepo_FindByID(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreCleanupJob(ctx, mockData)
			assert.NoError(t, err)

			// Execute
			job, err := repo.FindCleanupJobByID(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, job.Name)
			assert.Equal(t, mockData.ID, job.ID)

			// Cleanup
			_ = repo.DeleteCleanupJob(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_Not_Found [%s]", dbType), func(t *testing.T) {
			// Execute
			_, err := repo.FindCleanupJobByID(ctx, 99999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_List(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
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

			err := repo.StoreCleanupJob(ctx, job1)
			assert.NoError(t, err)
			err = repo.StoreCleanupJob(ctx, job2)
			assert.NoError(t, err)
			err = repo.StoreCleanupJob(ctx, job3)
			assert.NoError(t, err)

			// Execute
			jobs, err := repo.ListCleanupJobs(ctx)
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
			_ = repo.DeleteCleanupJob(ctx, job1.ID)
			_ = repo.DeleteCleanupJob(ctx, job2.ID)
			_ = repo.DeleteCleanupJob(ctx, job3.ID)
		})

		t.Run(fmt.Sprintf("List_Empty_Table [%s]", dbType), func(t *testing.T) {
			// Execute
			jobs, err := repo.ListCleanupJobs(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, jobs)
		})
	}
}

func TestReleaseCleanupJobRepo_Update(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreCleanupJob(ctx, mockData)
			assert.NoError(t, err)

			// Update data
			mockData.Name = "Updated Name"
			mockData.Schedule = "0 4 * * *"
			mockData.OlderThan = 168
			mockData.Enabled = false
			mockData.Indexers = "hdt,blu"
			mockData.Statuses = "PUSH_APPROVED"

			// Execute
			err = repo.UpdateCleanupJob(ctx, mockData)
			assert.NoError(t, err)

			// Verify
			updatedJob, err := repo.FindCleanupJobByID(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, "Updated Name", updatedJob.Name)
			assert.Equal(t, "0 4 * * *", updatedJob.Schedule)
			assert.Equal(t, 168, updatedJob.OlderThan)
			assert.Equal(t, false, updatedJob.Enabled)
			assert.Equal(t, "hdt,blu", updatedJob.Indexers)
			assert.Equal(t, "PUSH_APPROVED", updatedJob.Statuses)

			// Cleanup
			_ = repo.DeleteCleanupJob(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Setup
			nonExistingJob := getMockReleaseCleanupJob()
			nonExistingJob.ID = 99999

			// Execute
			err := repo.UpdateCleanupJob(ctx, nonExistingJob)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_UpdateLastRun(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("UpdateLastRun_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreCleanupJob(ctx, mockData)
			assert.NoError(t, err)

			// Update last run data
			newLastRun := time.Now().Add(-1 * time.Hour)
			mockData.LastRun = newLastRun
			mockData.LastRunStatus = domain.ReleaseCleanupStatusError
			mockData.LastRunData = `{"error": "test error"}`

			// Execute
			err = repo.UpdateCleanupJobLastRun(ctx, mockData)
			assert.NoError(t, err)

			// Verify
			updatedJob, err := repo.FindCleanupJobByID(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, domain.ReleaseCleanupStatusError, updatedJob.LastRunStatus)
			assert.Equal(t, `{"error": "test error"}`, updatedJob.LastRunData)
			assert.WithinDuration(t, newLastRun, updatedJob.LastRun, 2*time.Second)

			// Cleanup
			_ = repo.DeleteCleanupJob(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("UpdateLastRun_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Setup
			nonExistingJob := getMockReleaseCleanupJob()
			nonExistingJob.ID = 99999

			// Execute
			err := repo.UpdateCleanupJobLastRun(ctx, nonExistingJob)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_ToggleEnabled(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData.Enabled = true
			err := repo.StoreCleanupJob(ctx, mockData)
			assert.NoError(t, err)

			// Execute - disable
			err = repo.CleanupJobToggleEnabled(ctx, mockData.ID, false)
			assert.NoError(t, err)

			// Verify
			job, err := repo.FindCleanupJobByID(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.False(t, job.Enabled)

			// Execute - enable
			err = repo.CleanupJobToggleEnabled(ctx, mockData.ID, true)
			assert.NoError(t, err)

			// Verify
			job, err = repo.FindCleanupJobByID(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.True(t, job.Enabled)

			// Cleanup
			_ = repo.DeleteCleanupJob(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.CleanupJobToggleEnabled(ctx, 99999, false)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestReleaseCleanupJobRepo_Delete(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewReleaseRepo(log, db)
		mockData := getMockReleaseCleanupJob()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.StoreCleanupJob(ctx, mockData)
			assert.NoError(t, err)

			// Execute
			err = repo.DeleteCleanupJob(ctx, mockData.ID)
			assert.NoError(t, err)

			// Verify
			_, err = repo.FindCleanupJobByID(ctx, mockData.ID)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})

		t.Run(fmt.Sprintf("Delete_Fails_Non_Existing_Job [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.DeleteCleanupJob(ctx, 99999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}
