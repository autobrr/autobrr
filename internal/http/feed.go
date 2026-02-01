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

type feedService interface {
	Find(ctx context.Context) ([]domain.Feed, error)
	FindByID(ctx context.Context, id int) (*domain.Feed, error)
	Store(ctx context.Context, feed *domain.Feed) error
	Update(ctx context.Context, feed *domain.Feed) error
	Delete(ctx context.Context, id int) error
	DeleteFeedCache(ctx context.Context, id int) error
	ToggleEnabled(ctx context.Context, id int, enabled bool) error
	Test(ctx context.Context, feed *domain.Feed) error
	GetLastRunData(ctx context.Context, id int) (string, error)
	ForceRun(ctx context.Context, id int) error
	FetchCaps(ctx context.Context, feed *domain.Feed) (*domain.FeedCapabilities, error)
	FetchCapsByID(ctx context.Context, id int) (*domain.FeedCapabilities, error)
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
	r.Post("/", h.store)
	r.Post("/test", h.test)
	r.Post("/caps", h.caps)

	r.Route("/{feedID}", func(r chi.Router) {
		r.Get("/", h.findByID)
		r.Put("/", h.update)
		r.Delete("/", h.delete)
		r.Delete("/cache", h.deleteCache)
		r.Patch("/enabled", h.toggleEnabled)
		r.Get("/latest", h.latestRun)
		r.Post("/forcerun", h.forceRun)
		r.Get("/caps", h.capsByID)
	})
}

func (h feedHandler) find(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.service.Find(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, feeds)
}

func (h feedHandler) findByID(w http.ResponseWriter, r *http.Request) {
	feedID, err := strconv.Atoi(chi.URLParam(r, "feedID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	feed, err := h.service.FindByID(r.Context(), feedID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find feed with id %d", feedID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, feed)
}

func (h feedHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.Feed
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

func (h feedHandler) test(w http.ResponseWriter, r *http.Request) {
	var data *domain.Feed
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

func (h feedHandler) caps(w http.ResponseWriter, r *http.Request) {
	var data *domain.Feed
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	caps, err := h.service.FetchCaps(r.Context(), data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, caps)
}

func (h feedHandler) update(w http.ResponseWriter, r *http.Request) {
	var data *domain.Feed
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

func (h feedHandler) forceRun(w http.ResponseWriter, r *http.Request) {
	feedID, err := strconv.Atoi(chi.URLParam(r, "feedID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.ForceRun(r.Context(), feedID); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find feed with id %d", feedID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h feedHandler) capsByID(w http.ResponseWriter, r *http.Request) {
	feedID, err := strconv.Atoi(chi.URLParam(r, "feedID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	caps, err := h.service.FetchCapsByID(r.Context(), feedID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find feed with id %d", feedID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, caps)
}

func (h feedHandler) toggleEnabled(w http.ResponseWriter, r *http.Request) {
	feedID, err := strconv.Atoi(chi.URLParam(r, "feedID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	var data struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.ToggleEnabled(r.Context(), feedID, data.Enabled); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find feed with id %d", feedID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h feedHandler) delete(w http.ResponseWriter, r *http.Request) {
	feedID, err := strconv.Atoi(chi.URLParam(r, "feedID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), feedID); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find feed with id %d", feedID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h feedHandler) deleteCache(w http.ResponseWriter, r *http.Request) {
	feedID, err := strconv.Atoi(chi.URLParam(r, "feedID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.DeleteFeedCache(r.Context(), feedID); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h feedHandler) latestRun(w http.ResponseWriter, r *http.Request) {
	feedID, err := strconv.Atoi(chi.URLParam(r, "feedID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	feed, err := h.service.GetLastRunData(r.Context(), feedID)
	if err != nil {
		h.encoder.Error(w, err)
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
