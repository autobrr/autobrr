// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
)

type indexerService interface {
	Store(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	Update(ctx context.Context, indexer domain.Indexer) (*domain.Indexer, error)
	List(ctx context.Context) ([]domain.Indexer, error)
	GetAll() ([]*domain.IndexerDefinition, error)
	GetTemplates() ([]domain.IndexerDefinition, error)
	Delete(ctx context.Context, id int) error
	TestApi(ctx context.Context, req domain.IndexerTestApiRequest) error
	ToggleEnabled(ctx context.Context, indexerID int, enabled bool) error
}

type indexerHandler struct {
	encoder encoder
	service indexerService
	ircSvc  ircService
}

func newIndexerHandler(encoder encoder, service indexerService, ircSvc ircService) *indexerHandler {
	return &indexerHandler{
		encoder: encoder,
		service: service,
		ircSvc:  ircSvc,
	}
}

func (h indexerHandler) Routes(r chi.Router) {
	r.Get("/schema", h.getSchema)
	r.Post("/", h.store)
	r.Get("/", h.getAll)
	r.Get("/options", h.list)

	r.Route("/{indexerID}", func(r chi.Router) {
		r.Put("/", h.update)
		r.Delete("/", h.delete)
		r.Post("/api/test", h.testApi)

		r.Patch("/enabled", h.toggleEnabled)
	})
}

func (h indexerHandler) getSchema(w http.ResponseWriter, r *http.Request) {
	indexers, err := h.service.GetTemplates()
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, indexers)
}

func (h indexerHandler) store(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Indexer
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	indexer, err := h.service.Store(ctx, data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, indexer)
}

func (h indexerHandler) update(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Indexer
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	indexer, err := h.service.Update(ctx, data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, indexer)
}

func (h indexerHandler) delete(w http.ResponseWriter, r *http.Request) {
	var (
		ctx     = r.Context()
		idParam = chi.URLParam(r, "indexerID")
	)

	id, err := strconv.Atoi(idParam)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Delete(ctx, id); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h indexerHandler) getAll(w http.ResponseWriter, r *http.Request) {
	indexers, err := h.service.GetAll()
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, indexers)
}

func (h indexerHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	indexers, err := h.service.List(ctx)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, indexers)
}

func (h indexerHandler) testApi(w http.ResponseWriter, r *http.Request) {
	var (
		ctx     = r.Context()
		idParam = chi.URLParam(r, "indexerID")
		req     domain.IndexerTestApiRequest
	)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.encoder.Error(w, err)
		return
	}

	id, err := strconv.Atoi(idParam)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if req.IndexerId == 0 {
		req.IndexerId = id
	}

	if err := h.service.TestApi(ctx, req); err != nil {
		h.encoder.Error(w, err)
		return
	}

	res := struct {
		Message string `json:"message"`
	}{
		Message: "Indexer api test OK",
	}

	h.encoder.StatusResponse(w, http.StatusOK, res)
}

func (h indexerHandler) toggleEnabled(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		indexerID = chi.URLParam(r, "indexerID")
		data      struct {
			Enabled bool `json:"enabled"`
		}
	)

	id, err := strconv.Atoi(indexerID)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.ToggleEnabled(ctx, id, data.Enabled); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
