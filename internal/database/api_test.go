// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"fmt"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAPIRepo_Store(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewAPIRepo(log, db)

		t.Run(fmt.Sprintf("Store_Succeeds_With_Valid_Key [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Name: "TestKey", KeyHash: domain.HashAPIKey("123"), Scopes: []string{"read", "write"}}
			err := repo.Store(ctx, key)
			require.NoError(t, err)
			assert.NotZero(t, key.ID)
			assert.NotZero(t, key.CreatedAt)

			_ = repo.Delete(ctx, key.ID)
		})

		t.Run(fmt.Sprintf("Store_Fails_If_Duplicate_Key [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{KeyHash: domain.HashAPIKey("789"), Scopes: []string{}}
			err1 := repo.Store(ctx, key)
			err2 := repo.Store(ctx, &domain.APIKey{KeyHash: key.KeyHash, Scopes: []string{}})
			require.NoError(t, err1)
			assert.Error(t, err2)

			_ = repo.Delete(ctx, key.ID)
		})
	}
}

func TestAPIRepo_Update(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewAPIRepo(log, db)

		t.Run(fmt.Sprintf("Update_Succeeds_With_Existing_Key [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Name: "TestKey", KeyHash: domain.HashAPIKey("123"), Scopes: []string{"filters:write"}}
			require.NoError(t, repo.Store(ctx, key))

			update := &domain.APIKey{ID: key.ID, Name: "Renamed", Scopes: []string{"filters:read", "lists:invoke"}}
			err := repo.Update(ctx, update)
			require.NoError(t, err)
			assert.Equal(t, key.KeyHash, update.KeyHash)
			assert.NotZero(t, update.CreatedAt)

			apiKey, err := repo.FindByID(ctx, key.ID)
			require.NoError(t, err)
			assert.Equal(t, "Renamed", apiKey.Name)
			assert.Equal(t, []string{"filters:read", "lists:invoke"}, apiKey.Scopes)

			_ = repo.Delete(ctx, key.ID)
		})

		t.Run(fmt.Sprintf("Update_Fails_If_Key_Does_Not_Exist [%s]", dbType), func(t *testing.T) {
			err := repo.Update(ctx, &domain.APIKey{ID: 9999, Name: "TestKey", Scopes: []string{"filters:write"}})
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})
	}
}

func TestAPIRepo_Delete(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewAPIRepo(log, db)

		t.Run(fmt.Sprintf("Delete_Succeeds_With_Existing_Key [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Name: "TestKey", KeyHash: domain.HashAPIKey("123"), Scopes: []string{"read", "write"}}
			require.NoError(t, repo.Store(ctx, key))

			err := repo.Delete(ctx, key.ID)
			assert.NoError(t, err)

			_, err = repo.FindByID(ctx, key.ID)
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
		})

		t.Run(fmt.Sprintf("Delete_Succeeds_If_Key_Does_Not_Exist [%s]", dbType), func(t *testing.T) {
			err := repo.Delete(ctx, 9999)
			assert.NoError(t, err)
		})
	}
}

func TestAPIRepo_List(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewAPIRepo(log, db)

		t.Run(fmt.Sprintf("List_Returns_Keys_If_Exists [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Name: "TestKey", KeyHash: domain.HashAPIKey("123"), Scopes: []string{"read", "write"}}
			require.NoError(t, repo.Store(ctx, key))

			keys, err := repo.List(ctx)
			require.NoError(t, err)
			require.Len(t, keys, 1)
			assert.Equal(t, key.ID, keys[0].ID)
			assert.Empty(t, keys[0].Key)

			_ = repo.Delete(ctx, key.ID)
		})

		t.Run(fmt.Sprintf("List_Returns_Empty_If_No_Keys [%s]", dbType), func(t *testing.T) {
			keys, err := repo.List(ctx)
			require.NoError(t, err)
			assert.Empty(t, keys)
		})
	}
}

func TestAPIRepo_FindByKeyHash(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewAPIRepo(log, db)

		t.Run(fmt.Sprintf("FindByKeyHash_Returns_Key_If_Exists [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Name: "TestKey", KeyHash: domain.HashAPIKey("123"), Scopes: []string{"read", "write"}}
			require.NoError(t, repo.Store(ctx, key))

			apiKey, err := repo.FindByKeyHash(ctx, key.KeyHash)
			require.NoError(t, err)
			assert.Equal(t, key.ID, apiKey.ID)
			assert.Equal(t, "TestKey", apiKey.Name)

			_ = repo.Delete(ctx, key.ID)
		})

		t.Run(fmt.Sprintf("FindByKeyHash_Returns_Not_Found_If_No_Key [%s]", dbType), func(t *testing.T) {
			key, err := repo.FindByKeyHash(ctx, domain.HashAPIKey("nonexistent"))
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
			assert.Nil(t, key)
		})
	}
}
