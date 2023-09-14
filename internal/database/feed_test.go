package database

import (
	"context"
	"fmt"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/stretchr/testify/assert"
	"testing"
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
	for _, dbType := range getDbs() {
		db := SetupDatabase(t, dbType)
		defer func(db *DB) {
			err := db.Close()
			if err != nil {
				t.Fatalf("Could not close db connection: %v", err)
			}
		}(db)
		log := SetupLogger()
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
	for _, dbType := range getDbs() {
		db := SetupDatabase(t, dbType)
		defer func(db *DB) {
			err := db.Close()
			if err != nil {
				t.Fatalf("Could not close db connection: %v", err)
			}
		}(db)
		log := SetupLogger()
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
	for _, dbType := range getDbs() {
		db := SetupDatabase(t, dbType)
		defer func(db *DB) {
			err := db.Close()
			if err != nil {
				t.Fatalf("Could not close db connection: %v", err)
			}
		}(db)
		log := SetupLogger()
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
