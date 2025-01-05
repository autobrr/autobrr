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

func getMockFeed() *domain.Feed {
	settings := &domain.FeedSettingsJSON{
		DownloadType: domain.FeedDownloadTypeTorrent,
	}

	return &domain.Feed{
		Name:      "ExampleFeed",
		Type:      "RSS",
		Enabled:   true,
		URL:       "https://example.com/feed",
		Interval:  15,
		Timeout:   30,
		ApiKey:    "API_KEY_HERE",
		IndexerID: 1,
		Settings:  settings,
	}
}

func TestFeedRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			// Execute
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Verify
			feed, err := repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, feed.Name)
			assert.Equal(t, mockData.Type, feed.Type)
			assert.Equal(t, mockData.Enabled, feed.Enabled)
			assert.Equal(t, mockData.URL, feed.URL)
			assert.Equal(t, mockData.Interval, feed.Interval)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("Store_Fails_Missing_Wrong_Foreign_Key [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.Store(context.Background(), mockData)
			assert.Error(t, err)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})
	}
}

func TestFeedRepo_Update(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Update data
			mockData.Name = "NewName"
			mockData.Type = "NewType"

			// Execute
			err = repo.Update(context.Background(), mockData)
			assert.NoError(t, err)

			// Verify
			updatedFeed, err := repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)
			assert.Equal(t, "NewName", updatedFeed.Name)
			assert.Equal(t, "NewType", updatedFeed.Type)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("Update_Fails_Non_Existing_Feed [%s]", dbType), func(t *testing.T) {
			// Setup
			nonExistingFeed := getMockFeed()
			nonExistingFeed.ID = 9999

			// Execute
			err := repo.Update(context.Background(), nonExistingFeed)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "sql: no rows in result set")
		})

	}
}

func TestFeedRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute
			err = repo.Delete(context.Background(), mockData.ID)
			assert.NoError(t, err)

			// Verify
			_, err = repo.FindByID(context.Background(), mockData.ID)
			assert.Error(t, err)

			// Cleanup
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("Delete_Fails_Non_Existing_Feed [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.Delete(context.Background(), 9999)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "sql: no rows in result set")
		})
	}
}

func TestFeedRepo_FindByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute
			feed, err := repo.FindByID(context.Background(), mockData.ID)
			assert.NoError(t, err)

			// Verify
			assert.Equal(t, mockData.Name, feed.Name)
			assert.Equal(t, mockData.Type, feed.Type)
			assert.Equal(t, mockData.Enabled, feed.Enabled)
			assert.Equal(t, mockData.URL, feed.URL)
			assert.Equal(t, mockData.Interval, feed.Interval)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("FindByID_Fails_Wrong_ID [%s]", dbType), func(t *testing.T) {
			// Execute
			feed, err := repo.FindByID(context.Background(), -1)
			assert.Error(t, err)
			assert.Nil(t, feed)
		})

	}
}

func TestFeedRepo_FindOne(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("FindByIndexerIdentifier_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute
			feed, err := repo.FindOne(context.Background(), domain.FindOneParams{IndexerIdentifier: indexer.Identifier})
			assert.NoError(t, err)

			// Verify
			assert.NotNil(t, feed)
			assert.Equal(t, mockData.Name, feed.Name)
			assert.Equal(t, mockData.Type, feed.Type)
			assert.Equal(t, mockData.Enabled, feed.Enabled)
			assert.Equal(t, mockData.URL, feed.URL)
			assert.Equal(t, mockData.Interval, feed.Interval)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("FindByIndexerIdentifier_Fails_Wrong_Identifier [%s]", dbType), func(t *testing.T) {
			// Execute
			feed, err := repo.FindOne(context.Background(), domain.FindOneParams{IndexerIdentifier: "wrong-identifier"})
			assert.Error(t, err)
			assert.Nil(t, feed)
		})
	}
}

func TestFeedRepo_Find(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		indexerMockData := getMockIndexer()
		feedMockData1 := getMockFeed()
		feedMockData2 := getMockFeed()
		// Change some values in feedMockData2 for variety
		feedMockData2.Name = "Different Feed"
		feedMockData2.URL = "http://different.url"

		t.Run(fmt.Sprintf("Find_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			feedMockData1.IndexerID = int(indexer.ID)
			feedMockData2.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), feedMockData1)
			assert.NoError(t, err)
			err = repo.Store(context.Background(), feedMockData2)
			assert.NoError(t, err)

			// Execute
			feeds, err := repo.Find(context.Background())
			assert.NoError(t, err)

			// Verify
			assert.Len(t, feeds, 2)

			// Cleanup
			for _, feed := range feeds {
				_ = repo.Delete(context.Background(), feed.ID)
			}
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("Find_Fails_EmptyDB [%s]", dbType), func(t *testing.T) {
			// Execute
			feeds, err := repo.Find(context.Background())

			// Verify
			assert.NoError(t, err)
			assert.Empty(t, feeds)
		})

	}
}

func TestFeedRepo_GetLastRunDataByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		indexerMockData := getMockIndexer()
		feedMockData := getMockFeed()
		feedMockData.LastRunData = "Some data"

		t.Run(fmt.Sprintf("GetLastRunDataByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			feedMockData.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), feedMockData)
			assert.NoError(t, err)
			err = repo.UpdateLastRunWithData(context.Background(), feedMockData.ID, feedMockData.LastRunData)
			assert.NoError(t, err)
			// Execute
			data, err := repo.GetLastRunDataByID(context.Background(), feedMockData.ID)
			assert.NoError(t, err)

			// Verify
			assert.Equal(t, "Some data", data)

			// Cleanup
			_ = repo.Delete(context.Background(), feedMockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("GetLastRunDataByID_Fails_InvalidID [%s]", dbType), func(t *testing.T) {
			// Execute
			_, err := repo.GetLastRunDataByID(context.Background(), -1)

			// Verify
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("GetLastRunDataByID_Fails_NullData [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			feedMockData.IndexerID = int(indexer.ID)
			feedMockData.LastRunData = ""
			err = repo.Store(context.Background(), feedMockData)
			assert.NoError(t, err)

			// Execute
			data, err := repo.GetLastRunDataByID(context.Background(), feedMockData.ID)
			assert.NoError(t, err)

			// Verify
			assert.Empty(t, data)

			// Cleanup
			_ = repo.Delete(context.Background(), feedMockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})
	}
}

func TestFeedRepo_UpdateLastRun(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		indexerMockData := getMockIndexer()
		feedMockData := getMockFeed()

		t.Run(fmt.Sprintf("UpdateLastRun_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			feedMockData.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), feedMockData)
			assert.NoError(t, err)

			// Execute
			err = repo.UpdateLastRun(context.Background(), feedMockData.ID)
			assert.NoError(t, err)

			// Verify
			updatedFeed, err := repo.Find(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, updatedFeed)
			assert.True(t, updatedFeed[0].LastRun.After(time.Now().Add(-1*time.Minute)))

			// Cleanup
			_ = repo.Delete(context.Background(), feedMockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("UpdateLastRun_Fails_InvalidID [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.UpdateLastRun(context.Background(), -1)

			// Verify
			assert.Error(t, err)
		})
	}
}

func TestFeedRepo_UpdateLastRunWithData(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		indexerMockData := getMockIndexer()
		feedMockData := getMockFeed()

		t.Run(fmt.Sprintf("UpdateLastRunWithData_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			feedMockData.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), feedMockData)
			assert.NoError(t, err)

			// Execute
			err = repo.UpdateLastRunWithData(context.Background(), feedMockData.ID, "newData")
			assert.NoError(t, err)

			// Verify
			updatedFeed, err := repo.Find(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, updatedFeed)
			assert.True(t, updatedFeed[0].LastRun.After(time.Now().Add(-1*time.Minute)))
			assert.Equal(t, "newData", updatedFeed[0].LastRunData)

			// Cleanup
			_ = repo.Delete(context.Background(), feedMockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("UpdateLastRunWithData_Fails_InvalidID [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.UpdateLastRunWithData(context.Background(), -1, "data")

			// Verify
			assert.Error(t, err)
		})
	}
}

func TestFeedRepo_ToggleEnabled(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		indexerMockData := getMockIndexer()
		feedMockData := getMockFeed()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(context.Background(), indexerMockData)
			assert.NoError(t, err)
			feedMockData.IndexerID = int(indexer.ID)
			err = repo.Store(context.Background(), feedMockData)
			assert.NoError(t, err)

			// Execute & Verify
			err = repo.ToggleEnabled(context.Background(), feedMockData.ID, false)
			assert.NoError(t, err)
			updatedFeed, err := repo.FindByID(context.Background(), feedMockData.ID)
			assert.NoError(t, err)
			assert.NotNil(t, updatedFeed)
			assert.False(t, updatedFeed.Enabled)

			// Cleanup
			_ = repo.Delete(context.Background(), feedMockData.ID)
			_ = indexerRepo.Delete(context.Background(), int(indexer.ID))
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_InvalidID [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.ToggleEnabled(context.Background(), -1, true)

			// Verify
			assert.Error(t, err)
		})
	}
}
