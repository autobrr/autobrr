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

type downloaderService interface {
	List(ctx context.Context) ([]domain.Downloader, error)
	FindByID(ctx context.Context, id int32) (*domain.Downloader, error)
	Store(ctx context.Context, client *domain.Downloader) error
	Update(ctx context.Context, client *domain.Downloader) error
	Delete(ctx context.Context, clientID int32) error
	Test(ctx context.Context, client *domain.Downloader) error
	GetArrTags(ctx context.Context, id int32) ([]domain.ArrTag, error)
}

type downloaderHandler struct {
	encoder encoder
	service downloaderService
}

func newDownloaderHandler(encoder encoder, service downloaderService) *downloaderHandler {
	return &downloaderHandler{
		encoder: encoder,
		service: service,
	}
}

func (h downloaderHandler) Routes(r chi.Router) {
	r.Get("/", h.listDownloaders)
	r.Post("/", h.store)
	r.Put("/", h.update)
	r.Post("/test", h.test)

	r.Route("/{clientID}", func(r chi.Router) {
		r.Get("/", h.findByID)
		r.Delete("/", h.delete)

		r.Get("/arr/tags", h.findArrTagsByID)
	})
}

func (h downloaderHandler) listDownloaders(w http.ResponseWriter, r *http.Request) {
	clients, err := h.service.List(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, clients)
}

func (h downloaderHandler) findByID(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseURLParamInt32(r, "clientID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	client, err := h.service.FindByID(r.Context(), clientID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("download client with id %d not found", clientID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, client)
}

func (h downloaderHandler) findArrTagsByID(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseURLParamInt32(r, "clientID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	client, err := h.service.GetArrTags(r.Context(), clientID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("download client with id %d not found", clientID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, client)
}

func (h downloaderHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.Downloader
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

func (h downloaderHandler) test(w http.ResponseWriter, r *http.Request) {
	var data *domain.Downloader
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Test(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("download client with id %d not found", data.ID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h downloaderHandler) update(w http.ResponseWriter, r *http.Request) {
	var data *domain.Downloader
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	err := h.service.Update(r.Context(), data)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("download client with id %d not found", data.ID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
}

func (h downloaderHandler) delete(w http.ResponseWriter, r *http.Request) {
	clientID, err := parseURLParamInt32(r, "clientID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err = h.service.Delete(r.Context(), clientID); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("download client with id %d not found", clientID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
