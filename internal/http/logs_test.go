// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsHandler_DownloadFileRejectsPaths(t *testing.T) {
	dir := t.TempDir()
	logsDir := filepath.Join(dir, "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "autobrr.log"), []byte("line\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secret.log"), []byte("secret\n"), 0o600))

	h := newLogsHandler(&config.AppConfig{Config: &domain.Config{LogPath: filepath.Join(logsDir, "autobrr.log")}})

	tests := []struct {
		name string
		file string
		want int
	}{
		{name: "log file", file: "autobrr.log", want: http.StatusOK},
		{name: "parent with slash", file: "../secret.log", want: http.StatusBadRequest},
		{name: "parent with backslash", file: `..\secret.log`, want: http.StatusBadRequest},
		{name: "log in a cleaned away component", file: `x.log\..\..\secret.log`, want: http.StatusBadRequest},
		{name: "not a log", file: "config.toml", want: http.StatusBadRequest},
		{name: "log in the middle", file: "autobrr.log.txt", want: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("logFile", tt.file)

			req := httptest.NewRequest(http.MethodGet, "/api/logs/files/x", nil)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
			rec := httptest.NewRecorder()

			h.downloadFile(rec, req)

			assert.Equal(t, tt.want, rec.Code)
			if tt.want == http.StatusOK {
				assert.Contains(t, rec.Body.String(), "line")
			}
			assert.NotContains(t, rec.Body.String(), "secret")
		})
	}
}

func TestLogsHandler_DownloadFileWithoutLogPath(t *testing.T) {
	h := newLogsHandler(&config.AppConfig{Config: &domain.Config{}})

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("logFile", "autobrr.log")

	req := httptest.NewRequest(http.MethodGet, "/api/logs/files/autobrr.log", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	h.downloadFile(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "log file path is not configured")
}

func TestLogsHandler_FilesListsTopLevelOnly(t *testing.T) {
	logsDir := filepath.Join(t.TempDir(), "logs")
	require.NoError(t, os.MkdirAll(filepath.Join(logsDir, "old"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "autobrr.log"), []byte("line\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(logsDir, "old", "nested.log"), []byte("line\n"), 0o600))

	h := newLogsHandler(&config.AppConfig{Config: &domain.Config{LogPath: filepath.Join(logsDir, "autobrr.log")}})

	rec := httptest.NewRecorder()
	h.files(rec, httptest.NewRequest(http.MethodGet, "/api/logs/files", nil))

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"autobrr.log"`)
	assert.NotContains(t, rec.Body.String(), "nested.log")
}
