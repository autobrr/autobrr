// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"fmt"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestAPIRepo_Store(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewAPIRepo(log, db)

		t.Run(fmt.Sprintf("Store_Succeeds_With_Valid_Key [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Name: "TestKey", Key: "123", Scopes: []string{"read", "write"}}
			err := repo.Store(ctx, key)
			assert.NoError(t, err)
			assert.NotZero(t, key.CreatedAt)
			// Cleanup
			_ = repo.Delete(ctx, key.Key)
		})

		t.Run(fmt.Sprintf("Store_Fails_If_No_Name_Or_Scopes [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Key: "456"}
			err := repo.Store(ctx, key)
			assert.Error(t, err) // Should fail when trying to insert a key without scopes (null constraint)
			// Cleanup
			_ = repo.Delete(ctx, key.Key)
		})

		t.Run(fmt.Sprintf("Store_Fails_If_Duplicate_Key [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Key: "789", Scopes: []string{}}
			err1 := repo.Store(ctx, key)
			err2 := repo.Store(ctx, key)
			assert.NoError(t, err1)
			assert.Error(t, err2) // Should fail when trying to insert a duplicate key
			// Cleanup
			_ = repo.Delete(ctx, key.Key)
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
			key := &domain.APIKey{Name: "TestKey", Key: "123", Scopes: []string{"read", "write"}}
			_ = repo.Store(ctx, key)
			err := repo.Delete(ctx, key.Key)
			assert.NoError(t, err)
		})

		t.Run(fmt.Sprintf("Delete_Succeeds_If_Key_Does_Not_Exist [%s]", dbType), func(t *testing.T) {
			err := repo.Delete(ctx, "nonexistent")
			assert.NoError(t, err)
		})
	}

}

func TestAPIRepo_GetAllAPIKeys(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewAPIRepo(log, db)

		t.Run(fmt.Sprintf("GetKeys_Returns_Keys_If_Exists [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Name: "TestKey", Key: "123", Scopes: []string{"read", "write"}}
			_ = repo.Store(ctx, key)
			keys, err := repo.GetAllAPIKeys(ctx)
			assert.NoError(t, err)
			assert.Greater(t, len(keys), 0)
			// Cleanup
			_ = repo.Delete(ctx, key.Key)
		})

		t.Run(fmt.Sprintf("GetKeys_Returns_Empty_If_No_Keys [%s]", dbType), func(t *testing.T) {
			keys, err := repo.GetAllAPIKeys(ctx)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(keys))
		})
	}
}

func TestAPIRepo_GetKey(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewAPIRepo(log, db)

		t.Run(fmt.Sprintf("GetKey_Returns_Key_If_Exists [%s]", dbType), func(t *testing.T) {
			key := &domain.APIKey{Name: "TestKey", Key: "123", Scopes: []string{"read", "write"}}
			_ = repo.Store(ctx, key)
			apiKey, err := repo.GetKey(ctx, key.Key)
			assert.NoError(t, err)
			assert.NotNil(t, apiKey)
			// Cleanup
			_ = repo.Delete(ctx, key.Key)
		})

		t.Run(fmt.Sprintf("GetKeys_Returns_Empty_If_No_Keys [%s]", dbType), func(t *testing.T) {
			key, err := repo.GetKey(ctx, "nonexistent")
			assert.ErrorIs(t, err, domain.ErrRecordNotFound)
			assert.Nil(t, key)
		})
	}
}
