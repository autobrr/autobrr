// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type apikeyService interface {
	List(ctx context.Context) ([]domain.APIKey, error)
	Store(ctx context.Context, key *domain.APIKey) error
	Update(ctx context.Context, key *domain.APIKey) error
	Delete(ctx context.Context, key string) error
	ValidateAPIKey(ctx context.Context, token string) bool
}

type apikeyHandler struct {
	encoder encoder
	service apikeyService
}

func newAPIKeyHandler(encoder encoder, service apikeyService) *apikeyHandler {
	return &apikeyHandler{
		encoder: encoder,
		service: service,
	}
}

func (h apikeyHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.store)
	r.Delete("/{apikey}", h.delete)
}

func (h apikeyHandler) list(w http.ResponseWriter, r *http.Request) {
	keys, err := h.service.List(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	render.JSON(w, r, keys)
}

func (h apikeyHandler) store(w http.ResponseWriter, r *http.Request) {

	var (
		ctx  = r.Context()
		data domain.APIKey
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Store(ctx, &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h apikeyHandler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), chi.URLParam(r, "apikey")); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
