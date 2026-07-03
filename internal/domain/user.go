// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"encoding/json"
)

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
