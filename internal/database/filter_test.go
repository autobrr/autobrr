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

func getMockFilter() *domain.Filter {
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
		MaxDownloadsPeriod:   1,
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

func getMockFilterExternal() domain.FilterExternal {
	return domain.FilterExternal{
		Name:                     "ExternalFilter",
		Index:                    1,
		Type:                     domain.ExternalFilterTypeExec,
		Enabled:                  true,
		ExecCmd:                  "",
		ExecArgs:                 "",
		ExecExpectStatus:         0,
		WebhookHost:              "",
		WebhookMethod:            "",
		WebhookData:              "",
		WebhookHeaders:           "",
		WebhookExpectStatus:      0,
		WebhookRetryStatus:       "",
		WebhookRetryAttempts:     0,
		WebhookRetryDelaySeconds: 0,
		OnError:                  domain.FilterExternalOnErrorReject,
	}
}

func TestFilterRepo_Store(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)
			assert.Equal(t, mockData.Name, createdFilters[0].Name)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Missing_or_empty_fields [%s]", dbType), func(t *testing.T) {
			mockData := domain.Filter{}
			err := repo.Store(ctx, &mockData)
			assert.Error(t, err)

			createdFilters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			// Cleanup
			// No cleanup needed
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
			defer cancel()

			err := repo.Store(timeoutCtx, mockData)
			assert.Error(t, err)
		})
	}
}

func TestFilterRepo_Update(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			// Update mockData
			mockData.Name = "Updated Filter"
			mockData.Enabled = false
			mockData.CreatedAt = time.Now()

			// Execute
			err = repo.Update(ctx, mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)
			assert.Equal(t, "Updated Filter", createdFilters[0].Name)
			assert.Equal(t, false, createdFilters[0].Enabled)

			// Cleanup
			_ = repo.Delete(ctx, createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			mockData.ID = -1
			err := repo.Update(ctx, mockData)
			assert.Error(t, err)
		})
	}
}

func TestFilterRepo_Delete(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)
			assert.Equal(t, mockData.Name, createdFilters[0].Name)

			// Execute
			err = repo.Delete(ctx, createdFilters[0].ID)
			assert.NoError(t, err)

			// Verify that the filter is deleted and return error ErrRecordNotFound
			filter, err := repo.FindByID(ctx, createdFilters[0].ID)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
			assert.Nil(t, filter)
		})

		t.Run(fmt.Sprintf("Delete_Fails_No_Record [%s]", dbType), func(t *testing.T) {
			err := repo.Delete(ctx, 9999)
			assert.Error(t, err)
		})

	}
}

func TestFilterRepo_UpdatePartial(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("UpdatePartial_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)
			updatedName := "Updated Name"

			createdFilters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			// Execute
			updateData := domain.FilterUpdate{ID: createdFilters[0].ID, Name: &updatedName}
			err = repo.UpdatePartial(ctx, updateData)
			assert.NoError(t, err)

			// Verify that the filter is updated
			filter, err := repo.FindByID(ctx, createdFilters[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, filter)
			assert.Equal(t, updatedName, filter.Name)

			// Cleanup
			_ = repo.Delete(ctx, createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("UpdatePartial_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			// Setup
			updatedName := "Should Fail"
			updateData := domain.FilterUpdate{ID: -1, Name: &updatedName}
			err := repo.UpdatePartial(ctx, updateData)
			assert.Error(t, err)
		})
	}
}

func TestFilterRepo_ToggleEnabled(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)
			assert.Equal(t, true, createdFilters[0].Enabled)

			// Execute
			err = repo.ToggleEnabled(ctx, mockData.ID, false)
			assert.NoError(t, err)

			// Verify that the filter is updated
			filter, err := repo.FindByID(ctx, createdFilters[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, filter)
			assert.Equal(t, false, filter.Enabled)

			// Cleanup
			_ = repo.Delete(ctx, createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.ToggleEnabled(ctx, -1, false)
			assert.Error(t, err)
		})

	}
}

func TestFilterRepo_ListFilters(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("ListFilters_ReturnsFilters [%s]", dbType), func(t *testing.T) {
			// Setup
			for i := 0; i < 10; i++ {
				err := repo.Store(ctx, mockData)
				assert.NoError(t, err)
			}

			// Execute
			filters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)

			// Cleanup
			for _, filter := range filters {
				_ = repo.Delete(ctx, filter.ID)
			}
		})

		t.Run(fmt.Sprintf("ListFilters_ReturnsEmptyList [%s]", dbType), func(t *testing.T) {
			filters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.Empty(t, filters)
		})

	}
}

func TestFilterRepo_Find(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFilter()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Find_Basic [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData.Name = "Test Filter"
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			params := domain.FilterQueryParams{
				Search: "Test",
			}

			// Execute
			filters, err := repo.Find(ctx, params)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)

			// Cleanup
			_ = repo.Delete(ctx, filters[0].ID)
		})

		t.Run(fmt.Sprintf("Find_Sort [%s]", dbType), func(t *testing.T) {
			// Setup
			for i := 0; i < 10; i++ {
				mockData.Name = fmt.Sprintf("Test Filter %d", i)
				err := repo.Store(ctx, mockData)
				assert.NoError(t, err)
			}

			params := domain.FilterQueryParams{
				Sort: map[string]string{
					"name": "desc",
				},
			}

			// Execute
			filters, err := repo.Find(ctx, params)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)
			assert.Equal(t, "Test Filter 9", filters[0].Name)
			assert.Equal(t, 10, len(filters))

			// Cleanup
			for _, filter := range filters {
				_ = repo.Delete(ctx, filter.ID)
			}
		})

		t.Run(fmt.Sprintf("Find_Filters [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData.Name = "New Filter With Filters"
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			allFilter, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, allFilter)

			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			// Store indexer connection
			err = repo.StoreIndexerConnection(ctx, allFilter[0].ID, int(indexer.ID))

			params := domain.FilterQueryParams{
				Filters: struct{ Indexers []string }{Indexers: []string{"indexer1"}},
			}

			// Execute
			filters, err := repo.Find(ctx, params)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)
			assert.Equal(t, "New Filter With Filters", filters[0].Name)
			assert.Equal(t, 1, len(filters))

			// Cleanup
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, filters[0].ID)
		})

	}
}

func TestFilterRepo_FindByID(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			// Execute
			filter, err := repo.FindByID(ctx, createdFilters[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, filter)
			assert.Equal(t, createdFilters[0].ID, filter.ID)

			// Cleanup
			_ = repo.Delete(ctx, createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			// Test using an invalid ID
			filter, err := repo.FindByID(ctx, -1)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound) // should return an error
			assert.Nil(t, filter)                            // should be nil
		})

	}
}

func TestFilterRepo_FindByIndexerIdentifier(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		//mockData := getMockFilter()
		indexerMockData := getMockIndexer()

		filtersData := []*domain.Filter{
			{
				Enabled:     true,
				Name:        "filter 1",
				Priority:    20,
				Resolutions: []string{},
				Codecs:      []string{},
				Sources:     []string{},
				Containers:  []string{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				Enabled:     true,
				Name:        "filter 2",
				Priority:    30,
				Resolutions: []string{},
				Codecs:      []string{},
				Sources:     []string{},
				Containers:  []string{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
			{
				Enabled:     true,
				Name:        "filter 20",
				Priority:    100,
				Resolutions: []string{},
				Codecs:      []string{},
				Sources:     []string{},
				Containers:  []string{},
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			},
		}

		t.Run(fmt.Sprintf("FindByIndexerIdentifier_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			for _, filter := range filtersData {
				filter := filter
				err := repo.Store(ctx, filter)
				assert.NoError(t, err)

				err = repo.StoreIndexerConnection(ctx, filter.ID, int(indexer.ID))
				assert.NoError(t, err)
			}

			filtersList, err := repo.ListFilters(ctx)
			assert.NoError(t, err)
			assert.NotEmpty(t, filtersList)

			// Execute
			filters, err := repo.FindByIndexerIdentifier(ctx, indexerMockData.Identifier)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)

			assert.Equal(t, filters[0].Priority, int32(100))
			assert.Equal(t, filters[1].Priority, int32(30))
			assert.Equal(t, filters[2].Priority, int32(20))

			// Cleanup
			_ = indexerRepo.Delete(ctx, int(indexer.ID))

			for _, filter := range filtersData {
				filter := filter

				_ = repo.Delete(ctx, filter.ID)
			}
		})

		t.Run(fmt.Sprintf("FindByIndexerIdentifier_Fails_Invalid_Identifier [%s]", dbType), func(t *testing.T) {
			filters, err := repo.FindByIndexerIdentifier(ctx, "invalid-identifier")
			assert.NoError(t, err) // should return an error??
			assert.Nil(t, filters)
		})

	}
}

func TestFilterRepo_FindExternalFiltersByID(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)

		t.Run(fmt.Sprintf("FindExternalFiltersByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			mockDataExternal := getMockFilterExternal()
			err = repo.StoreFilterExternal(ctx, mockData.ID, []domain.FilterExternal{mockDataExternal})
			assert.NoError(t, err)

			// Execute
			filters, err := repo.FindExternalFiltersByID(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("FindExternalFiltersByID_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			filters, err := repo.FindExternalFiltersByID(ctx, -1)
			assert.NoError(t, err) // should return an error??
			assert.Nil(t, filters)
		})

	}
}

func TestFilterRepo_FindExternalFiltersByFilterIDs(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)

		t.Run(fmt.Sprintf("FindExternalFiltersByFilterIDs_Groups_By_Filter [%s]", dbType), func(t *testing.T) {
			// Setup
			firstFilter := getMockFilter()
			err := repo.Store(ctx, firstFilter)
			assert.NoError(t, err)

			secondFilter := getMockFilter()
			err = repo.Store(ctx, secondFilter)
			assert.NoError(t, err)

			first := getMockFilterExternal()
			first.Index = 1
			second := getMockFilterExternal()
			second.Name = "ExternalFilterTwo"
			second.Index = 2

			err = repo.StoreFilterExternal(ctx, firstFilter.ID, []domain.FilterExternal{first, second})
			assert.NoError(t, err)

			err = repo.StoreFilterExternal(ctx, secondFilter.ID, []domain.FilterExternal{first})
			assert.NoError(t, err)

			// Execute
			externalFilters, err := repo.FindExternalFiltersByFilterIDs(ctx, []int{firstFilter.ID, secondFilter.ID})

			// Verify
			assert.NoError(t, err)
			assert.Len(t, externalFilters, 2)
			assert.Len(t, externalFilters[firstFilter.ID], 2)
			assert.Len(t, externalFilters[secondFilter.ID], 1)

			// idx DESC is preserved within each filter
			assert.Equal(t, 2, externalFilters[firstFilter.ID][0].Index)
			assert.Equal(t, 1, externalFilters[firstFilter.ID][1].Index)

			// matches what the per-filter lookup returns
			single, err := repo.FindExternalFiltersByID(ctx, firstFilter.ID)
			assert.NoError(t, err)
			assert.Equal(t, single, externalFilters[firstFilter.ID])

			// Cleanup
			_ = repo.Delete(ctx, firstFilter.ID)
			_ = repo.Delete(ctx, secondFilter.ID)
		})

		t.Run(fmt.Sprintf("FindExternalFiltersByFilterIDs_Empty_Input [%s]", dbType), func(t *testing.T) {
			externalFilters, err := repo.FindExternalFiltersByFilterIDs(ctx, []int{})
			assert.NoError(t, err)
			assert.Empty(t, externalFilters)
		})

		t.Run(fmt.Sprintf("FindExternalFiltersByFilterIDs_Unknown_ID [%s]", dbType), func(t *testing.T) {
			externalFilters, err := repo.FindExternalFiltersByFilterIDs(ctx, []int{-1})
			assert.NoError(t, err)
			assert.Empty(t, externalFilters)
			assert.Nil(t, externalFilters[-1])
		})
	}
}

func TestFilterRepo_StoreIndexerConnection(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		t.Run(fmt.Sprintf("StoreIndexerConnection_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			indexerMockData := getMockIndexer()
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			// Execute
			err = repo.StoreIndexerConnection(ctx, mockData.ID, int(indexer.ID))
			assert.NoError(t, err)

			// Cleanup
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("StoreIndexerConnection_Fails_Invalid_IDs [%s]", dbType), func(t *testing.T) {
			// Execute with invalid IDs
			err := repo.StoreIndexerConnection(ctx, -1, -1)
			assert.Error(t, err)
		})

	}
}

func TestFilterRepo_StoreIndexerConnections(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		t.Run(fmt.Sprintf("StoreIndexerConnections_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			indexerMockData := getMockIndexer()
			var indexers []domain.Indexer
			for i := 0; i < 2; i++ {
				// identifier must be unique
				indexerMockData.Identifier = fmt.Sprintf("indexer%d", i)
				indexer, err := indexerRepo.Store(ctx, indexerMockData)
				assert.NoError(t, err)
				indexers = append(indexers, *indexer)
			}

			// Execute
			err = repo.StoreIndexerConnections(ctx, mockData.ID, indexers)
			assert.NoError(t, err)

			// Validate that the connections were successfully stored in the database
			connections, err := repo.FindByIndexerIdentifier(ctx, indexerMockData.Identifier)
			assert.NoError(t, err)
			assert.NotNil(t, connections)
			assert.NotEmpty(t, connections)

			// Cleanup
			for _, indexer := range indexers {
				_ = indexerRepo.Delete(ctx, int(indexer.ID))
			}
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("StoreIndexerConnections_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.StoreIndexerConnections(ctx, -1, []domain.Indexer{})
			assert.NoError(t, err) //TODO: // this should return an error.
		})
	}
}

func TestFilterRepo_StoreFilterExternal(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)

		t.Run(fmt.Sprintf("StoreFilterExternal_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			// Execute
			mockDataExternal := getMockFilterExternal()
			err = repo.StoreFilterExternal(ctx, mockData.ID, []domain.FilterExternal{mockDataExternal})
			assert.NoError(t, err)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("StoreFilterExternal_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.StoreFilterExternal(ctx, -1, []domain.FilterExternal{})
			assert.NoError(t, err) // TODO: this should return an error
		})
	}
}

func TestFilterRepo_DeleteIndexerConnections(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		t.Run(fmt.Sprintf("DeleteIndexerConnections_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			indexerMockData := getMockIndexer()
			indexer, err := indexerRepo.Store(ctx, indexerMockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			err = repo.StoreIndexerConnection(ctx, mockData.ID, int(indexer.ID))
			assert.NoError(t, err)

			// Execute
			err = repo.DeleteIndexerConnections(ctx, mockData.ID)
			assert.NoError(t, err)

			// Validate that the connections were successfully deleted from the database
			connections, err := repo.FindByIndexerIdentifier(ctx, indexerMockData.Identifier)
			assert.NoError(t, err)
			assert.Nil(t, connections)

			// Cleanup
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("DeleteIndexerConnections_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.DeleteIndexerConnections(ctx, -1)
			assert.NoError(t, err) // TODO: this should return an error
		})

	}
}

func TestFilterRepo_DeleteFilterExternal(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)

		t.Run(fmt.Sprintf("DeleteFilterExternal_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			mockDataExternal := getMockFilterExternal()
			err = repo.StoreFilterExternal(ctx, mockData.ID, []domain.FilterExternal{mockDataExternal})
			assert.NoError(t, err)

			// Execute
			err = repo.DeleteFilterExternal(ctx, mockData.ID)
			assert.NoError(t, err)

			// Validate that the connections were successfully deleted from the database
			connections, err := repo.FindExternalFiltersByID(ctx, mockData.ID)
			assert.NoError(t, err)
			assert.Nil(t, connections)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("DeleteFilterExternal_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.DeleteFilterExternal(ctx, -1)
			assert.NoError(t, err) // TODO: this should return an error
		})

	}
}

func TestFilterRepo_GetFilterDownloads(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		releaseRepo := NewReleaseRepo(log, db)
		downloadClientRepo := NewDownloadClientRepo(log, db)
		actionRepo := NewActionRepo(log, db)

		t.Run(fmt.Sprintf("GetFilterDownloads_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()

			err = downloadClientRepo.Store(ctx, &mockClient)
			assert.NoError(t, err)
			assert.NotNil(t, mockClient)

			mockAction := getMockAction()
			mockAction.FilterID = mockData.ID
			mockAction.ClientID = mockClient.ID

			err = actionRepo.Store(ctx, mockAction)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.FilterID = int64(mockData.ID)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockData.ID

			err = releaseRepo.Store(ctx, mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID

			err = releaseRepo.StoreReleaseActionStatus(ctx, mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute
			err = repo.GetFilterDownloadCount(ctx, mockData)
			assert.NoError(t, err)
			assert.NotNil(t, mockData.Downloads)
			assert.Equal(t, mockData.Downloads, &domain.FilterDownloads{
				PeriodCount: 1,
				TotalCount:  1,
			})

			// Cleanup
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(ctx, mockData.ID)
			_ = downloadClientRepo.Delete(ctx, mockClient.ID)
			_ = releaseRepo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Returns_Zero_For_Unknown_ID [%s]", dbType), func(t *testing.T) {
			mockFilter := &domain.Filter{ID: -1, MaxDownloadsUnit: domain.FilterMaxDownloadsHour}
			err := repo.GetFilterDownloadCount(ctx, mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				PeriodCount: 0,
				TotalCount:  0,
			})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Multiple_Actions [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			err := repo.Store(ctx, mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()

			err = downloadClientRepo.Store(ctx, &mockClient)
			assert.NoError(t, err)
			assert.NotNil(t, mockClient)

			mockAction1 := getMockAction()
			mockAction1.FilterID = mockFilter.ID
			mockAction1.ClientID = mockClient.ID

			actionErr := actionRepo.Store(ctx, mockAction1)
			assert.NoError(t, actionErr)

			mockAction2 := getMockAction()
			mockAction2.FilterID = mockFilter.ID
			mockAction2.ClientID = mockClient.ID

			action2Err := actionRepo.Store(ctx, mockAction2)
			assert.NoError(t, action2Err)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID

			err = releaseRepo.Store(ctx, mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus1 := getMockReleaseActionStatus()
			mockReleaseActionStatus1.ActionID = int64(mockAction1.ID)
			mockReleaseActionStatus1.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus1.ReleaseID = mockRelease.ID

			err = releaseRepo.StoreReleaseActionStatus(ctx, mockReleaseActionStatus1)
			assert.NoError(t, err)

			mockReleaseActionStatus2 := getMockReleaseActionStatus()
			mockReleaseActionStatus2.ActionID = int64(mockAction2.ID)
			mockReleaseActionStatus2.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus2.ReleaseID = mockRelease.ID

			err = releaseRepo.StoreReleaseActionStatus(ctx, mockReleaseActionStatus2)
			assert.NoError(t, err)

			// Execute
			err = repo.GetFilterDownloadCount(ctx, mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				PeriodCount: 1,
				TotalCount:  1,
			})

			// Cleanup
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: mockAction1.ID})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: mockAction2.ID})
			_ = repo.Delete(ctx, mockFilter.ID)
			_ = downloadClientRepo.Delete(ctx, mockClient.ID)
			_ = releaseRepo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Old_Release [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			mockFilter.Shows = "nope"
			err := repo.Store(ctx, mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()

			err = downloadClientRepo.Store(ctx, &mockClient)
			assert.NoError(t, err)
			assert.NotNil(t, mockClient)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID

			err = actionRepo.Store(ctx, mockAction)
			assert.NoError(t, err)

			mockAction2 := getMockAction()
			mockAction2.FilterID = mockFilter.ID
			mockAction2.ClientID = mockClient.ID

			err = actionRepo.Store(ctx, mockAction2)
			assert.NoError(t, err)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID

			err = releaseRepo.Store(ctx, mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID
			mockReleaseActionStatus.Timestamp = mockReleaseActionStatus.Timestamp.Add(-35 * 24 * time.Hour) // use Add instead of AddDate(0, -1, 0) to not cause issues where previous month is shorter than current month.

			err = releaseRepo.StoreReleaseActionStatus(ctx, mockReleaseActionStatus)
			assert.NoError(t, err)

			mockReleaseActionStatus2 := getMockReleaseActionStatus()
			mockReleaseActionStatus2.ActionID = int64(mockAction2.ID)
			mockReleaseActionStatus2.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus2.ReleaseID = mockRelease.ID
			mockReleaseActionStatus2.Timestamp = mockReleaseActionStatus2.Timestamp.Add(-35 * 24 * time.Hour) // use Add instead of AddDate(0, -1, 0) to not cause issues where previous month is shorter than current month.

			err = releaseRepo.StoreReleaseActionStatus(ctx, mockReleaseActionStatus2)
			assert.NoError(t, err)

			// Execute
			err = repo.GetFilterDownloadCount(ctx, mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				PeriodCount: 0,
				TotalCount:  1,
			})

			// Cleanup
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = actionRepo.Delete(ctx, &domain.DeleteActionRequest{ActionId: mockAction2.ID})
			_ = repo.Delete(ctx, mockFilter.ID)
			_ = downloadClientRepo.Delete(ctx, mockClient.ID)
			_ = releaseRepo.Delete(ctx, &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_No_Releases [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			err := repo.Store(ctx, mockFilter)
			assert.NoError(t, err)

			err = repo.GetFilterDownloadCount(ctx, mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				PeriodCount: 0,
				TotalCount:  0,
			})

			// Cleanup
			_ = repo.Delete(ctx, mockFilter.ID)
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Rolling_Window_Minute [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			mockFilter.MaxDownloadsWindowType = domain.FilterMaxDownloadsWindowRolling
			mockFilter.MaxDownloadsUnit = domain.FilterMaxDownloadsMinute
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()
			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID
			err = actionRepo.Store(t.Context(), mockAction)
			assert.NoError(t, err)

			// Create release with status within the last minute (should be counted)
			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID
			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID
			mockReleaseActionStatus.Timestamp = time.Now().Add(-30 * time.Second)

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute - should count the release from 30 seconds ago
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, 1, mockFilter.Downloads.PeriodCount, "should count release within last minute")

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Rolling_Window_Hour [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			mockFilter.MaxDownloadsWindowType = domain.FilterMaxDownloadsWindowRolling
			mockFilter.MaxDownloadsUnit = domain.FilterMaxDownloadsHour
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()
			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID
			err = actionRepo.Store(t.Context(), mockAction)
			assert.NoError(t, err)

			// Create release within the last hour
			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID
			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID
			mockReleaseActionStatus.Timestamp = time.Now().Add(-30 * time.Minute)

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute - should count the release from 30 minutes ago
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, 1, mockFilter.Downloads.PeriodCount, "should count release within last hour")

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Rolling_Window_Excludes_Old [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			mockFilter.MaxDownloadsWindowType = domain.FilterMaxDownloadsWindowRolling
			mockFilter.MaxDownloadsUnit = domain.FilterMaxDownloadsMinute
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()
			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID
			err = actionRepo.Store(t.Context(), mockAction)
			assert.NoError(t, err)

			// Create release outside the rolling window (should NOT be counted)
			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID
			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID
			mockReleaseActionStatus.Timestamp = time.Now().Add(-2 * time.Minute)

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute - should NOT count the release from 2 minutes ago for a 1-minute window
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, 0, mockFilter.Downloads.PeriodCount, "should not count release outside rolling window")
			assert.Equal(t, 1, mockFilter.Downloads.TotalCount, "total count should include all releases")

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Default_To_Fixed [%s]", dbType), func(t *testing.T) {
			// Setup - filter without MaxDownloadsWindowType should default to FIXED
			mockFilter := getMockFilter()
			mockFilter.MaxDownloadsWindowType = "" // Explicitly unset
			mockFilter.MaxDownloadsUnit = domain.FilterMaxDownloadsHour
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()
			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID
			err = actionRepo.Store(t.Context(), mockAction)
			assert.NoError(t, err)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID
			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute - should use FIXED window (calendar boundaries) by default
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			// Since we created a release just now, it should be in the current hour for FIXED window
			assert.Equal(t, 1, mockFilter.Downloads.PeriodCount, "should count using FIXED window by default")

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Rolling_Window_Day [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			mockFilter.MaxDownloadsWindowType = domain.FilterMaxDownloadsWindowRolling
			mockFilter.MaxDownloadsUnit = domain.FilterMaxDownloadsDay
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()
			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID
			err = actionRepo.Store(t.Context(), mockAction)
			assert.NoError(t, err)

			// Create release within the last 24 hours
			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID
			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID
			mockReleaseActionStatus.Timestamp = time.Now().Add(-12 * time.Hour)

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute - should count the release from 12 hours ago
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, 1, mockFilter.Downloads.PeriodCount, "should count release within last 24 hours")

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Rolling_Window_Period [%s]", dbType), func(t *testing.T) {
			// Setup - a 2 days old release must be inside a rolling 3 day window
			mockFilter := getMockFilter()
			mockFilter.MaxDownloadsWindowType = domain.FilterMaxDownloadsWindowRolling
			mockFilter.MaxDownloadsUnit = domain.FilterMaxDownloadsDay
			mockFilter.MaxDownloadsPeriod = 3
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()
			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID
			err = actionRepo.Store(t.Context(), mockAction)
			assert.NoError(t, err)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID
			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID
			mockReleaseActionStatus.Timestamp = time.Now().Add(-2 * 24 * time.Hour)

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, 1, mockFilter.Downloads.PeriodCount, "should count release within rolling 3 day window")

			// Execute - the same release must be outside a rolling 1 day window
			mockFilter.MaxDownloadsPeriod = 1
			err = repo.Update(t.Context(), mockFilter)
			assert.NoError(t, err)

			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, 0, mockFilter.Downloads.PeriodCount, "should not count release outside rolling 1 day window")
			assert.Equal(t, 1, mockFilter.Downloads.TotalCount)

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Ever [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			mockFilter.MaxDownloadsUnit = domain.FilterMaxDownloadsEver
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()
			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID
			err = actionRepo.Store(t.Context(), mockAction)
			assert.NoError(t, err)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID
			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID
			mockReleaseActionStatus.Timestamp = time.Now().Add(-365 * 24 * time.Hour)

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				PeriodCount: 1,
				TotalCount:  1,
			})

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})
	}
}
