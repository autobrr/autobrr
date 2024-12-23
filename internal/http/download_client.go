// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
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

type downloadClientService interface {
	List(ctx context.Context) ([]domain.DownloadClient, error)
	FindByID(ctx context.Context, id int32) (*domain.DownloadClient, error)
	Store(ctx context.Context, client *domain.DownloadClient) error
	Update(ctx context.Context, client *domain.DownloadClient) error
	Delete(ctx context.Context, clientID int32) error
	Test(ctx context.Context, client domain.DownloadClient) error
	GetArrTags(ctx context.Context, id int32) ([]*domain.ArrTag, error)
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

	r.Route("/{clientID}", func(r chi.Router) {
		r.Get("/", h.findByID)
		r.Delete("/", h.delete)

		r.Get("/arr/tags", h.findArrTagsByID)
	})
}

func (h downloadClientHandler) listDownloadClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.service.List(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, clients)
}

func (h downloadClientHandler) findByID(w http.ResponseWriter, r *http.Request) {
	clientID, err := strconv.ParseInt(chi.URLParam(r, "clientID"), 10, 32)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	client, err := h.service.FindByID(r.Context(), int32(clientID))
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("download client with id %d not found", clientID))
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, client)
}

func (h downloadClientHandler) findArrTagsByID(w http.ResponseWriter, r *http.Request) {
	clientID, err := strconv.ParseInt(chi.URLParam(r, "clientID"), 10, 32)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	client, err := h.service.GetArrTags(r.Context(), int32(clientID))
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("download client with id %d not found", clientID))
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, client)
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
	clientID, err := strconv.ParseInt(chi.URLParam(r, "clientID"), 10, 32)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err = h.service.Delete(r.Context(), int32(clientID)); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
