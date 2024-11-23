// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDownloadClient_qbitBuildLegacyHost(t *testing.T) {
	t.Parallel()
	type fields struct {
		ID            int32
		Name          string
		Type          DownloadClientType
		Enabled       bool
		Host          string
		Port          int
		TLS           bool
		TLSSkipVerify bool
		Username      string
		Password      string
		Settings      DownloadClientSettings
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "build_url_1",
			fields: fields{
				Host:          "https://qbit.domain.ltd",
				Port:          0,
				Username:      "",
				Password:      "",
				TLS:           true,
				TLSSkipVerify: false,
			},
			want: "https://qbit.domain.ltd",
		},
		{
			name: "build_url_2",
			fields: fields{
				Host:          "http://qbit.domain.ltd",
				Port:          0,
				Username:      "",
				Password:      "",
				TLS:           false,
				TLSSkipVerify: false,
			},
			want: "http://qbit.domain.ltd",
		},
		{
			name: "build_url_3",
			fields: fields{
				Host:          "https://qbit.domain.ltd:8080",
				Port:          0,
				Username:      "",
				Password:      "",
				TLS:           true,
				TLSSkipVerify: false,
			},
			want: "https://qbit.domain.ltd:8080",
		},
		{
			name: "build_url_4",
			fields: fields{
				Host:          "qbit.domain.ltd:8080",
				Port:          0,
				Username:      "",
				Password:      "",
				TLS:           false,
				TLSSkipVerify: false,
			},
			want: "http://qbit.domain.ltd:8080",
		},
		{
			name: "build_url_5",
			fields: fields{
				Host:          "qbit.domain.ltd",
				Port:          8080,
				Username:      "",
				Password:      "",
				TLS:           false,
				TLSSkipVerify: false,
			},
			want: "http://qbit.domain.ltd:8080",
		},
		{
			name: "build_url_6",
			fields: fields{
				Host:          "qbit.domain.ltd",
				Port:          443,
				Username:      "",
				Password:      "",
				TLS:           true,
				TLSSkipVerify: false,
			},
			want: "https://qbit.domain.ltd",
		},
		{
			name: "build_url_7",
			fields: fields{
				Host:          "qbit.domain.ltd",
				Port:          10200,
				Username:      "",
				Password:      "",
				TLS:           false,
				TLSSkipVerify: false,
			},
			want: "http://qbit.domain.ltd:10200",
		},
		{
			name: "build_url_8",
			fields: fields{
				Host:          "https://domain.ltd/qbittorrent",
				Port:          0,
				Username:      "",
				Password:      "",
				TLS:           true,
				TLSSkipVerify: false,
			},
			want: "https://domain.ltd/qbittorrent",
		},
		{
			name: "build_url_9",
			fields: fields{
				Host:          "127.0.0.1",
				Port:          8080,
				Username:      "",
				Password:      "",
				TLS:           false,
				TLSSkipVerify: false,
			},
			want: "http://127.0.0.1:8080",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := DownloadClient{
				ID:            tt.fields.ID,
				Name:          tt.fields.Name,
				Type:          tt.fields.Type,
				Enabled:       tt.fields.Enabled,
				Host:          tt.fields.Host,
				Port:          tt.fields.Port,
				TLS:           tt.fields.TLS,
				TLSSkipVerify: tt.fields.TLSSkipVerify,
				Username:      tt.fields.Username,
				Password:      tt.fields.Password,
				Settings:      tt.fields.Settings,
			}
			assert.Equalf(t, tt.want, c.qbitBuildLegacyHost(), "qbitBuildLegacyHost()")
		})
	}
}
