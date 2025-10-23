// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package release

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCleanupJobKey_ToString(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		expected string
	}{
		{
			name:     "ID 1",
			id:       1,
			expected: "release-cleanup-1",
		},
		{
			name:     "ID 42",
			id:       42,
			expected: "release-cleanup-42",
		},
		{
			name:     "ID 999",
			id:       999,
			expected: "release-cleanup-999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cleanupJobKey{id: tt.id}
			assert.Equal(t, tt.expected, key.ToString())
		})
	}
}
