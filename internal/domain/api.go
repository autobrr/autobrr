// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"time"
)

type APIRepo interface {
	Store(ctx context.Context, key *APIKey) error
	Delete(ctx context.Context, key string) error
	GetKeys(ctx context.Context) ([]APIKey, error)
}

type APIKey struct {
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
}
