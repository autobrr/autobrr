// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/downloader"
	"github.com/autobrr/autobrr/pkg/arr/lidarr"
	"github.com/autobrr/autobrr/pkg/arr/radarr"
	"github.com/autobrr/autobrr/pkg/arr/readarr"
	"github.com/autobrr/autobrr/pkg/arr/sonarr"
	"github.com/autobrr/autobrr/pkg/arr/sportarr"
	"github.com/autobrr/autobrr/pkg/arr/whisparr"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

type mockDownloaderService struct {
	instance *downloader.Instance
}

func (m *mockDownloaderService) GetInstance(_ context.Context, _ int32) (*downloader.Instance, error) {
	return m.instance, nil
}

func TestService_getProcessor(t *testing.T) {
	type fields struct {
		clientType domain.DownloaderType
		client     any
	}
	type args struct {
		list *domain.List
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:    "lidarr",
			fields:  fields{clientType: domain.DownloaderTypeLidarr, client: &lidarr.Client{}},
			args:    args{list: &domain.List{Type: domain.ListTypeLidarr, ClientID: 1}},
			wantErr: assert.NoError,
		},
		{
			name:    "radarr",
			fields:  fields{clientType: domain.DownloaderTypeRadarr, client: &radarr.Client{}},
			args:    args{list: &domain.List{Type: domain.ListTypeRadarr, ClientID: 1}},
			wantErr: assert.NoError,
		},
		{
			name:    "readarr",
			fields:  fields{clientType: domain.DownloaderTypeReadarr, client: &readarr.Client{}},
			args:    args{list: &domain.List{Type: domain.ListTypeReadarr, ClientID: 1}},
			wantErr: assert.NoError,
		},
		{
			name:    "sonarr",
			fields:  fields{clientType: domain.DownloaderTypeSonarr, client: &sonarr.Client{}},
			args:    args{list: &domain.List{Type: domain.ListTypeSonarr, ClientID: 1}},
			wantErr: assert.NoError,
		},
		{
			name:    "sportarr",
			fields:  fields{clientType: domain.DownloaderTypeSportarr, client: &sportarr.Client{}},
			args:    args{list: &domain.List{Type: domain.ListTypeSportarr, ClientID: 1}},
			wantErr: assert.NoError,
		},
		{
			name:    "whisparr",
			fields:  fields{clientType: domain.DownloaderTypeWhisparr, client: &whisparr.Client{}},
			args:    args{list: &domain.List{Type: domain.ListTypeWhisparr, ClientID: 1}},
			wantErr: assert.NoError,
		},
		{
			name:    "client type mismatch",
			fields:  fields{clientType: domain.DownloaderTypeRadarr, client: &radarr.Client{}},
			args:    args{list: &domain.List{Type: domain.ListTypeSonarr, ClientID: 1}},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &domain.Downloader{ID: 1, Type: tt.fields.clientType, Enabled: true}
			s := &Service{
				log:           zerolog.Nop(),
				downloaderSvc: &mockDownloaderService{instance: downloader.NewInstance(cfg, tt.fields.client)},
			}

			got, err := s.getProcessor(t.Context(), tt.args.list)
			if !tt.wantErr(t, err) {
				return
			}
			if err == nil {
				assert.NotNil(t, got)
			}
		})
	}
}
