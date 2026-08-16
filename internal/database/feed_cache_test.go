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

func TestFeedCacheRepo_Get(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db

		log := setupLoggerForTest()
		repo := NewFeedCacheRepo(log, db)
		feedRepo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Get_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			// Execute
			value, err := repo.Get(mockData.ID, "test_key")
			assert.NoError(t, err)
			assert.Equal(t, []byte("test_value"), value)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID, "test_key")
		})

		t.Run(fmt.Sprintf("Get_Fails_NoRows [%s]", dbType), func(t *testing.T) {
			// Execute
			value, err := repo.Get(-1, "non_existent_key")
			assert.NoError(t, err)
			assert.Nil(t, value)
		})

		t.Run(fmt.Sprintf("Get_Fails_Foreign_Key_Constraint [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Put(999, "bad_foreign_key", []byte("test_value"), time.Now().Add(-time.Hour))
			assert.Error(t, err)

			// Execute
			value, err := repo.Get(999, "bad_foreign_key")
			assert.NoError(t, err)
			assert.Nil(t, value)
		})
	}
}

func TestFeedCacheRepo_GetByFeed(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db

		log := setupLoggerForTest()
		repo := NewFeedCacheRepo(log, db)
		feedRepo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("GetByFeed_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			// Execute
			items, err := repo.GetByFeed(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.Len(t, items, 1)
			assert.Equal(t, "test_key", items[0].Key)
			assert.Equal(t, []byte("test_value"), items[0].Value)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID, "test_key")
		})

		t.Run(fmt.Sprintf("GetByFeed_Empty [%s]", dbType), func(t *testing.T) {
			// Execute
			items, err := repo.GetByFeed(ctx, -1)
			assert.NoError(t, err)
			assert.Empty(t, items)
		})
	}
}

func TestFeedCacheRepo_Exists(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db

		log := setupLoggerForTest()
		repo := NewFeedCacheRepo(log, db)
		feedRepo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Exists_True [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			// Execute
			exists, err := repo.Exists(mockData.ID, "test_key")
			assert.NoError(t, err)
			assert.True(t, exists)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID, "test_key")
		})

		t.Run(fmt.Sprintf("Exists_False [%s]", dbType), func(t *testing.T) {
			// Execute
			exists, err := repo.Exists(-1, "nonexistent_key")
			assert.NoError(t, err)
			assert.False(t, exists)
		})
	}
}

func TestFeedCacheRepo_ExistingItems(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db

		log := setupLoggerForTest()
		repo := NewFeedCacheRepo(log, db)
		feedRepo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("ExistingItems_SingleItem_Multi_Keys [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			keys := []string{"test_key", "test_key_2"}

			// Execute
			items, err := repo.ExistingItems(ctx, mockData.ID, keys)
			assert.NoError(t, err)
			assert.Len(t, items, 1)
			//assert.True(t, exists)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID, "test_key")
		})

		t.Run(fmt.Sprintf("ExistingItems_MultipleItems [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key_2", []byte("test_value_2"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			keys := []string{"test_key", "test_key_2"}

			// Execute
			items, err := repo.ExistingItems(ctx, mockData.ID, keys)
			assert.NoError(t, err)
			assert.Len(t, items, 2)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID, "test_key")
		})

		t.Run(fmt.Sprintf("ExistingItems_MultipleItems_Single_Key [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key_2", []byte("test_value_2"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			keys := []string{"test_key"}

			// Execute
			items, err := repo.ExistingItems(ctx, mockData.ID, keys)
			assert.NoError(t, err)
			assert.Len(t, items, 1)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID, "test_key")
		})

		t.Run(fmt.Sprintf("ExistsItems_Nonexistent_Key [%s]", dbType), func(t *testing.T) {
			// Execute
			exists, err := repo.Exists(-1, "nonexistent_key")
			assert.NoError(t, err)
			assert.False(t, exists)
		})
	}
}

func TestFeedCacheRepo_Put(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFeedCacheRepo(log, db)
		feedRepo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Put_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			// Execute
			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			// Verify
			value, err := repo.Get(mockData.ID, "test_key")
			assert.NoError(t, err)
			assert.Equal(t, []byte("test_value"), value)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID, "test_key")
		})

		t.Run(fmt.Sprintf("Put_Fails_InvalidID [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.Put(-1, "test_key", []byte("test_value"), time.Now().Add(time.Hour))

			// Verify
			assert.Error(t, err)
		})
	}
}

func TestFeedCacheRepo_Delete(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db

		log := setupLoggerForTest()
		repo := NewFeedCacheRepo(log, db)
		feedRepo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			// Execute
			err = repo.Delete(ctx, mockData.ID, "test_key")
			assert.NoError(t, err)

			// Verify
			exists, err := repo.Exists(mockData.ID, "test_key")
			assert.NoError(t, err)
			assert.False(t, exists)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
		})

		t.Run(fmt.Sprintf("Delete_Fails_NoRecord [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.Delete(ctx, -1, "nonexistent_key")

			// Verify
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestFeedCacheRepo_DeleteByFeed(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFeedCacheRepo(log, db)
		feedRepo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("DeleteByFeed_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			err = repo.Put(mockData.ID, "test_key", []byte("test_value"), time.Now().Add(time.Hour))
			assert.NoError(t, err)

			// Execute
			err = repo.DeleteByFeed(ctx, mockData.ID)
			assert.NoError(t, err)

			// Verify
			exists, err := repo.Exists(mockData.ID, "test_key")
			assert.NoError(t, err)
			assert.False(t, exists)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
		})

		t.Run(fmt.Sprintf("DeleteByFeed_Fails_NoRecords [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.DeleteByFeed(ctx, -1)

			// Verify
			assert.NoError(t, err)
		})
	}
}

func TestFeedCacheRepo_DeleteStale(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db

		log := setupLoggerForTest()
		repo := NewFeedCacheRepo(log, db)
		feedRepo := NewFeedRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFeed()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("DeleteStale_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			mockData.IndexerID = int(indexer.ID)

			err = feedRepo.Store(ctx, mockData)
			assert.NoError(t, err)

			// Adding a stale record (older than 30 days)
			err = repo.Put(mockData.ID, "test_stale_key", []byte("test_stale_value"), time.Now().AddDate(0, 0, -31))
			assert.NoError(t, err)

			// Execute
			err = repo.DeleteStale(ctx)
			assert.NoError(t, err)

			// Verify
			exists, err := repo.Exists(mockData.ID, "test_stale_key")
			assert.NoError(t, err)
			assert.False(t, exists)

			// Cleanup
			_ = feedRepo.Delete(ctx, mockData.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
		})

		t.Run(fmt.Sprintf("DeleteStale_Fails_NoRecords [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.DeleteStale(ctx)

			// Verify
			assert.NoError(t, err)
		})
	}
}
