// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/autobrr/autobrr/internal/domain"
)

type downloadClientService interface {
	List(ctx context.Context) ([]domain.DownloadClient, error)
	Store(ctx context.Context, client *domain.DownloadClient) error
	Update(ctx context.Context, client *domain.DownloadClient) error
	Delete(ctx context.Context, clientID int) error
	Test(ctx context.Context, client domain.DownloadClient) error
}

type downloadClientHandler struct {
	encoder encoder
	service downloadClientService
}

func newDownloadClientHandler(encoder encoder, service downloadClientService) *downloadClientHandler {
	return &downloadClientHandler{
		encoder: encoder,
		service: service,
	}
}

func (h downloadClientHandler) Routes(r chi.Router) {
	r.Get("/", h.listDownloadClients)
	r.Post("/", h.store)
	r.Put("/", h.update)
	r.Post("/test", h.test)
	r.Delete("/{clientID}", h.delete)
}

func (h downloadClientHandler) listDownloadClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clients, err := h.service.List(ctx)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, clients)
}

func (h downloadClientHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.DownloadClient

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	err := h.service.Store(r.Context(), data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h downloadClientHandler) test(w http.ResponseWriter, r *http.Request) {
	var data domain.DownloadClient

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Test(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h downloadClientHandler) update(w http.ResponseWriter, r *http.Request) {
	var data *domain.DownloadClient

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	err := h.service.Update(r.Context(), data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h downloadClientHandler) delete(w http.ResponseWriter, r *http.Request) {
	var clientID = chi.URLParam(r, "clientID")

	if clientID == "" {
		h.encoder.Error(w, errors.New("no clientID given"))
		return
	}

	id, err := strconv.Atoi(clientID)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err = h.service.Delete(r.Context(), id); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
