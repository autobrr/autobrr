// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build integration

package database

import (
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListRepo_List(t *testing.T) {
	ctx := t.Context()

	for dbType, testDb := range testDBs {
		db := testDb.db
		log := setupLoggerForTest()
		repo := NewListRepo(log, db)

		t.Run(fmt.Sprintf("List_Filters_By_Params [%s]", dbType), func(t *testing.T) {
			seed := []struct {
				name    string
				enabled bool
				status  domain.ListRefreshStatus
			}{
				{name: "enabled-error", enabled: true, status: domain.ListRefreshStatusError},
				{name: "enabled-success", enabled: true, status: domain.ListRefreshStatusSuccess},
				{name: "disabled-error", enabled: false, status: domain.ListRefreshStatusError},
				{name: "never-refreshed", enabled: true},
			}

			for _, item := range seed {
				list := &domain.List{
					Name:        item.name,
					Type:        domain.ListTypePlaintext,
					Enabled:     item.enabled,
					URL:         "https://example.com/list.txt",
					Headers:     []string{},
					TagsInclude: []string{},
					TagsExclude: []string{},
				}
				require.NoError(t, repo.Store(ctx, list))

				t.Cleanup(func() {
					_ = repo.Delete(ctx, list.ID)
				})

				if item.status != "" {
					list.LastRefreshTime = time.Now()
					list.LastRefreshStatus = item.status
					require.NoError(t, repo.UpdateLastRefresh(ctx, list))
				}
			}

			tests := []struct {
				name   string
				params domain.ListQueryParams
				want   []string
			}{
				{name: "no_params", params: domain.ListQueryParams{}, want: []string{"disabled-error", "enabled-error", "enabled-success", "never-refreshed"}},
				{name: "enabled", params: domain.ListQueryParams{Enabled: new(true)}, want: []string{"enabled-error", "enabled-success", "never-refreshed"}},
				{name: "disabled", params: domain.ListQueryParams{Enabled: new(false)}, want: []string{"disabled-error"}},
				{name: "status_error", params: domain.ListQueryParams{LastRefreshStatus: domain.ListRefreshStatusError}, want: []string{"disabled-error", "enabled-error"}},
				{name: "enabled_status_error", params: domain.ListQueryParams{Enabled: new(true), LastRefreshStatus: domain.ListRefreshStatusError}, want: []string{"enabled-error"}},
			}
			for _, tt := range tests {
				lists, err := repo.List(ctx, tt.params)
				require.NoError(t, err, tt.name)

				names := make([]string, 0, len(lists))
				for _, list := range lists {
					names = append(names, list.Name)
				}

				assert.Equal(t, tt.want, names, tt.name)
			}
		})
	}
}
