// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
)

type proxyService interface {
	Store(ctx context.Context, p *domain.Proxy) error
	Update(ctx context.Context, p *domain.Proxy) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]domain.Proxy, error)
	FindByID(ctx context.Context, id int64) (*domain.Proxy, error)
	Test(ctx context.Context, p *domain.Proxy) error
}

type proxyHandler struct {
	encoder encoder
	service proxyService
}

func newProxyHandler(encoder encoder, service proxyService) *proxyHandler {
	return &proxyHandler{
		encoder: encoder,
		service: service,
	}
}

func (h proxyHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.store)
	r.Post("/test", h.test)

	r.Route("/{proxyID}", func(r chi.Router) {
		r.Get("/", h.findByID)
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
}

func (h proxyHandler) store(w http.ResponseWriter, r *http.Request) {
	var data domain.Proxy
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Store(r.Context(), &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h proxyHandler) update(w http.ResponseWriter, r *http.Request) {
	var data domain.Proxy
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Update(r.Context(), &data); err != nil {
		if errors.Is(err, domain.ErrUpdateFailed) {
			h.encoder.StatusError(w, http.StatusBadRequest, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h proxyHandler) list(w http.ResponseWriter, r *http.Request) {
	proxies, err := h.service.List(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, proxies)
}

func (h proxyHandler) findByID(w http.ResponseWriter, r *http.Request) {
	proxyID, err := strconv.Atoi(chi.URLParam(r, "proxyID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	proxies, err := h.service.FindByID(r.Context(), int64(proxyID))
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find proxy with id %d", proxyID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, proxies)
}

func (h proxyHandler) delete(w http.ResponseWriter, r *http.Request) {
	proxyID, err := strconv.Atoi(chi.URLParam(r, "proxyID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	err = h.service.Delete(r.Context(), int64(proxyID))
	if err != nil {
		if errors.Is(err, domain.ErrDeleteFailed) {
			h.encoder.StatusError(w, http.StatusBadRequest, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h proxyHandler) test(w http.ResponseWriter, r *http.Request) {
	var data domain.Proxy
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Test(r.Context(), &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
