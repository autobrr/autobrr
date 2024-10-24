// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import "context"

type UserRepo interface {
	GetUserCount(ctx context.Context) (int, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Store(ctx context.Context, req CreateUserRequest) error
	Update(ctx context.Context, req UpdateUserRequest) error
	Delete(ctx context.Context, username string) error
	Enable2FA(ctx context.Context, username string, secret string) error
	Verify2FA(ctx context.Context, username string, code string) error
	Disable2FA(ctx context.Context, username string) error
	Get2FASecret(ctx context.Context, username string) (string, error)
	Store2FASecret(ctx context.Context, username string, secret string) error
}

type User struct {
	ID            int    `json:"id"`
	Username      string `json:"username"`
	Password      string `json:"password"`
	TwoFactorAuth bool   `json:"two_factor_auth"`
	TFASecret     string `json:"-"` // Secret is never exposed to JSON
}

type UpdateUserRequest struct {
	UsernameCurrent string `json:"username_username"`
	UsernameNew     string `json:"username_new"`
	PasswordCurrent string `json:"password_current"`
	PasswordNew     string `json:"password_new"`
	PasswordNewHash string `json:"-"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Store2FASecret struct {
	Username string `json:"username"`
	Secret   string `json:"secret"`
	Code     string `json:"code"`
}

type Verify2FARequest struct {
	Username string `json:"username"`
	Code     string `json:"code"`
}
