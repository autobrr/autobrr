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
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("Store_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			proxies, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, proxies)
			assert.Equal(t, mockData.Name, proxies[0].Name)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("Store_Fails_With_Missing_or_empty_fields [%s]", dbType), func(t *testing.T) {
			mockData := domain.Proxy{}
			err := repo.Store(ctx, &mockData)
			assert.Error(t, err)

			proxies, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.Empty(t, proxies)
			//assert.Nil(t, proxies)

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

func TestProxyRepo_Update(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("Update_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			// Update mockData
			updatedProxy := mockData
			updatedProxy.Name = "Updated Proxy"
			updatedProxy.Enabled = false

			// Execute
			err = repo.Update(ctx, updatedProxy)
			assert.NoError(t, err)

			proxies, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, proxies)
			assert.Equal(t, "Updated Proxy", proxies[0].Name)
			assert.Equal(t, false, proxies[0].Enabled)

			// Cleanup
			_ = repo.Delete(ctx, proxies[0].ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			mockData.ID = -1
			err := repo.Update(ctx, mockData)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrUpdateFailed)
		})
	}
}

func TestProxyRepo_Delete(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("Delete_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			proxies, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, proxies)
			assert.Equal(t, mockData.Name, proxies[0].Name)

			// Execute
			err = repo.Delete(ctx, proxies[0].ID)
			assert.NoError(t, err)

			// Verify that the proxy is deleted and return error ErrRecordNotFound
			proxy, err := repo.FindByID(ctx, proxies[0].ID)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
			assert.Nil(t, proxy)
		})

		t.Run(fmt.Sprintf("Delete_Fails_No_Record [%s]", dbType), func(t *testing.T) {
			err := repo.Delete(ctx, 9999)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrDeleteFailed)
		})

		t.Run(fmt.Sprintf("Delete_Detaches_Indexer_And_Network [%s]", dbType), func(t *testing.T) {
			// Setup
			indexerRepo := NewIndexerRepo(log, db)
			ircRepo := NewIrcRepo(log, db)

			mockData := getMockProxy()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			mockIndexer := getMockIndexer()
			mockIndexer.UseProxy = true
			mockIndexer.ProxyID = mockData.ID
			indexer, err := indexerRepo.Store(ctx, mockIndexer)
			assert.NoError(t, err)

			network := getMockIrcNetwork()
			network.UseProxy = true
			network.ProxyId = mockData.ID
			err = ircRepo.StoreNetwork(ctx, &network)
			assert.NoError(t, err)

			// Execute
			err = repo.Delete(ctx, mockData.ID)
			assert.NoError(t, err)

			// Verify
			detachedIndexer, err := indexerRepo.FindByID(ctx, int(indexer.ID))
			assert.NoError(t, err)
			assert.False(t, detachedIndexer.UseProxy)
			assert.Equal(t, int64(0), detachedIndexer.ProxyID)

			detachedNetwork, err := ircRepo.GetNetworkByID(ctx, network.ID)
			assert.NoError(t, err)
			assert.False(t, detachedNetwork.UseProxy)
			assert.Equal(t, int64(0), detachedNetwork.ProxyId)

			// Cleanup
			_ = ircRepo.DeleteNetwork(ctx, network.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
		})
	}
}

func TestProxyRepo_ToggleEnabled(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("ToggleEnabled_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			proxies, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, proxies)
			assert.Equal(t, true, proxies[0].Enabled)

			// Execute
			err = repo.ToggleEnabled(ctx, mockData.ID, false)
			assert.NoError(t, err)

			// Verify that the proxy is updated
			proxy, err := repo.FindByID(ctx, proxies[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			assert.Equal(t, false, proxy.Enabled)

			// Cleanup
			_ = repo.Delete(ctx, proxies[0].ID)
		})

		t.Run(fmt.Sprintf("ToggleEnabled_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			err := repo.ToggleEnabled(ctx, -1, false)
			assert.Error(t, err)
			assert.ErrorIs(t, err, domain.ErrUpdateFailed)
		})
	}
}

func TestProxyRepo_FindByID(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		mockData := getMockProxy()

		t.Run(fmt.Sprintf("FindByID_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			proxies, err := repo.List(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, proxies)

			// Execute
			proxy, err := repo.FindByID(ctx, proxies[0].ID)
			assert.NoError(t, err)
			assert.NotNil(t, proxy)
			assert.Equal(t, proxies[0].ID, proxy.ID)

			// Cleanup
			_ = repo.Delete(ctx, proxies[0].ID)
		})

		t.Run(fmt.Sprintf("FindByID_Fails_Invalid_ID [%s]", dbType), func(t *testing.T) {
			// Test using an invalid ID
			proxy, err := repo.FindByID(ctx, -1)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound) // should return an error
			assert.Nil(t, proxy)                             // should be nil
		})

	}
}

func TestProxyRepo_Usage(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewProxyRepo(log, db)
		indexerRepo := NewIndexerRepo(log, db)
		ircRepo := NewIrcRepo(log, db)
		feedRepo := NewFeedRepo(log, db)

		t.Run(fmt.Sprintf("Usage_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockProxy()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			proxiedIndexer := getMockIndexer()
			proxiedIndexer.Name = "proxied indexer"
			proxiedIndexer.Identifier = "proxied-indexer"
			proxiedIndexer.UseProxy = true
			proxiedIndexer.ProxyID = mockData.ID
			indexer, err := indexerRepo.Store(ctx, proxiedIndexer)
			assert.NoError(t, err)

			directIndexer := getMockIndexer()
			directIndexer.Name = "direct indexer"
			directIndexer.Identifier = "direct-indexer"
			directIndexer.ProxyID = mockData.ID
			unusedIndexer, err := indexerRepo.Store(ctx, directIndexer)
			assert.NoError(t, err)

			network := getMockIrcNetwork()
			network.UseProxy = true
			network.ProxyId = mockData.ID
			err = ircRepo.StoreNetwork(ctx, &network)
			assert.NoError(t, err)

			feed := getMockFeed()
			feed.IndexerID = int(indexer.ID)
			err = feedRepo.Store(ctx, feed)
			assert.NoError(t, err)

			// Execute
			usage, err := repo.Usage(ctx, mockData.ID)
			assert.NoError(t, err)

			// Verify
			assert.Equal(t, []domain.ProxyUsageItem{{ID: indexer.ID, Name: proxiedIndexer.Name}}, usage.Indexers)
			assert.Equal(t, []domain.ProxyUsageItem{{ID: network.ID, Name: network.Name}}, usage.IrcNetworks)
			assert.Equal(t, []domain.ProxyUsageItem{{ID: int64(feed.ID), Name: feed.Name}}, usage.Feeds)

			// Cleanup
			_ = feedRepo.Delete(ctx, feed.ID)
			_ = ircRepo.DeleteNetwork(ctx, network.ID)
			_ = indexerRepo.Delete(ctx, int(indexer.ID))
			_ = indexerRepo.Delete(ctx, int(unusedIndexer.ID))
			_ = repo.Delete(ctx, mockData.ID)
		})

		t.Run(fmt.Sprintf("Usage_Empty_For_Unused_Proxy [%s]", dbType), func(t *testing.T) {
			// Setup
			mockData := getMockProxy()
			err := repo.Store(ctx, mockData)
			assert.NoError(t, err)

			// Execute
			usage, err := repo.Usage(ctx, mockData.ID)
			assert.NoError(t, err)

			// Verify
			assert.Empty(t, usage.Indexers)
			assert.Empty(t, usage.IrcNetworks)
			assert.Empty(t, usage.Feeds)

			// Cleanup
			_ = repo.Delete(ctx, mockData.ID)
		})
	}
}
