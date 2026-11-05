// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/version"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type updateService interface {
	CheckUpdates(ctx context.Context)
	GetLatestRelease(ctx context.Context) *version.Release
}

type updateHandler struct {
	encoder encoder
	service updateService
}

func newUpdateHandler(encoder encoder, service updateService) *updateHandler {
	return &updateHandler{
		encoder: encoder,
		service: service,
	}
}

func (h updateHandler) Routes(r chi.Router) {
	read := requireScope(domain.APIResourceUpdates, domain.APIAccessRead)

	r.With(read).Get("/latest", h.getLatest)
	r.With(read).Get("/check", h.checkUpdates)
}

func (h updateHandler) getLatest(w http.ResponseWriter, r *http.Request) {
	latest := h.service.GetLatestRelease(r.Context())
	if latest != nil {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, latest)
		return
	}

	render.NoContent(w, r)
}

func (h updateHandler) checkUpdates(w http.ResponseWriter, r *http.Request) {
	h.service.CheckUpdates(r.Context())

	render.NoContent(w, r)
}
