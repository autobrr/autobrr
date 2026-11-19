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
		{name: "scoped", scopes: []string{"filters:read", "releases:write", "lists:invoke"}},
		{name: "empty", scopes: []string{}, wantErr: true},
		{name: "nil", scopes: nil, wantErr: true},
		{name: "wildcard", scopes: []string{"*"}, wantErr: true},
		{name: "wildcard_combined", scopes: []string{"*", "filters:read"}, wantErr: true},
		{name: "missing_access", scopes: []string{"filters"}, wantErr: true},
		{name: "unknown_resource", scopes: []string{"users:read"}, wantErr: true},
		{name: "unknown_access", scopes: []string{"filters:admin"}, wantErr: true},
		{name: "unsupported_access", scopes: []string{"logs:write"}, wantErr: true},
		{name: "config_write", scopes: []string{"config:write"}, wantErr: true},
		{name: "webhooks_removed", scopes: []string{"webhooks:write"}, wantErr: true},
		{name: "invoke_unsupported", scopes: []string{"filters:invoke"}, wantErr: true},
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
		{name: "wildcard_grants_nothing", scopes: []string{"*"}, args: args{APIResourceFilters, APIAccessRead}, want: false},
		{name: "read_grants_read", scopes: []string{"filters:read"}, args: args{APIResourceFilters, APIAccessRead}, want: true},
		{name: "read_denies_write", scopes: []string{"filters:read"}, args: args{APIResourceFilters, APIAccessWrite}, want: false},
		{name: "write_implies_read", scopes: []string{"filters:write"}, args: args{APIResourceFilters, APIAccessRead}, want: true},
		{name: "write_implies_invoke", scopes: []string{"lists:write"}, args: args{APIResourceLists, APIAccessInvoke}, want: true},
		{name: "invoke_grants_invoke", scopes: []string{"lists:invoke"}, args: args{APIResourceLists, APIAccessInvoke}, want: true},
		{name: "invoke_denies_read", scopes: []string{"lists:invoke"}, args: args{APIResourceLists, APIAccessRead}, want: false},
		{name: "invoke_denies_write", scopes: []string{"lists:invoke"}, args: args{APIResourceLists, APIAccessWrite}, want: false},
		{name: "read_denies_invoke", scopes: []string{"lists:read"}, args: args{APIResourceLists, APIAccessInvoke}, want: false},
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
