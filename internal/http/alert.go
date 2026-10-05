// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
)

type alertService interface {
	List(ctx context.Context) []domain.Alert
}

type alertHandler struct {
	encoder encoder
	service alertService
}

func newAlertHandler(encoder encoder, service alertService) *alertHandler {
	return &alertHandler{encoder: encoder, service: service}
}

func (h alertHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
}

func (h alertHandler) list(w http.ResponseWriter, r *http.Request) {
	h.encoder.StatusResponse(w, http.StatusOK, h.service.List(r.Context()))
}
