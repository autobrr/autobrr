// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"encoding/json"
)

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

func (u User) MarshalJSON() ([]byte, error) {
	type Alias User
	return json.Marshal(&struct {
		*Alias
		Password string `json:"password"`
	}{
		Password: RedactString(u.Password),
		Alias:    (*Alias)(&u),
	})
}

func (u *User) UnmarshalJSON(data []byte) error {
	type Alias User
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(u),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	// If the password appears to be redacted, don't overwrite the existing value
	if isRedactedValue(u.Password) {
		// Keep the original password by not updating it
		return nil
	}

	return nil
}

type UserLoginRequest struct {
	Username   string `json:"username"`
	Password   string `json:"password"`
	RememberMe bool   `json:"remember_me"`
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
