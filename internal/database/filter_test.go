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
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)
			assert.Equal(t, mockData.Name, createdFilters[0].Name)

			// Cleanup
			_ = repo.Delete(t.Context(), mockData.ID)
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Missing_or_empty_fields [%s]", dbType), func(t *testing.T) {
			mockData := domain.Filter{}
			err := repo.Store(t.Context(), &mockData)
			assert.Error(t, err)

			createdFilters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			// Cleanup
			// No cleanup needed
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 1*time.Nanosecond)
			defer cancel()

			err := repo.Store(ctx, mockData)
			assert.Error(t, err)
		})
	}
}

func TestFilterRepo_Update(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			// Update mockData
			mockData.Name = "Updated Filter"
			mockData.Enabled = false
			mockData.CreatedAt = time.Now()

			// Execute
			err = repo.Update(t.Context(), mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)
			assert.Equal(t, "Updated Filter", createdFilters[0].Name)
			assert.Equal(t, false, createdFilters[0].Enabled)

			// Cleanup
			_ = repo.Delete(t.Context(), createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			mockData.ID = -1
			err := repo.Update(t.Context(), mockData)
			assert.Error(t, err)
		})
	}
}

func TestFilterRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)
			assert.Equal(t, mockData.Name, createdFilters[0].Name)

			// Execute
			err = repo.Delete(t.Context(), createdFilters[0].ID)
			assert.NoError(t, err)

			// Verify that the filter is deleted and return error ErrRecordNotFound
			filter, err := repo.FindByID(t.Context(), createdFilters[0].ID)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
			assert.Nil(t, filter)
		})

		t.Run(fmt.Sprintf("Delete_Fails_No_Record [%s]", dbType), func(t *testing.T) {
			err := repo.Delete(t.Context(), 9999)
			assert.Error(t, err)
		})

	}
}

func TestFilterRepo_UpdatePartial(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("UpdatePartial_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)
			updatedName := "Updated Name"

			createdFilters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			// Execute
			updateData := domain.FilterUpdate{ID: createdFilters[0].ID, Name: &updatedName}
			err = repo.UpdatePartial(t.Context(), updateData)
			assert.NoError(t, err)

			// Verify that the filter is updated
			filter, err := repo.FindByID(t.Context(), createdFilters[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, filter)
			assert.Equal(t, updatedName, filter.Name)

			// Cleanup
			_ = repo.Delete(t.Context(), createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("UpdatePartial_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			// Setup
			updatedName := "Should Fail"
			updateData := domain.FilterUpdate{ID: -1, Name: &updatedName}
			err := repo.UpdatePartial(t.Context(), updateData)
			assert.Error(t, err)
		})
	}
}

func TestFilterRepo_ToggleEnabled(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)
			assert.Equal(t, true, createdFilters[0].Enabled)

			// Execute
			err = repo.ToggleEnabled(t.Context(), mockData.ID, false)
			assert.NoError(t, err)

			// Verify that the filter is updated
			filter, err := repo.FindByID(t.Context(), createdFilters[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, filter)
			assert.Equal(t, false, filter.Enabled)

			// Cleanup
			_ = repo.Delete(t.Context(), createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.ToggleEnabled(t.Context(), -1, false)
			assert.Error(t, err)
		})

	}
}

func TestFilterRepo_ListFilters(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("ListFilters_ReturnsFilters [%s]", dbType), func(t *testing.T) {
			// Setup
			for i := 0; i < 10; i++ {
				err := repo.Store(t.Context(), mockData)
				assert.NoError(t, err)
			}

			// Execute
			filters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)

			// Cleanup
			for _, filter := range filters {
				_ = repo.Delete(t.Context(), filter.ID)
			}
		})

		t.Run(fmt.Sprintf("ListFilters_ReturnsEmptyList [%s]", dbType), func(t *testing.T) {
			filters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.Empty(t, filters)
		})

	}
}

func TestFilterRepo_Find(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		mockData := getMockFilter()
		indexerMockData := getMockIndexer()

		t.Run(fmt.Sprintf("Find_Basic [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData.Name = "Test Filter"
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			params := domain.FilterQueryParams{
				Search: "Test",
			}

			// Execute
			filters, err := repo.Find(t.Context(), params)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)

			// Cleanup
			_ = repo.Delete(t.Context(), filters[0].ID)
		})

		t.Run(fmt.Sprintf("Find_Sort [%s]", dbType), func(t *testing.T) {
			// Setup
			for i := 0; i < 10; i++ {
				mockData.Name = fmt.Sprintf("Test Filter %d", i)
				err := repo.Store(t.Context(), mockData)
				assert.NoError(t, err)
			}

			params := domain.FilterQueryParams{
				Sort: map[string]string{
					"name": "desc",
				},
			}

			// Execute
			filters, err := repo.Find(t.Context(), params)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)
			assert.Equal(t, "Test Filter 9", filters[0].Name)
			assert.Equal(t, 10, len(filters))

			// Cleanup
			for _, filter := range filters {
				_ = repo.Delete(t.Context(), filter.ID)
			}
		})

		t.Run(fmt.Sprintf("Find_Filters [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData.Name = "New Filter With Filters"
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			allFilter, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, allFilter)

			indexer, err := indexerRepo.Store(t.Context(), indexerMockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			// Store indexer connection
			err = repo.StoreIndexerConnection(t.Context(), allFilter[0].ID, int(indexer.ID))

			params := domain.FilterQueryParams{
				Filters: struct{ Indexers []string }{Indexers: []string{"indexer1"}},
			}

			// Execute
			filters, err := repo.Find(t.Context(), params)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)
			assert.Equal(t, "New Filter With Filters", filters[0].Name)
			assert.Equal(t, 1, len(filters))

			// Cleanup
			_ = indexerRepo.Delete(t.Context(), int(indexer.ID))
			_ = repo.Delete(t.Context(), filters[0].ID)
		})

	}
}

func TestFilterRepo_FindByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		mockData := getMockFilter()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			createdFilters, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			// Execute
			filter, err := repo.FindByID(t.Context(), createdFilters[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, filter)
			assert.Equal(t, createdFilters[0].ID, filter.ID)

			// Cleanup
			_ = repo.Delete(t.Context(), createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			// Test using an invalid ID
			filter, err := repo.FindByID(t.Context(), -1)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound) // should return an error
			assert.Nil(t, filter)                            // should be nil
		})

	}
}

func TestFilterRepo_FindByIndexerIdentifier(t *testing.T) {
	for dbType, db := range testDBs {
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
			indexer, err := indexerRepo.Store(t.Context(), indexerMockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			for _, filter := range filtersData {
				filter := filter
				err := repo.Store(t.Context(), filter)
				assert.NoError(t, err)

				err = repo.StoreIndexerConnection(t.Context(), filter.ID, int(indexer.ID))
				assert.NoError(t, err)
			}

			filtersList, err := repo.ListFilters(t.Context())
			assert.NoError(t, err)
			assert.NotEmpty(t, filtersList)

			// Execute
			filters, err := repo.FindByIndexerIdentifier(t.Context(), indexerMockData.Identifier)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)

			assert.Equal(t, filters[0].Priority, int32(100))
			assert.Equal(t, filters[1].Priority, int32(30))
			assert.Equal(t, filters[2].Priority, int32(20))

			// Cleanup
			_ = indexerRepo.Delete(t.Context(), int(indexer.ID))

			for _, filter := range filtersData {
				filter := filter

				_ = repo.Delete(t.Context(), filter.ID)
			}
		})

		t.Run(fmt.Sprintf("FindByIndexerIdentifier_Fails_Invalid_Identifier [%s]", dbType), func(t *testing.T) {
			filters, err := repo.FindByIndexerIdentifier(t.Context(), "invalid-identifier")
			assert.NoError(t, err) // should return an error??
			assert.Nil(t, filters)
		})

	}
}

func TestFilterRepo_FindExternalFiltersByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)

		t.Run(fmt.Sprintf("FindExternalFiltersByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			mockDataExternal := getMockFilterExternal()
			err = repo.StoreFilterExternal(t.Context(), mockData.ID, []domain.FilterExternal{mockDataExternal})
			assert.NoError(t, err)

			// Execute
			filters, err := repo.FindExternalFiltersByID(t.Context(), mockData.ID)
			assert.NoError(t, err)
			assert.NotNil(t, filters)
			assert.NotEmpty(t, filters)

			// Cleanup
			_ = repo.Delete(t.Context(), mockData.ID)
		})

		t.Run(fmt.Sprintf("FindExternalFiltersByID_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			filters, err := repo.FindExternalFiltersByID(t.Context(), -1)
			assert.NoError(t, err) // should return an error??
			assert.Nil(t, filters)
		})

	}
}

func TestFilterRepo_StoreIndexerConnection(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		t.Run(fmt.Sprintf("StoreIndexerConnection_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			indexerMockData := getMockIndexer()
			indexer, err := indexerRepo.Store(t.Context(), indexerMockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			// Execute
			err = repo.StoreIndexerConnection(t.Context(), mockData.ID, int(indexer.ID))
			assert.NoError(t, err)

			// Cleanup
			_ = indexerRepo.Delete(t.Context(), int(indexer.ID))
			_ = repo.Delete(t.Context(), mockData.ID)
		})

		t.Run(fmt.Sprintf("StoreIndexerConnection_Fails_Invalid_IDs [%s]", dbType), func(t *testing.T) {
			// Execute with invalid IDs
			err := repo.StoreIndexerConnection(t.Context(), -1, -1)
			assert.Error(t, err)
		})

	}
}

func TestFilterRepo_StoreIndexerConnections(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		t.Run(fmt.Sprintf("StoreIndexerConnections_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			indexerMockData := getMockIndexer()
			var indexers []domain.Indexer
			for i := 0; i < 2; i++ {
				// identifier must be unique
				indexerMockData.Identifier = fmt.Sprintf("indexer%d", i)
				indexer, err := indexerRepo.Store(t.Context(), indexerMockData)
				assert.NoError(t, err)
				indexers = append(indexers, *indexer)
			}

			// Execute
			err = repo.StoreIndexerConnections(t.Context(), mockData.ID, indexers)
			assert.NoError(t, err)

			// Validate that the connections were successfully stored in the database
			connections, err := repo.FindByIndexerIdentifier(t.Context(), indexerMockData.Identifier)
			assert.NoError(t, err)
			assert.NotNil(t, connections)
			assert.NotEmpty(t, connections)

			// Cleanup
			for _, indexer := range indexers {
				_ = indexerRepo.Delete(t.Context(), int(indexer.ID))
			}
			_ = repo.Delete(t.Context(), mockData.ID)
		})

		t.Run(fmt.Sprintf("StoreIndexerConnections_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.StoreIndexerConnections(t.Context(), -1, []domain.Indexer{})
			assert.NoError(t, err) //TODO: // this should return an error.
		})
	}
}

func TestFilterRepo_StoreFilterExternal(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)

		t.Run(fmt.Sprintf("StoreFilterExternal_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			// Execute
			mockDataExternal := getMockFilterExternal()
			err = repo.StoreFilterExternal(t.Context(), mockData.ID, []domain.FilterExternal{mockDataExternal})
			assert.NoError(t, err)

			// Cleanup
			_ = repo.Delete(t.Context(), mockData.ID)
		})

		t.Run(fmt.Sprintf("StoreFilterExternal_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.StoreFilterExternal(t.Context(), -1, []domain.FilterExternal{})
			assert.NoError(t, err) // TODO: this should return an error
		})
	}
}

func TestFilterRepo_DeleteIndexerConnections(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)

		t.Run(fmt.Sprintf("DeleteIndexerConnections_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			indexerMockData := getMockIndexer()
			indexer, err := indexerRepo.Store(t.Context(), indexerMockData)
			assert.NoError(t, err)
			assert.NotNil(t, indexer)

			err = repo.StoreIndexerConnection(t.Context(), mockData.ID, int(indexer.ID))
			assert.NoError(t, err)

			// Execute
			err = repo.DeleteIndexerConnections(t.Context(), mockData.ID)
			assert.NoError(t, err)

			// Validate that the connections were successfully deleted from the database
			connections, err := repo.FindByIndexerIdentifier(t.Context(), indexerMockData.Identifier)
			assert.NoError(t, err)
			assert.Nil(t, connections)

			// Cleanup
			_ = indexerRepo.Delete(t.Context(), int(indexer.ID))
			_ = repo.Delete(t.Context(), mockData.ID)
		})

		t.Run(fmt.Sprintf("DeleteIndexerConnections_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.DeleteIndexerConnections(t.Context(), -1)
			assert.NoError(t, err) // TODO: this should return an error
		})

	}
}

func TestFilterRepo_DeleteFilterExternal(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)

		t.Run(fmt.Sprintf("DeleteFilterExternal_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			mockDataExternal := getMockFilterExternal()
			err = repo.StoreFilterExternal(t.Context(), mockData.ID, []domain.FilterExternal{mockDataExternal})
			assert.NoError(t, err)

			// Execute
			err = repo.DeleteFilterExternal(t.Context(), mockData.ID)
			assert.NoError(t, err)

			// Validate that the connections were successfully deleted from the database
			connections, err := repo.FindExternalFiltersByID(t.Context(), mockData.ID)
			assert.NoError(t, err)
			assert.Nil(t, connections)

			// Cleanup
			_ = repo.Delete(t.Context(), mockData.ID)
		})

		t.Run(fmt.Sprintf("DeleteFilterExternal_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.DeleteFilterExternal(t.Context(), -1)
			assert.NoError(t, err) // TODO: this should return an error
		})

	}
}

func TestFilterRepo_GetFilterDownloads(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewFilterRepo(log, db)
		releaseRepo := NewReleaseRepo(log, db)
		downloadClientRepo := NewDownloadClientRepo(log, db)
		actionRepo := NewActionRepo(log, db, downloadClientRepo)

		t.Run(fmt.Sprintf("GetFilterDownloads_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockFilter()
			err := repo.Store(t.Context(), mockData)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()

			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)
			assert.NotNil(t, mockClient)

			mockAction := getMockAction()
			mockAction.FilterID = mockData.ID
			mockAction.ClientID = mockClient.ID

			err = actionRepo.Store(t.Context(), mockAction)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.FilterID = int64(mockData.ID)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockData.ID

			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			// Execute
			err = repo.GetFilterDownloadCount(t.Context(), mockData)
			assert.NoError(t, err)
			assert.NotNil(t, mockData.Downloads)
			assert.Equal(t, mockData.Downloads, &domain.FilterDownloads{
				HourCount:  1,
				DayCount:   1,
				WeekCount:  1,
				MonthCount: 1,
				TotalCount: 1,
			})

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockData.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			mockFilter := &domain.Filter{ID: -1}
			err := repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				HourCount:  0,
				DayCount:   0,
				WeekCount:  0,
				MonthCount: 0,
				TotalCount: 0,
			})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Multiple_Actions [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()

			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)
			assert.NotNil(t, mockClient)

			mockAction1 := getMockAction()
			mockAction1.FilterID = mockFilter.ID
			mockAction1.ClientID = mockClient.ID

			actionErr := actionRepo.Store(t.Context(), mockAction1)
			assert.NoError(t, actionErr)

			mockAction2 := getMockAction()
			mockAction2.FilterID = mockFilter.ID
			mockAction2.ClientID = mockClient.ID

			action2Err := actionRepo.Store(t.Context(), mockAction2)
			assert.NoError(t, action2Err)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID

			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus1 := getMockReleaseActionStatus()
			mockReleaseActionStatus1.ActionID = int64(mockAction1.ID)
			mockReleaseActionStatus1.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus1.ReleaseID = mockRelease.ID

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus1)
			assert.NoError(t, err)

			mockReleaseActionStatus2 := getMockReleaseActionStatus()
			mockReleaseActionStatus2.ActionID = int64(mockAction2.ID)
			mockReleaseActionStatus2.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus2.ReleaseID = mockRelease.ID

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus2)
			assert.NoError(t, err)

			// Execute
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				HourCount:  1,
				DayCount:   1,
				WeekCount:  1,
				MonthCount: 1,
				TotalCount: 1,
			})

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction1.ID})
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction2.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_Old_Release [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			mockFilter.Shows = "nope"
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			mockClient := getMockDownloadClient()

			err = downloadClientRepo.Store(t.Context(), &mockClient)
			assert.NoError(t, err)
			assert.NotNil(t, mockClient)

			mockAction := getMockAction()
			mockAction.FilterID = mockFilter.ID
			mockAction.ClientID = mockClient.ID

			err = actionRepo.Store(t.Context(), mockAction)
			assert.NoError(t, err)

			mockAction2 := getMockAction()
			mockAction2.FilterID = mockFilter.ID
			mockAction2.ClientID = mockClient.ID

			err = actionRepo.Store(t.Context(), mockAction2)
			assert.NoError(t, err)

			mockRelease := getMockRelease()
			mockRelease.FilterID = mockFilter.ID

			err = releaseRepo.Store(t.Context(), mockRelease)
			assert.NoError(t, err)

			mockReleaseActionStatus := getMockReleaseActionStatus()
			mockReleaseActionStatus.ActionID = int64(mockAction.ID)
			mockReleaseActionStatus.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus.ReleaseID = mockRelease.ID
			mockReleaseActionStatus.Timestamp = mockReleaseActionStatus.Timestamp.AddDate(0, -1, 0)

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus)
			assert.NoError(t, err)

			mockReleaseActionStatus2 := getMockReleaseActionStatus()
			mockReleaseActionStatus2.ActionID = int64(mockAction2.ID)
			mockReleaseActionStatus2.FilterID = int64(mockFilter.ID)
			mockReleaseActionStatus2.ReleaseID = mockRelease.ID
			mockReleaseActionStatus2.Timestamp = mockReleaseActionStatus2.Timestamp.AddDate(0, -1, 0)

			err = releaseRepo.StoreReleaseActionStatus(t.Context(), mockReleaseActionStatus2)
			assert.NoError(t, err)

			// Execute
			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				HourCount:  0,
				DayCount:   0,
				WeekCount:  0,
				MonthCount: 0,
				TotalCount: 1,
			})

			// Cleanup
			_ = actionRepo.Delete(t.Context(), &domain.DeleteActionRequest{ActionId: mockAction.ID})
			_ = repo.Delete(t.Context(), mockFilter.ID)
			_ = downloadClientRepo.Delete(t.Context(), mockClient.ID)
			_ = releaseRepo.Delete(t.Context(), &domain.DeleteReleaseRequest{OlderThan: 0})
		})

		t.Run(fmt.Sprintf("GetFilterDownloads_No_Releases [%s]", dbType), func(t *testing.T) {
			// Setup
			mockFilter := getMockFilter()
			err := repo.Store(t.Context(), mockFilter)
			assert.NoError(t, err)

			err = repo.GetFilterDownloadCount(t.Context(), mockFilter)
			assert.NoError(t, err)
			assert.NotNil(t, mockFilter.Downloads)
			assert.Equal(t, mockFilter.Downloads, &domain.FilterDownloads{
				HourCount:  0,
				DayCount:   0,
				WeekCount:  0,
				MonthCount: 0,
				TotalCount: 0,
			})

			// Cleanup
			_ = repo.Delete(t.Context(), mockFilter.ID)
		})

	}
}
