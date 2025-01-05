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

func getMockAction() *domain.Action {
	return &domain.Action{
		Name:                     "randomAction",
		Type:                     domain.ActionTypeTest,
		Enabled:                  true,
		ExecCmd:                  "/home/user/Downloads/test.sh",
		ExecArgs:                 "WGET_URL",
		WatchFolder:              "/home/user/Downloads",
		Category:                 "HD, 720p",
		Tags:                     "P2P, x264",
		Label:                    "testLabel",
		SavePath:                 "/home/user/Downloads",
		Paused:                   false,
		IgnoreRules:              false,
		SkipHashCheck:            false,
		FirstLastPiecePrio:       false,
		ContentLayout:            domain.ActionContentLayoutOriginal,
		LimitUploadSpeed:         0,
		LimitDownloadSpeed:       0,
		LimitRatio:               0,
		LimitSeedTime:            0,
		ReAnnounceSkip:           false,
		ReAnnounceDelete:         false,
		ReAnnounceInterval:       0,
		ReAnnounceMaxAttempts:    0,
		WebhookHost:              "http://localhost:8080",
		WebhookType:              "test",
		WebhookMethod:            "POST",
		WebhookData:              "testData",
		WebhookHeaders:           []string{"testHeader"},
		ExternalDownloadClientID: 21,
		FilterID:                 1,
		ClientID:                 1,
	}
}

func TestActionRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		repo := NewActionRepo(log, db, downloadClientRepo)
		mockData := getMockAction()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
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

			mockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Actual test for Store
			err = repo.Store(context.Background(), mockData)
			assert.NoError(t, err)
			assert.NotNil(t, mockData)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: mockData.ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("Store_Succeeds_With_Missing_or_empty_fields [%s]", dbType), func(t *testing.T) {
			mockData := &domain.Action{}
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: mockData.ID})
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Invalid_ClientID [%s]", dbType), func(t *testing.T) {
			mockData := getMockAction()
			mockData.ClientID = 9999
			err := repo.Store(context.Background(), mockData)
			assert.Error(t, err)
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			mockData := getMockAction()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			err := repo.Store(ctx, mockData)
			assert.Error(t, err)
		})
	}
}

func TestActionRepo_StoreFilterActions(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		repo := NewActionRepo(log, db, downloadClientRepo)
		mockData := getMockAction()

		t.Run(fmt.Sprintf("StoreFilterActions_Succeeds [%s]", dbType), func(t *testing.T) {
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

			mockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID

			// Actual test for StoreFilterActions
			createdActions, err := repo.StoreFilterActions(context.Background(), int64(createdFilters[0].ID), []*domain.Action{mockData})

			assert.NoError(t, err)
			assert.NotNil(t, createdActions)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdActions[0].ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("StoreFilterActions_Fails_Invalid_FilterID [%s]", dbType), func(t *testing.T) {
			_, err := repo.StoreFilterActions(context.Background(), 9999, []*domain.Action{mockData})
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("StoreFilterActions_Fails_Empty_Actions_Array [%s]", dbType), func(t *testing.T) {
			// Setup
			err := filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			_, err = repo.StoreFilterActions(context.Background(), int64(createdFilters[0].ID), []*domain.Action{})
			assert.NoError(t, err)

			// Cleanup
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("StoreFilterActions_Fails_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			err := filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			_, err = repo.StoreFilterActions(ctx, int64(createdFilters[0].ID), []*domain.Action{mockData})
			assert.Error(t, err)

			// Cleanup
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
		})
	}
}

func TestActionRepo_FindByFilterID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		repo := NewActionRepo(log, db, downloadClientRepo)
		mockData := getMockAction()

		t.Run(fmt.Sprintf("FindByFilterID_Succeeds [%s]", dbType), func(t *testing.T) {
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

			mockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID
			createdActions, err := repo.StoreFilterActions(context.Background(), int64(createdFilters[0].ID), []*domain.Action{mockData})
			assert.NoError(t, err)

			// Actual test for FindByFilterID
			actions, err := repo.FindByFilterID(context.Background(), createdFilters[0].ID, nil, false)
			assert.NoError(t, err)
			assert.NotNil(t, actions)
			assert.Equal(t, 1, len(actions))

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdActions[0].ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("FindByFilterID_Fails_No_Actions [%s]", dbType), func(t *testing.T) {
			// Setup
			err := filterRepo.Store(context.Background(), getMockFilter())
			assert.NoError(t, err)

			createdFilters, err := filterRepo.ListFilters(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, createdFilters)

			// Actual test for FindByFilterID
			actions, err := repo.FindByFilterID(context.Background(), createdFilters[0].ID, nil, false)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(actions))

			// Cleanup
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
		})

		t.Run(fmt.Sprintf("FindByFilterID_Succeeds_With_Invalid_FilterID [%s]", dbType), func(t *testing.T) {
			actions, err := repo.FindByFilterID(context.Background(), 9999, nil, false) // 9999 is an invalid filter ID
			assert.NoError(t, err)
			assert.NotNil(t, actions)
			assert.Equal(t, 0, len(actions))
		})

		t.Run(fmt.Sprintf("FindByFilterID_Fails_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			actions, err := repo.FindByFilterID(ctx, 1, nil, false)
			assert.Error(t, err)
			assert.Nil(t, actions)
		})
	}
}

func TestActionRepo_List(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		repo := NewActionRepo(log, db, downloadClientRepo)
		mockData := getMockAction()

		t.Run(fmt.Sprintf("List_Succeeds [%s]", dbType), func(t *testing.T) {
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

			mockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID
			createdActions, err := repo.StoreFilterActions(context.Background(), int64(createdFilters[0].ID), []*domain.Action{mockData})
			assert.NoError(t, err)

			// Actual test for List
			actions, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, actions)
			assert.GreaterOrEqual(t, len(actions), 1)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdActions[0].ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("List_Fails_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			actions, err := repo.List(ctx)
			assert.Error(t, err)
			assert.Nil(t, actions)
		})
	}
}

func TestActionRepo_Get(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		repo := NewActionRepo(log, db, downloadClientRepo)
		mockData := getMockAction()

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

			mockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID
			createdActions, err := repo.StoreFilterActions(context.Background(), int64(createdFilters[0].ID), []*domain.Action{mockData})
			assert.NoError(t, err)

			// Actual test for Get
			action, err := repo.Get(context.Background(), &domain.GetActionRequest{Id: createdActions[0].ID})
			assert.NoError(t, err)
			assert.NotNil(t, action)
			assert.Equal(t, createdActions[0].ID, action.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdActions[0].ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("Get_Fails_No_Record [%s]", dbType), func(t *testing.T) {
			action, err := repo.Get(context.Background(), &domain.GetActionRequest{Id: 9999})
			assert.Error(t, err)
			assert.Equal(t, domain.ErrRecordNotFound, err)
			assert.Nil(t, action)
		})

		t.Run(fmt.Sprintf("Get_Fails_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			action, err := repo.Get(ctx, &domain.GetActionRequest{Id: 1})
			assert.Error(t, err)
			assert.Nil(t, action)
		})
	}
}

func TestActionRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		repo := NewActionRepo(log, db, downloadClientRepo)
		mockData := getMockAction()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
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

			mockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID
			createdActions, err := repo.StoreFilterActions(context.Background(), int64(createdFilters[0].ID), []*domain.Action{mockData})
			assert.NoError(t, err)

			// Actual test for Delete
			err = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdActions[0].ID})
			assert.NoError(t, err)

			// Verify that the record was actually deleted
			action, err := repo.Get(context.Background(), &domain.GetActionRequest{Id: createdActions[0].ID})
			assert.Error(t, err)
			assert.Equal(t, domain.ErrRecordNotFound, err)
			assert.Nil(t, action)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdActions[0].ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("Delete_Fails_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			err := repo.Delete(ctx, &domain.DeleteActionRequest{ActionId: 1})
			assert.Error(t, err)
		})

	}
}

func TestActionRepo_DeleteByFilterID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		repo := NewActionRepo(log, db, downloadClientRepo)
		mockData := getMockAction()

		t.Run(fmt.Sprintf("DeleteByFilterID_Succeeds [%s]", dbType), func(t *testing.T) {
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

			mockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID
			createdActions, err := repo.StoreFilterActions(context.Background(), int64(createdFilters[0].ID), []*domain.Action{mockData})
			assert.NoError(t, err)

			err = repo.DeleteByFilterID(context.Background(), mockData.FilterID)
			assert.NoError(t, err)

			// Verify that actions with the given filterID are actually deleted
			action, err := repo.Get(context.Background(), &domain.GetActionRequest{Id: createdActions[0].ID})
			assert.Error(t, err)
			assert.Equal(t, domain.ErrRecordNotFound, err)
			assert.Nil(t, action)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdActions[0].ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("DeleteByFilterID_Fails_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			err := repo.DeleteByFilterID(ctx, mockData.FilterID)
			assert.Error(t, err)
		})
	}
}

func TestActionRepo_ToggleEnabled(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		downloadClientRepo := NewDownloadClientRepo(log, db)
		filterRepo := NewFilterRepo(log, db)
		repo := NewActionRepo(log, db, downloadClientRepo)
		mockData := getMockAction()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
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

			mockData.ClientID = mock.ID
			mockData.FilterID = createdFilters[0].ID
			mockData.Enabled = false
			createdActions, err := repo.StoreFilterActions(context.Background(), int64(createdFilters[0].ID), []*domain.Action{mockData})
			assert.NoError(t, err)

			// Actual test for ToggleEnabled
			err = repo.ToggleEnabled(createdActions[0].ID)
			assert.NoError(t, err)

			// Verify that the record was actually updated
			action, err := repo.Get(context.Background(), &domain.GetActionRequest{Id: createdActions[0].ID})
			assert.NoError(t, err)
			assert.Equal(t, true, action.Enabled)

			// Cleanup
			_ = repo.Delete(context.Background(), &domain.DeleteActionRequest{ActionId: createdActions[0].ID})
			_ = filterRepo.Delete(context.Background(), createdFilters[0].ID)
			_ = downloadClientRepo.Delete(context.Background(), mock.ID)
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_No_Record [%s]", dbType), func(t *testing.T) {
			err := repo.ToggleEnabled(9999)
			assert.Error(t, err)
		})

	}
}
