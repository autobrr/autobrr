// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"context"
	"strings"
	"time"
)

type APIRepo interface {
	Store(ctx context.Context, key *APIKey) error
	Delete(ctx context.Context, key string) error
	GetAllAPIKeys(ctx context.Context) ([]APIKey, error)
	GetKey(ctx context.Context, key string) (*APIKey, error)
}

type APIKey struct {
	Name      string    `json:"name"`
	Key       string    `json:"key"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
}

func RedactString(s string) string {
	if len(s) == 0 {
		return ""
	}

	return strings.Repeat("*", len(s))
}

// Helper function to check if a value appears to be redacted
func isRedactedValue(value string) bool {
	if value == "" {
		return false
	}

	// Check if the value is all asterisks
	for _, char := range value {
		if char != '*' {
			return false
		}
	}
	return true
}
