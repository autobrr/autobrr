// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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

func getMockProxy() *domain.Proxy {
	return &domain.Proxy{
		//ID:      0,
		Name:    "Proxy",
		Enabled: true,
		Type:    domain.ProxyTypeSocks5,
		Addr:    "socks5://127.0.0.1:1080",
		User:    "",
		Pass:    "",
		Timeout: 0,
	}
}

func TestProxyRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			proxies, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, proxies)
			assert.Equal(t, mockData.Name, proxies[0].Name)

			// Cleanup
			_ = repo.Delete(context.Background(), mockData.ID)
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Missing_or_empty_fields [%s]", dbType), func(t *testing.T) {
			mockData := domain.Proxy{}
			err := repo.Store(context.Background(), &mockData)
			assert.Error(t, err)

			proxies, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.Empty(t, proxies)
			//assert.Nil(t, proxies)

			// Cleanup
			// No cleanup needed
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Context_Timeout [%s]", dbType), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
			defer cancel()

			err := repo.Store(ctx, mockData)
			assert.Error(t, err)
		})
	}
}

func TestProxyRepo_Update(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			// Update mockData
			updatedProxy := mockData
			updatedProxy.Name = "Updated Proxy"
			updatedProxy.Enabled = false

			// Execute
			err = repo.Update(context.Background(), updatedProxy)
			assert.NoError(t, err)

			proxies, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, proxies)
			assert.Equal(t, "Updated Proxy", proxies[0].Name)
			assert.Equal(t, false, proxies[0].Enabled)

			// Cleanup
			_ = repo.Delete(context.Background(), proxies[0].ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			mockData.ID = -1
			err := repo.Update(context.Background(), mockData)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrUpdateFailed)
		})
	}
}

func TestProxyRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			proxies, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, proxies)
			assert.Equal(t, mockData.Name, proxies[0].Name)

			// Execute
			err = repo.Delete(context.Background(), proxies[0].ID)
			assert.NoError(t, err)

			// Verify that the proxy is deleted and return error ErrRecordNotFound
			proxy, err := repo.FindByID(context.Background(), proxies[0].ID)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
			assert.Nil(t, proxy)
		})

		t.Run(fmt.Sprintf("Delete_Fails_No_Record [%s]", dbType), func(t *testing.T) {
			err := repo.Delete(context.Background(), 9999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrDeleteFailed)
		})
	}
}

func TestProxyRepo_ToggleEnabled(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			proxies, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, proxies)
			assert.Equal(t, true, proxies[0].Enabled)

			// Execute
			err = repo.ToggleEnabled(context.Background(), mockData.ID, false)
			assert.NoError(t, err)

			// Verify that the proxy is updated
			proxy, err := repo.FindByID(context.Background(), proxies[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			assert.Equal(t, false, proxy.Enabled)

			// Cleanup
			_ = repo.Delete(context.Background(), proxies[0].ID)
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.ToggleEnabled(context.Background(), -1, false)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrUpdateFailed)
		})
	}
}

func TestProxyRepo_FindByID(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(context.Background(), mockData)
			assert.NoError(t, err)

			proxies, err := repo.List(context.Background())
			assert.NoError(t, err)
			assert.NotNil(t, proxies)

			// Execute
			proxy, err := repo.FindByID(context.Background(), proxies[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			assert.Equal(t, proxies[0].ID, proxy.ID)

			// Cleanup
			_ = repo.Delete(context.Background(), proxies[0].ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			// Test using an invalid ID
			proxy, err := repo.FindByID(context.Background(), -1)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound) // should return an error
			assert.Nil(t, proxy)                             // should be nil
		})

	}
}
