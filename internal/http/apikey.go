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
)

type apikeyService interface {
	List(ctx context.Context) ([]domain.APIKey, error)
	Store(ctx context.Context, key *domain.APIKey) error
	Update(ctx context.Context, key *domain.APIKey) error
	Delete(ctx context.Context, key string) error
	ValidateAPIKey(ctx context.Context, token string) (*domain.APIKey, bool)
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

	r.Route("/{apikey}", func(r chi.Router) {
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
}

func (h apikeyHandler) list(w http.ResponseWriter, r *http.Request) {
	keys, err := h.service.List(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, keys)
}

func (h apikeyHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.APIKey
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Store(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrInvalidAPIKeyScopes) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusCreatedData(w, data)
}

func (h apikeyHandler) update(w http.ResponseWriter, r *http.Request) {
	apiKey := chi.URLParam(r, "apikey")

	var data *domain.APIKey
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	data.Key = apiKey

	if err := h.service.Update(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("api key %s not found", apiKey))
			return
		}
		if errors.Is(err, domain.ErrInvalidAPIKeyScopes) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
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
