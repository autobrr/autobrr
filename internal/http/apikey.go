// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type apikeyService interface {
	List(ctx context.Context) ([]domain.APIKey, error)
	Store(ctx context.Context, key *domain.APIKey) error
	Delete(ctx context.Context, key string) error
	ValidateAPIKey(ctx context.Context, token string) bool
}

type apikeyHandler struct {
	encoder encoder
	service apikeyService
	config  *domain.Config
}

func newAPIKeyHandler(encoder encoder, service apikeyService, config *domain.Config) *apikeyHandler {
	return &apikeyHandler{
		encoder: encoder,
		service: service,
		config:  config,
	}
}

func (h apikeyHandler) Routes(r chi.Router) {
	r.Use(h.rejectIfAuthDisabled)
	r.Get("/", h.list)
	r.Post("/", h.store)
	r.Delete("/{apikey}", h.delete)
}

// rejectIfAuthDisabled blocks API key management while built-in authentication is
// disabled. Otherwise an authless caller could mint a persistent credential that
// stays valid after authentication is re-enabled.
func (h apikeyHandler) rejectIfAuthDisabled(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.config.IsAuthDisabled() {
			h.encoder.StatusError(w, http.StatusForbidden, errors.New("API key management is unavailable while authentication is disabled"))
			return
		}
		next.ServeHTTP(w, r)
	})
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
	var data domain.APIKey
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Store(r.Context(), &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h apikeyHandler) delete(w http.ResponseWriter, r *http.Request) {
	apiKey := chi.URLParam(r, "apikey")

	if err := h.service.Delete(r.Context(), apiKey); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("api key %s not found", apiKey))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
