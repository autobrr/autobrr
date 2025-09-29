// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package config

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/autobrr/autobrr/internal/domain"
)

func TestAppConfig_processLines(t *testing.T) {
	t.Parallel()
	type fields struct {
		Config *domain.Config
		m      *sync.Mutex
	}
	type args struct {
		lines []string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
	}{
		{
			name: "append missing",
			fields: fields{
				Config: &domain.Config{CheckForUpdates: true, LogLevel: "TRACE"},
				m:      new(sync.Mutex),
			},
			args: args{[]string{}},
			want: []string{"# Check for updates", "#", "checkForUpdates = true", "# Log level", "#", "# Default: \"DEBUG\"", "#", "# Options: \"ERROR\", \"DEBUG\", \"INFO\", \"WARN\", \"TRACE\"", "#", `logLevel = "TRACE"`, "# Log Path", "#", "# Optional", "#", "#logPath = \"\""},
		},
		{
			name: "update existing",
			fields: fields{
				Config: &domain.Config{CheckForUpdates: true, LogLevel: "TRACE"},
				m:      new(sync.Mutex),
			},
			args: args{[]string{"# Check for updates", "#", "checkForUpdates = false", "# Log level", "#", "# Default: \"DEBUG\"", "#", "# Options: \"ERROR\", \"DEBUG\", \"INFO\", \"WARN\", \"TRACE\"", "#", `logLevel = "TRACE"`, "# Log Path", "#", "# Optional", "#", "#logPath = \"\""}},
			want: []string{"# Check for updates", "#", "checkForUpdates = true", "# Log level", "#", "# Default: \"DEBUG\"", "#", "# Options: \"ERROR\", \"DEBUG\", \"INFO\", \"WARN\", \"TRACE\"", "#", `logLevel = "TRACE"`, "# Log Path", "#", "# Optional", "#", "#logPath = \"\""},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &AppConfig{
				Config: tt.fields.Config,
				m:      tt.fields.m,
			}

			assert.Equalf(t, tt.want, c.processLines(tt.args.lines), tt.name)
		})
	}
}
