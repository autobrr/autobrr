// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"net/http"

	"github.com/autobrr/autobrr/internal/database"

	"github.com/go-chi/chi/v5"
)

type healthHandler struct {
	encoder encoder
	db      *database.DB
}

func newHealthHandler(encoder encoder, db *database.DB) *healthHandler {
	return &healthHandler{
		encoder: encoder,
		db:      db,
	}
}

func (h healthHandler) Routes(r chi.Router) {
	r.Get("/liveness", h.handleLiveness)
	r.Get("/readiness", h.handleReadiness)
}

func (h healthHandler) handleLiveness(w http.ResponseWriter, _ *http.Request) {
	writeHealthy(w)
}

func (h healthHandler) handleReadiness(w http.ResponseWriter, _ *http.Request) {
	if err := h.db.Ping(); err != nil {
		writeUnhealthy(w)
		return
	}

	writeHealthy(w)
}

func writeHealthy(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func writeUnhealthy(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Unhealthy. Database unreachable"))
}
