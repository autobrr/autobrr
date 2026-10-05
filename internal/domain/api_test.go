// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIKey_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		scopes  []string
		wantErr bool
	}{
		{name: "full_access", scopes: []string{"*"}},
		{name: "scoped", scopes: []string{"filters:read", "releases:write", "webhooks:write"}},
		{name: "empty", scopes: []string{}, wantErr: true},
		{name: "nil", scopes: nil, wantErr: true},
		{name: "full_access_combined", scopes: []string{"*", "filters:read"}, wantErr: true},
		{name: "missing_access", scopes: []string{"filters"}, wantErr: true},
		{name: "unknown_resource", scopes: []string{"users:read"}, wantErr: true},
		{name: "unknown_access", scopes: []string{"filters:admin"}, wantErr: true},
		{name: "unsupported_access", scopes: []string{"logs:write"}, wantErr: true},
		{name: "webhooks_read", scopes: []string{"webhooks:read"}, wantErr: true},
		{name: "duplicate_resource", scopes: []string{"filters:read", "filters:write"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &APIKey{Name: "mock", Scopes: tt.scopes}
			err := k.Validate()
			if tt.wantErr {
				assert.ErrorIs(t, err, ErrInvalidAPIKeyScopes)
				return
			}

			assert.NoError(t, err)
		})
	}
}

func TestAPIKey_HasAccess(t *testing.T) {
	t.Parallel()

	type args struct {
		resource APIResource
		access   APIAccess
	}
	tests := []struct {
		name   string
		scopes []string
		args   args
		want   bool
	}{
		{name: "full_access_read", scopes: []string{"*"}, args: args{APIResourceFilters, APIAccessRead}, want: true},
		{name: "full_access_write", scopes: []string{"*"}, args: args{APIResourceConfig, APIAccessWrite}, want: true},
		{name: "read_grants_read", scopes: []string{"filters:read"}, args: args{APIResourceFilters, APIAccessRead}, want: true},
		{name: "read_denies_write", scopes: []string{"filters:read"}, args: args{APIResourceFilters, APIAccessWrite}, want: false},
		{name: "write_implies_read", scopes: []string{"filters:write"}, args: args{APIResourceFilters, APIAccessRead}, want: true},
		{name: "other_resource", scopes: []string{"filters:write"}, args: args{APIResourceReleases, APIAccessRead}, want: false},
		{name: "empty", scopes: []string{}, args: args{APIResourceFilters, APIAccessRead}, want: false},
		{name: "malformed", scopes: []string{"filters"}, args: args{APIResourceFilters, APIAccessRead}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &APIKey{Scopes: tt.scopes}
			assert.Equalf(t, tt.want, k.HasAccess(tt.args.resource, tt.args.access), "HasAccess(%v, %v)", tt.args.resource, tt.args.access)
		})
	}
}
