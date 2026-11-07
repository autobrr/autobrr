// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/dustin/go-humanize"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type logsHandler struct {
	cfg *config.AppConfig
}

func newLogsHandler(cfg *config.AppConfig) *logsHandler {
	return &logsHandler{cfg: cfg}
}

func (h logsHandler) Routes(r chi.Router) {
	read := requireScope(domain.APIResourceLogs, domain.APIAccessRead)

	r.With(read).Get("/files", h.files)
	r.With(read).Get("/files/{logFile}", h.downloadFile)
}

func (h logsHandler) files(w http.ResponseWriter, r *http.Request) {
	response := LogfilesResponse{
		Files: []LogFile{},
		Count: 0,
	}

	if h.cfg.Config.LogPath == "" {
		render.JSON(w, r, response)
		return
	}

	logsDir := filepath.Dir(h.cfg.Config.LogPath)

	// check if dir exists before walkDir
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		render.JSON(w, r, response)
		return
	}

	var walk = func(path string, d fs.DirEntry, err error) error {
		// downloads read from logsDir itself, so nested files could not be fetched
		if d.IsDir() {
			if path != logsDir {
				return fs.SkipDir
			}
			return nil
		}

		if filepath.Ext(path) == ".log" {
			i, err := d.Info()
			if err != nil {
				return err
			}

			response.Files = append(response.Files, LogFile{
				Name:      d.Name(),
				SizeBytes: i.Size(),
				Size:      humanize.Bytes(uint64(i.Size())),
				UpdatedAt: i.ModTime(),
			})
		}

		return nil
	}

	if err := filepath.WalkDir(logsDir, walk); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}

	response.Count = len(response.Files)

	render.JSON(w, r, response)
}

func (h logsHandler) downloadFile(w http.ResponseWriter, r *http.Request) {
	if h.cfg.Config.LogPath == "" {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{
			Message: "log file path is not configured",
			Status:  http.StatusNotFound,
		})
		return
	}

	logsDir := filepath.Dir(h.cfg.Config.LogPath)

	// check if dir exists before walkDir
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		render.Status(r, http.StatusNotFound)
		render.JSON(w, r, errorResponse{
			Message: "log directory not found or inaccessible",
			Status:  http.StatusNotFound,
		})
		return
	}

	logFile := chi.URLParam(r, "logFile")

	// a bare name only: Windows decodes %5C to a separator that filepath.Join
	// follows, which would reach files outside the log directory
	if strings.ContainsAny(logFile, `/\`) || logFile != filepath.Base(logFile) || filepath.Ext(logFile) != ".log" {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, errorResponse{
			Message: "invalid log file. Must be a .log file in the log directory",
			Status:  http.StatusBadRequest,
		})
		return
	}

	filePath := filepath.Join(logsDir, logFile)

	w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(logFile))
	w.Header().Set("Content-Type", "application/octet-stream")

	// Sanitize the log file and directly write the output to the HTTP socket
	if err := logger.SanitizeLogFile(filePath, w); err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, errorResponse{
			Message: err.Error(),
			Status:  http.StatusInternalServerError,
		})
		return
	}
}

type LogFile struct {
	Name      string    `json:"filename"`
	SizeBytes int64     `json:"size_bytes"`
	Size      string    `json:"size"`
	UpdatedAt time.Time `json:"updated_at"`
}

type LogfilesResponse struct {
	Files []LogFile `json:"files"`
	Count int       `json:"count"`
}
