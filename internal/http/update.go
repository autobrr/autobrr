// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
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
	config  *domain.Config
}

func newUpdateHandler(encoder encoder, service updateService, config *domain.Config) *updateHandler {
	return &updateHandler{
		encoder: encoder,
		service: service,
		config:  config,
	}
}

func (h updateHandler) Routes(r chi.Router) {
	r.Get("/latest", h.getLatest)
	r.Get("/check", h.checkUpdates)
}

func (h updateHandler) getLatest(w http.ResponseWriter, r *http.Request) {
	if !h.config.CheckForUpdates {
		render.NoContent(w, r)
		return
	}

	latest := h.service.GetLatestRelease(r.Context())
	if latest == nil {
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, latest)
}

func (h updateHandler) checkUpdates(w http.ResponseWriter, r *http.Request) {
	h.service.CheckUpdates(r.Context())

	render.NoContent(w, r)
}
