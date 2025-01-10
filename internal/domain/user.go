// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import "context"

type UserRepo interface {
	GetUserCount(ctx context.Context) (int, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Store(ctx context.Context, req CreateUserRequest) error
	Update(ctx context.Context, req UpdateUserRequest) error
	Delete(ctx context.Context, username string) error
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
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
