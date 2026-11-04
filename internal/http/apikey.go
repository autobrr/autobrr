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
	Delete(ctx context.Context, id int) error
	ValidateAPIKey(ctx context.Context, token string) (*domain.APIKey, bool)
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

	r.Route("/{apiKeyID}", func(r chi.Router) {
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
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

	h.encoder.StatusResponse(w, http.StatusOK, keys)
}

func (h apikeyHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.APIKey
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if data == nil {
		h.encoder.BadRequestErr(w, errors.New("api key is required"))
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
	apiKeyID, err := parseURLParamInt(r, "apiKeyID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	var data *domain.APIKey
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if data == nil {
		h.encoder.BadRequestErr(w, errors.New("api key is required"))
		return
	}

	data.ID = apiKeyID
	data.Key = ""

	if err := h.service.Update(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("api key %d not found", apiKeyID))
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
	apiKeyID, err := parseURLParamInt(r, "apiKeyID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), apiKeyID); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("api key %d not found", apiKeyID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
