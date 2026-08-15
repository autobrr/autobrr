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
	"github.com/stretchr/testify/require"
)

func getMockIndexer() domain.Indexer {
	return domain.Indexer{
		ID:             0,
		Name:           "indexer1",
		Identifier:     "indexer1",
		Enabled:        true,
		Implementation: domain.IndexerImplementationIRC,
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
			err = repo.Update(context.Background(), createdIndexer)
			assert.NoError(t, err)

			// Verify
			assert.NoError(t, err)
			assert.Equal(t, "UpdatedName", createdIndexer.Name)
			assert.Equal(t, createdIndexer.Enabled, false)

			// Cleanup
			_ = repo.Delete(context.Background(), int(createdIndexer.ID))
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

// storeTestIndexer stores a minimal enabled IRC indexer keyed by identifier.
func storeTestIndexer(t *testing.T, repo *IndexerRepo, identifier string) *domain.Indexer {
	t.Helper()

	indexer, err := repo.Store(context.Background(), domain.Indexer{
		Enabled:        true,
		Name:           identifier,
		Identifier:     identifier,
		Implementation: domain.IndexerImplementationIRC,
		Settings:       map[string]string{},
	})
	require.NoError(t, err)
	require.NotNil(t, indexer)

	return indexer
}

func TestIndexerRepo_ArchiveAndDeprecation(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewIndexerRepo(log, db)
		ctx := context.Background()

		t.Run(fmt.Sprintf("ArchiveAndDeprecation_Succeeds [%s]", dbType), func(t *testing.T) {
			stored := storeTestIndexer(t, repo, "fnp")

			require.NoError(t, repo.UpsertDeprecation(ctx, domain.IndexerDeprecation{
				Identifier:   "fnp",
				Name:         "old name",
				Reason:       "old reason",
				DeprecatedAt: time.Date(2026, time.May, 11, 0, 0, 0, 0, time.UTC),
			}))
			require.NoError(t, repo.UpsertDeprecation(ctx, domain.IndexerDeprecation{
				Identifier:   "fnp",
				Name:         "FearNoPeer",
				Reason:       "Tracker shut down",
				IssueURL:     "https://github.com/autobrr/autobrr/issues/2481",
				DeprecatedAt: time.Date(2026, time.May, 11, 0, 0, 0, 0, time.UTC),
			}))

			require.NoError(t, repo.ArchiveByIdentifier(ctx, "fnp"))

			got, err := repo.FindByID(ctx, int(stored.ID))
			require.NoError(t, err)
			assert.True(t, got.Archived, "indexer should be archived")
			assert.NotNil(t, got.ArchivedAt, "archived_at should be stamped")
			assert.True(t, got.Enabled, "archiving must not clobber the enabled flag")

			deps, err := repo.ListDeprecations(ctx)
			require.NoError(t, err)
			require.Len(t, deps, 1)
			assert.Equal(t, "FearNoPeer", deps[0].Name)
			assert.Equal(t, "Tracker shut down", deps[0].Reason)
			assert.Equal(t, 0, deps[0].FilterCount, "no filters reference it yet")

			firstStamp := *got.ArchivedAt
			require.NoError(t, repo.ArchiveByIdentifier(ctx, "fnp"))
			got2, err := repo.FindByID(ctx, int(stored.ID))
			require.NoError(t, err)
			require.NotNil(t, got2.ArchivedAt)
			assert.Equal(t, firstStamp.UTC(), got2.ArchivedAt.UTC(), "archived_at must be stable across re-archive")

			require.NoError(t, repo.UnarchiveByIdentifier(ctx, "fnp"))
			got3, err := repo.FindByID(ctx, int(stored.ID))
			require.NoError(t, err)
			assert.False(t, got3.Archived)
			assert.Nil(t, got3.ArchivedAt)
			assert.True(t, got3.Enabled, "un-archive must leave enabled untouched")

			_ = repo.Delete(ctx, int(stored.ID))
			_, _ = db.Handler.ExecContext(ctx, "DELETE FROM indexer_deprecation WHERE identifier = 'fnp'")
		})
	}
}

// TestReleaseNameResolution_Coalesce validates the COALESCE(i.name, d.name, r.indexer) logic
// the release list relies on, including the hard-deleted-row case (no indexer row, name still
// resolved from the deprecation table).
func TestReleaseNameResolution_Coalesce(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewIndexerRepo(log, db)
		ctx := context.Background()

		t.Run(fmt.Sprintf("ReleaseNameResolution_Coalesce [%s]", dbType), func(t *testing.T) {
			storeTestIndexer(t, repo, "activetracker")
			require.NoError(t, repo.UpsertDeprecation(ctx, domain.IndexerDeprecation{Identifier: "fnp", Name: "FearNoPeer"}))

			// identifiers are test-controlled constants, so inlining them keeps the query
			// db-agnostic (no ?/$1 placeholder differences between sqlite and postgres)
			resolve := func(identifier string) string {
				var name string
				query := fmt.Sprintf(`
					SELECT COALESCE(i.name, d.name, r.ind)
					FROM (SELECT '%s' AS ind) r
					LEFT JOIN indexer i ON r.ind = i.identifier
					LEFT JOIN indexer_deprecation d ON r.ind = d.identifier`, identifier)
				require.NoError(t, db.Handler.QueryRowContext(ctx, query).Scan(&name))
				return name
			}

			assert.Equal(t, "activetracker", resolve("activetracker"), "live indexer resolves via indexer.name")
			assert.Equal(t, "FearNoPeer", resolve("fnp"), "hard-deleted indexer resolves via deprecation.name")
			assert.Equal(t, "ghosttracker", resolve("ghosttracker"), "unknown indexer falls back to the raw identifier")

			active, err := repo.GetBy(ctx, domain.GetIndexerRequest{Identifier: "activetracker"})
			require.NoError(t, err)
			_ = repo.Delete(ctx, int(active.ID))
			_, _ = db.Handler.ExecContext(ctx, "DELETE FROM indexer_deprecation WHERE identifier = 'fnp'")
		})
	}
}

func TestStoreIndexerConnections_RejectsArchived(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		indexerRepo := NewIndexerRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		ctx := context.Background()

		t.Run(fmt.Sprintf("StoreIndexerConnections_RejectsArchived [%s]", dbType), func(t *testing.T) {
			active := storeTestIndexer(t, indexerRepo, "activetracker")
			dead := storeTestIndexer(t, indexerRepo, "fnp")
			require.NoError(t, indexerRepo.UpsertDeprecation(ctx, domain.IndexerDeprecation{Identifier: "fnp", Name: "FearNoPeer"}))
			require.NoError(t, indexerRepo.ArchiveByIdentifier(ctx, "fnp"))

			filter := getMockFilter()
			require.NoError(t, filterRepo.Store(ctx, filter))
			t.Cleanup(func() {
				_ = indexerRepo.Delete(ctx, int(active.ID))
				_ = indexerRepo.Delete(ctx, int(dead.ID))
				_ = filterRepo.Delete(ctx, filter.ID)
				_, _ = db.Handler.ExecContext(ctx, "DELETE FROM indexer_deprecation WHERE identifier = 'fnp'")
			})

			require.NoError(t, filterRepo.StoreIndexerConnections(ctx, filter.ID, []domain.Indexer{{ID: active.ID}}))

			err := filterRepo.StoreIndexerConnections(ctx, filter.ID, []domain.Indexer{
				{ID: active.ID},
				{ID: dead.ID},
			})
			require.ErrorIs(t, err, domain.ErrIndexerArchived)

			connected, err := indexerRepo.FindByFilterID(ctx, filter.ID)
			require.NoError(t, err)
			require.Len(t, connected, 1, "a rejected update must preserve existing connections")
			assert.Equal(t, "activetracker", connected[0].Identifier)

			require.NoError(t, indexerRepo.UnarchiveByIdentifier(ctx, "fnp"))
			require.NoError(t, filterRepo.StoreIndexerConnection(ctx, filter.ID, int(dead.ID)))
			require.NoError(t, indexerRepo.ArchiveByIdentifier(ctx, "fnp"))

			matchingFilters, err := filterRepo.FindByIndexerIdentifier(ctx, "fnp")
			require.NoError(t, err)
			assert.Empty(t, matchingFilters, "archived indexers must not dispatch releases to filters")

			deps, err := indexerRepo.ListDeprecations(ctx)
			require.NoError(t, err)
			require.Len(t, deps, 1)
			assert.Equal(t, 1, deps[0].FilterCount, "filter_count should see the legacy connection")

			require.ErrorIs(t, indexerRepo.DeleteArchived(ctx, int(dead.ID)), domain.ErrIndexerInUse)

			removed, err := filterRepo.DeleteArchivedIndexerConnections(ctx, []string{"fnp"})
			require.NoError(t, err)
			assert.Equal(t, int64(1), removed)

			connected, err = indexerRepo.FindByFilterID(ctx, filter.ID)
			require.NoError(t, err)
			require.Len(t, connected, 1, "only the live indexer remains after prune")
			assert.Equal(t, "activetracker", connected[0].Identifier)

			require.NoError(t, indexerRepo.DeleteArchived(ctx, int(dead.ID)))
			_, err = indexerRepo.GetBy(ctx, domain.GetIndexerRequest{Identifier: "fnp"})
			require.ErrorIs(t, err, domain.ErrRecordNotFound)
			deprecations, err := indexerRepo.ListDeprecations(ctx)
			require.NoError(t, err)
			require.Len(t, deprecations, 1, "purging saved settings must preserve release metadata")
		})
	}
}
