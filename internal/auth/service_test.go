// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package auth

import (
	"context"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/beevik/ntp"
	"github.com/pkg/errors"
	"github.com/pquerna/otp/totp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock user service
type mockUserService struct {
	mock.Mock
}

func (m *mockUserService) GetUserCount(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *mockUserService) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockUserService) CreateUser(ctx context.Context, req domain.CreateUserRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockUserService) Update(ctx context.Context, req domain.UpdateUserRequest) error {
	args := m.Called(ctx, req)
	return args.Error(0)
}

func (m *mockUserService) Delete(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

func (m *mockUserService) Store2FASecret(ctx context.Context, username string, secret string) error {
	args := m.Called(ctx, username, secret)
	return args.Error(0)
}

func (m *mockUserService) Get2FASecret(ctx context.Context, username string) (string, error) {
	args := m.Called(ctx, username)
	return args.String(0), args.Error(1)
}

func (m *mockUserService) Enable2FA(ctx context.Context, username string, secret string) error {
	args := m.Called(ctx, username, secret)
	return args.Error(0)
}

func (m *mockUserService) Disable2FA(ctx context.Context, username string) error {
	args := m.Called(ctx, username)
	return args.Error(0)
}

// mockNTPResponse creates a mock NTP response with the given offset and validation error
func mockNTPResponse(offset time.Duration, validateErr error) *ntp.Response {
	now := time.Now()
	r := &ntp.Response{
		Time:           now.Add(offset),
		RTT:            time.Millisecond * 100,
		ClockOffset:    offset,
		Stratum:        1,
		ReferenceTime:  now.Add(-time.Hour), // Set a recent reference time
		RootDelay:      time.Millisecond * 50,
		RootDispersion: time.Millisecond * 50,
		Poll:           8,
		Precision:      -20,
	}
	if validateErr != nil {
		// For testing validation errors, we set stratum to 0 which will cause
		// the real Validate() method to return an error
		r.Stratum = 0
	}
	return r
}

func TestEnable2FA_TimeSync(t *testing.T) {
	tests := []struct {
		name          string
		ntpOffset     time.Duration
		ntpError      error
		validateError error
		expectError   bool
		errorContains string
	}{
		{
			name:        "time in sync",
			ntpOffset:   5 * time.Second,
			expectError: false,
		},
		{
			name:          "time significantly off",
			ntpOffset:     45 * time.Second,
			expectError:   true,
			errorContains: "please sync your system time",
		},
		{
			name:          "ntp query fails",
			ntpError:      errors.New("ntp query failed"),
			expectError:   true,
			errorContains: "failed to query NTP server",
		},
		{
			name:          "invalid ntp response",
			validateError: errors.New("invalid response"),
			expectError:   true,
			errorContains: "invalid NTP response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock NTP query
			origQuery := internalNTPQuery
			defer func() { internalNTPQuery = origQuery }()

			internalNTPQuery = func() (*ntp.Response, error) {
				if tt.ntpError != nil {
					return nil, tt.ntpError
				}
				return mockNTPResponse(tt.ntpOffset, tt.validateError), nil
			}

			// Create service with mocked dependencies
			log := logger.New(&domain.Config{})
			userSvc := &mockUserService{}
			svc := &service{
				log:     log.With().Str("module", "auth").Logger(),
				userSvc: userSvc,
			}

			// Setup user service expectations
			if !tt.expectError {
				userSvc.On("Store2FASecret", mock.Anything, mock.Anything, mock.Anything).Return(nil)
			}

			// Test Enable2FA
			_, _, err := svc.Enable2FA(context.Background(), "testuser")

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				userSvc.AssertExpectations(t)
			}
		})
	}
}

func TestVerify2FALogin_TimeSync(t *testing.T) {
	// Generate a test TOTP secret and valid code
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" // Base32 encoded test secret
	now := time.Now()
	validCode, err := totp.GenerateCode(secret, now)
	if err != nil {
		t.Fatalf("Failed to generate TOTP code: %v", err)
	}

	tests := []struct {
		name          string
		ntpOffset     time.Duration
		ntpError      error
		validateError error
		code          string
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid code with time in sync",
			ntpOffset:   5 * time.Second,
			code:        validCode,
			expectError: false,
		},
		{
			name:          "invalid code with time off",
			ntpOffset:     45 * time.Second,
			code:          "123456",
			expectError:   true,
			errorContains: "invalid verification code",
		},
		{
			name:          "ntp query fails during validation",
			ntpError:      errors.New("ntp query failed"),
			code:          "123456",
			expectError:   true,
			errorContains: "invalid verification code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock NTP query
			origQuery := internalNTPQuery
			defer func() { internalNTPQuery = origQuery }()

			internalNTPQuery = func() (*ntp.Response, error) {
				if tt.ntpError != nil {
					return nil, tt.ntpError
				}
				return mockNTPResponse(tt.ntpOffset, tt.validateError), nil
			}

			// Create service with mocked dependencies
			log := logger.New(&domain.Config{})
			userSvc := &mockUserService{}
			svc := &service{
				log:     log.With().Str("module", "auth").Logger(),
				userSvc: userSvc,
			}

			// Setup user service expectations
			userSvc.On("Get2FASecret", mock.Anything, "testuser").Return(secret, nil)

			// Test Verify2FALogin
			err := svc.Verify2FALogin(context.Background(), "testuser", tt.code)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			userSvc.AssertExpectations(t)
		})
	}
}

func TestCheckTimeSync(t *testing.T) {
	tests := []struct {
		name           string
		ntpOffset      time.Duration
		ntpError       error
		validateError  error
		expectError    bool
		errorContains  string
		expectedOffset time.Duration
	}{
		{
			name:           "successful sync check",
			ntpOffset:      5 * time.Second,
			expectError:    false,
			expectedOffset: 5 * time.Second,
		},
		{
			name:          "ntp query fails",
			ntpError:      errors.New("ntp query failed"),
			expectError:   true,
			errorContains: "failed to query NTP server",
		},
		{
			name:          "invalid ntp response",
			validateError: errors.New("invalid response"),
			expectError:   true,
			errorContains: "invalid NTP response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock NTP query
			origQuery := internalNTPQuery
			defer func() { internalNTPQuery = origQuery }()

			internalNTPQuery = func() (*ntp.Response, error) {
				if tt.ntpError != nil {
					return nil, tt.ntpError
				}
				return mockNTPResponse(tt.ntpOffset, tt.validateError), nil
			}

			// Create service
			log := logger.New(&domain.Config{})
			svc := &service{
				log:     log.With().Str("module", "auth").Logger(),
				userSvc: &mockUserService{},
			}

			// Test checkTimeSync
			offset, err := svc.checkTimeSync()

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOffset, offset)
			}
		})
	}
}
