// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package config

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOIDCAutoRedirect(t *testing.T) {
	for _, tt := range []struct {
		name   string
		config string
		env    string
		want   bool
	}{
		{name: "disabled by default"},
		{name: "config file", config: "oidcAutoRedirect = true", want: true},
		{name: "environment variable", env: "true", want: true},
		{name: "case insensitive", env: "TRUE", want: true},
		{name: "environment overrides config", config: "oidcAutoRedirect = true", env: "false"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tt.config), 0o600))
			t.Setenv("AUTOBRR__OIDC_AUTO_REDIRECT", tt.env)
			t.Setenv("AUTOBRR__OIDC_AUTO_REDIRECT_FILE", "")

			cfg := New(dir)
			assert.Equal(t, tt.want, cfg.Config.OIDCAutoRedirect)
		})
	}
}

func TestOIDCAutoRedirectFromFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, "oidc-auto-redirect")
	require.NoError(t, os.WriteFile(envFile, []byte("true\n"), 0o600))
	t.Setenv("AUTOBRR__OIDC_AUTO_REDIRECT", "false")
	t.Setenv("AUTOBRR__OIDC_AUTO_REDIRECT_FILE", envFile)

	cfg := New(dir)
	assert.True(t, cfg.Config.OIDCAutoRedirect)
}

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
			want: []string{"# Check for updates", "#", "checkForUpdates = true", "# Log level", "#", "# Default: \"INFO\"", "#", "# Options: \"ERROR\", \"WARN\", \"INFO\", \"DEBUG\", \"TRACE\"", "#", `logLevel = "TRACE"`, "# Log Path", "#", "# Optional", "#", "#logPath = \"\""},
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
