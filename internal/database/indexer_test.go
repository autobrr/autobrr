// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"context"
	"fmt"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
)

func getMockIndexer() domain.Indexer {
	return domain.Indexer{
		ID:             0,
		Name:           "indexer1",
		Identifier:     "indexer1",
		Enabled:        true,
		Implementation: "meh",
		BaseURL:        "ok",
		Settings:       nil,
	}
}

func TestIndexerRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewIndexerRepo(log, db)
		mockData := getMockIndexer()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdIndexer, err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Verify
			indexer, err := repo.FindByID(context.Background(), int(createdIndexer.ID))
			assert.NoError(t, err)
			assert.Equal(t, mockData.Name, createdIndexer.Name)
			assert.Equal(t, mockData.Identifier, createdIndexer.Identifier)
			assert.Equal(t, mockData.Enabled, indexer.Enabled)

			// Cleanup
			_ = repo.Delete(context.Background(), int(createdIndexer.ID))
		})

	}
}

func TestIndexerRepo_Update(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewIndexerRepo(log, db)

		initialData := getMockIndexer()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdIndexer, err := repo.Store(context.Background(), initialData)
			assert.NoError(t, err)

			createdIndexer.Name = "UpdatedName"
			createdIndexer.Enabled = false

			// Execute
			updatedIndexer, err := repo.Update(context.Background(), *createdIndexer)
			assert.NoError(t, err)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, "UpdatedName", updatedIndexer.Name)
			assert.Equal(t, createdIndexer.Enabled, updatedIndexer.Enabled)

			// Cleanup
			_ = repo.Delete(context.Background(), int(updatedIndexer.ID))
		})
	}
}

func TestIndexerRepo_List(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewIndexerRepo(log, db)

		t.Run(fmt.Sprintf("List_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData1 := getMockIndexer()
			mockData1.Name = "Indexer1"
			mockData1.Identifier = "Identifier1"

			mockData2 := getMockIndexer()
			mockData2.Name = "Indexer2"
			mockData2.Identifier = "Identifier2"

			createdIndexer1, err := repo.Store(context.Background(), mockData1)
			assert.NoError(t, err)
			createdIndexer2, err := repo.Store(context.Background(), mockData2)
			assert.NoError(t, err)

			// Execute
			indexers, err := repo.List(context.Background())
			assert.NoError(t, err)

			// Verify
			assert.Contains(t, indexers, *createdIndexer1)
			assert.Contains(t, indexers, *createdIndexer2)

			assert.Equal(t, 2, len(indexers))

			// Cleanup
			_ = repo.Delete(context.Background(), int(createdIndexer1.ID))
			_ = repo.Delete(context.Background(), int(createdIndexer2.ID))
		})
	}
}

func TestIndexerRepo_FindByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewIndexerRepo(log, db)
		mockData := getMockIndexer()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData.Name = "TestIndexer"
			mockData.Identifier = "TestIdentifier"

			createdIndexer, err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Execute
			foundIndexer, err := repo.FindByID(context.Background(), int(createdIndexer.ID))
			assert.NoError(t, err)

			// Verify
			assert.Equal(t, createdIndexer.ID, foundIndexer.ID)
			assert.Equal(t, createdIndexer.Name, foundIndexer.Name)
			assert.Equal(t, createdIndexer.Identifier, foundIndexer.Identifier)
			assert.Equal(t, createdIndexer.Enabled, foundIndexer.Enabled)

			// Cleanup
			_ = repo.Delete(context.Background(), int(createdIndexer.ID))
		})
	}
}

func TestIndexerRepo_FindByFilterID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIndexerRepo(log, db)
		filterRepo := NewFilterRepo(log, db)

		filterMockData := getMockFilter()
		mockData := getMockIndexer()

		t.Run(fmt.Sprintf("FindByFilterID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := filterRepo.Store(context.Background(), filterMockData)
			assert.NoError(t, err)

			indexer, err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			err = filterRepo.StoreIndexerConnection(context.Background(), filterMockData.ID, int(indexer.ID))
			assert.NoError(t, err)

			// Execute
			foundIndexers, err := repo.FindByFilterID(context.Background(), filterMockData.ID)
			assert.NoError(t, err)

			// Verify
			assert.Len(t, foundIndexers, 1)
			assert.Equal(t, indexer.Name, foundIndexers[0].Name)
			assert.Equal(t, indexer.Identifier, foundIndexers[0].Identifier)

			// Cleanup
			_ = repo.Delete(context.Background(), int(indexer.ID))
			_ = filterRepo.Delete(context.Background(), filterMockData.ID)
		})
	}
}

func TestIndexerRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewIndexerRepo(log, db)
		mockData := getMockIndexer()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			createdIndexer, err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			assert.NotNil(t, createdIndexer)

			// Execute
			err = repo.Delete(context.Background(), int(createdIndexer.ID))
			assert.NoError(t, err)

			// Verify
			_, err = repo.FindByID(context.Background(), int(createdIndexer.ID))
			assert.Error(t, err)
		})
	}
}
