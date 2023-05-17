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

type feedService interface {
	Find(ctx context.Context) ([]domain.Feed, error)
	Store(ctx context.Context, feed *domain.Feed) error
	Update(ctx context.Context, feed *domain.Feed) error
	Delete(ctx context.Context, id int) error
	ToggleEnabled(ctx context.Context, id int, enabled bool) error
	Test(ctx context.Context, feed *domain.Feed) error
	GetLastRunData(ctx context.Context, id int) (string, error)
}

type feedHandler struct {
	encoder encoder
	service feedService
}

func newFeedHandler(encoder encoder, service feedService) *feedHandler {
	return &feedHandler{
		encoder: encoder,
		service: service,
	}
}

func (h feedHandler) Routes(r chi.Router) {
	r.Get("/", h.find)
	r.Get("/{feedID}/latest", h.latestRun)
	r.Post("/", h.store)
	r.Post("/test", h.test)
	r.Put("/{feedID}", h.update)
	r.Patch("/{feedID}/enabled", h.toggleEnabled)
	r.Delete("/{feedID}", h.delete)
}

func (h feedHandler) find(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	feeds, err := h.service.Find(ctx)
	if err != nil {
		h.encoder.StatusNotFound(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, feeds)
}

func (h feedHandler) store(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data *domain.Feed
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.StatusNotFound(w)
		return
	}

	err := h.service.Store(ctx, data)
	if err != nil {
		// encode error
		h.encoder.StatusInternalError(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h feedHandler) test(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data *domain.Feed
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.StatusInternalError(w)
		return
	}

	if err := h.service.Test(ctx, data); err != nil {
		// encode error
		h.encoder.StatusInternalError(w)
		return
	}

	h.encoder.NoContent(w)
}

func (h feedHandler) update(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data *domain.Feed
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.StatusInternalError(w)
		return
	}

	err := h.service.Update(ctx, data)
	if err != nil {
		// encode error
		h.encoder.StatusInternalError(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h feedHandler) toggleEnabled(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "feedID")
		data     struct {
			Enabled bool `json:"enabled"`
		}
	)

	id, _ := strconv.Atoi(filterID)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.StatusInternalError(w)
		return
	}

	err := h.service.ToggleEnabled(ctx, id, data.Enabled)
	if err != nil {
		// encode error
		h.encoder.StatusInternalError(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h feedHandler) delete(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "feedID")
	)

	id, _ := strconv.Atoi(filterID)

	if err := h.service.Delete(ctx, id); err != nil {
		h.encoder.StatusInternalError(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h feedHandler) latestRun(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "feedID")
	)

	id, err := strconv.Atoi(filterID)
	if err != nil {
		h.encoder.StatusInternalError(w)
		return
	}

	feed, err := h.service.GetLastRunData(ctx, id)
	if err != nil {
		h.encoder.StatusInternalError(w)
		return
	}

	if feed == "" {
		h.encoder.StatusNotFound(w)
		w.Write([]byte("No data found"))
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(feed))
}
