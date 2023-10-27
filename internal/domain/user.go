// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import "context"

type UserRepo interface {
	GetUserCount(ctx context.Context) (int, error)
	FindByUsername(ctx context.Context, username string) (*User, error)
	Store(ctx context.Context, req CreateUserRequest) error
	Update(ctx context.Context, user User) error
	Delete(ctx context.Context, username string) error
}

type User struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}
