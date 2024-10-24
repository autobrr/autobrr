// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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

func getMockUser() domain.User {
	return domain.User{
		ID:       0,
		Username: "AkenoHimejima",
		Password: "password",
	}
}

func TestUserRepo_Store(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewUserRepo(log, db)

		userMockData := getMockUser()

		t.Run(fmt.Sprintf("StoreUser_Succeeds [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.Store(context.Background(), domain.CreateUserRequest{
				Username: userMockData.Username,
				Password: userMockData.Password,
			})

			// Verify
			assert.NoError(t, err)

			// Cleanup
			_ = repo.Delete(context.Background(), userMockData.Username)
		})
	}
}

func TestUserRepo_Update(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewUserRepo(log, db)

		user := getMockUser()
		err := repo.Store(context.Background(), domain.CreateUserRequest{
			Username: user.Username,
			Password: user.Password,
		})
		assert.NoError(t, err)

		storedUser, err := repo.FindByUsername(context.Background(), user.Username)
		assert.NoError(t, err)
		user.ID = storedUser.ID

		t.Run(fmt.Sprintf("UpdateUser_Succeeds [%s]", dbType), func(t *testing.T) {
			// Update the user
			newPassword := "newPassword123"
			user.Password = newPassword
			req := domain.UpdateUserRequest{
				UsernameCurrent: user.Username,
				PasswordNewHash: newPassword,
			}
			err := repo.Update(context.Background(), req)
			assert.NoError(t, err)

			// Verify
			updatedUser, err := repo.FindByUsername(context.Background(), user.Username)
			assert.NoError(t, err)
			assert.Equal(t, newPassword, updatedUser.Password)

			// Cleanup
			_ = repo.Delete(context.Background(), updatedUser.Username)
		})
	}
}

func TestUserRepo_GetUserCount(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewUserRepo(log, db)

		t.Run(fmt.Sprintf("GetUserCount_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			initialCount, err := repo.GetUserCount(context.Background())
			assert.NoError(t, err)

			user := getMockUser()
			err = repo.Store(context.Background(), domain.CreateUserRequest{
				Username: user.Username,
				Password: user.Password,
			})
			assert.NoError(t, err)

			// Verify
			updatedCount, err := repo.GetUserCount(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, initialCount+1, updatedCount)

			// Cleanup
			_ = repo.Delete(context.Background(), user.Username)
		})
	}
}

func TestUserRepo_FindByUsername(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewUserRepo(log, db)

		userMockData := getMockUser()

		t.Run(fmt.Sprintf("FindByUsername_Succeeds [%s]", dbType), func(t *testing.T) {
			// Execute
			err := repo.Store(context.Background(), domain.CreateUserRequest{
				Username: userMockData.Username,
				Password: userMockData.Password,
			})
			assert.NoError(t, err)

			// Verify
			user, err := repo.FindByUsername(context.Background(), userMockData.Username)
			assert.NoError(t, err)
			assert.NotNil(t, user)
			assert.Equal(t, userMockData.Username, user.Username)

			// Cleanup
			_ = repo.Delete(context.Background(), userMockData.Username)
		})
	}
}

func TestUserRepo_Delete(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewUserRepo(log, db)

		user := getMockUser()
		err := repo.Store(context.Background(), domain.CreateUserRequest{
			Username: user.Username,
			Password: user.Password,
		})
		assert.NoError(t, err)

		t.Run(fmt.Sprintf("DeleteUser_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			err := repo.Delete(context.Background(), user.Username)
			assert.NoError(t, err)

			// Verify
			_, err = repo.FindByUsername(context.Background(), user.Username)
			assert.Error(t, err)
			assert.Equal(t, domain.ErrRecordNotFound, err)
		})
	}
}

func TestUserRepo_2FA(t *testing.T) {
	for dbType, db := range testDBs {
		log := setupLoggerForTest()

		repo := NewUserRepo(log, db)

		user := getMockUser()
		err := repo.Store(context.Background(), domain.CreateUserRequest{
			Username: user.Username,
			Password: user.Password,
		})
		assert.NoError(t, err)

		t.Run(fmt.Sprintf("Store2FASecret_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			secret := "TESTSECRET123"

			// Store secret without enabling 2FA
			err := repo.Store2FASecret(context.Background(), user.Username, secret)
			assert.NoError(t, err)

			// Verify secret is stored but 2FA is not enabled
			storedUser, err := repo.FindByUsername(context.Background(), user.Username)
			assert.NoError(t, err)
			assert.Equal(t, secret, storedUser.TFASecret)
			assert.False(t, storedUser.TwoFactorAuth)
		})

		t.Run(fmt.Sprintf("Enable2FA_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			secret := "TESTSECRET456"

			// Enable 2FA
			err := repo.Enable2FA(context.Background(), user.Username, secret)
			assert.NoError(t, err)

			// Verify 2FA is enabled and secret is stored
			storedUser, err := repo.FindByUsername(context.Background(), user.Username)
			assert.NoError(t, err)
			assert.Equal(t, secret, storedUser.TFASecret)
			assert.True(t, storedUser.TwoFactorAuth)
		})

		t.Run(fmt.Sprintf("Get2FASecret_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			secret := "TESTSECRET789"
			err := repo.Enable2FA(context.Background(), user.Username, secret)
			assert.NoError(t, err)

			// Get secret
			storedSecret, err := repo.Get2FASecret(context.Background(), user.Username)
			assert.NoError(t, err)
			assert.Equal(t, secret, storedSecret)
		})

		t.Run(fmt.Sprintf("Disable2FA_Succeeds [%s]", dbType), func(t *testing.T) {
			// Setup
			secret := "TESTSECRET101112"
			err := repo.Enable2FA(context.Background(), user.Username, secret)
			assert.NoError(t, err)

			// Disable 2FA
			err = repo.Disable2FA(context.Background(), user.Username)
			assert.NoError(t, err)

			// Verify 2FA is disabled and secret is cleared
			storedUser, err := repo.FindByUsername(context.Background(), user.Username)
			assert.NoError(t, err)
			assert.Empty(t, storedUser.TFASecret)
			assert.False(t, storedUser.TwoFactorAuth)
		})

		// Cleanup
		_ = repo.Delete(context.Background(), user.Username)
	}
}
