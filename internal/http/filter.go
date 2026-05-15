// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
)

type filterService interface {
	ListFilters(ctx context.Context) ([]domain.Filter, error)
	FindByID(ctx context.Context, filterID int) (*domain.Filter, error)
	Find(ctx context.Context, params domain.FilterQueryParams) ([]*domain.Filter, error)
	Store(ctx context.Context, filter *domain.Filter) error
	Delete(ctx context.Context, filterID int) error
	Update(ctx context.Context, filter *domain.Filter) error
	UpdatePartial(ctx context.Context, filter domain.FilterUpdate) error
	Duplicate(ctx context.Context, filterID int) (*domain.Filter, error)
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
}

type filterHandler struct {
	encoder encoder
	service filterService
}

func newFilterHandler(encoder encoder, service filterService) *filterHandler {
	return &filterHandler{
		encoder: encoder,
		service: service,
	}
}

func (h filterHandler) Routes(r chi.Router) {
	r.Get("/", h.getFilters)
	r.Post("/", h.store)

	r.Route("/{filterID}", func(r chi.Router) {
		r.Get("/", h.getByID)
		r.Put("/", h.update)
		r.Patch("/", h.updatePartial)
		r.Delete("/", h.delete)

		r.Get("/duplicate", h.duplicate)
		r.Put("/enabled", h.toggleEnabled)
		
		r.Route("/notifications", func(r chi.Router) {
			r.Get("/", h.getFilterNotifications)
			r.Put("/", h.updateFilterNotifications)
		})
	})
}

func (h filterHandler) getFilters(w http.ResponseWriter, r *http.Request) {
	params := domain.FilterQueryParams{
		Sort: map[string]string{},
		Filters: struct {
			Indexers []string
		}{},
		Search: "",
	}

	sort := r.URL.Query().Get("sort")
	if sort != "" && strings.Contains(sort, "-") {
		field := ""
		order := ""

		s := strings.Split(sort, "-")
		if s[0] == "name" || s[0] == "priority" || s[0] == "created_at" || s[0] == "updated_at" {
			field = s[0]
		}

		if s[1] == "asc" || s[1] == "desc" {
			order = s[1]
		}

		params.Sort[field] = order
	}

	u, err := url.Parse(r.URL.String())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]any{
			"code":    "BAD_REQUEST_PARAMS",
			"message": "indexer parameter is invalid",
		})
		return
	}
	vals := u.Query()
	params.Filters.Indexers = vals["indexer"]

	filters, err := h.service.Find(r.Context(), params)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, filters)
}

func (h filterHandler) getByID(w http.ResponseWriter, r *http.Request) {
	filterID, err := strconv.Atoi(chi.URLParam(r, "filterID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.FindByID(r.Context(), filterID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("filter with id %d not found", filterID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, filter)
}

func (h filterHandler) duplicate(w http.ResponseWriter, r *http.Request) {
	filterID, err := strconv.Atoi(chi.URLParam(r, "filterID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.Duplicate(r.Context(), filterID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("filter with id %d not found", filterID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, filter)
}

func (h filterHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.Filter
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Store(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusCreatedData(w, data)
}

func (h filterHandler) update(w http.ResponseWriter, r *http.Request) {
	var data *domain.Filter
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Update(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
}

func (h filterHandler) updatePartial(w http.ResponseWriter, r *http.Request) {
	var data domain.FilterUpdate
	filterID, err := strconv.Atoi(chi.URLParam(r, "filterID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}
	data.ID = filterID

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.UpdatePartial(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h filterHandler) toggleEnabled(w http.ResponseWriter, r *http.Request) {
	filterID, err := strconv.Atoi(chi.URLParam(r, "filterID"))
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

	if err := h.service.ToggleEnabled(r.Context(), filterID, data.Enabled); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h filterHandler) delete(w http.ResponseWriter, r *http.Request) {
	filterID, err := strconv.Atoi(chi.URLParam(r, "filterID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), filterID); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h filterHandler) getFilterNotifications(w http.ResponseWriter, r *http.Request) {
	filterID, err := strconv.Atoi(chi.URLParam(r, "filterID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.FindByID(r.Context(), filterID)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	// Return just the notifications array
	h.encoder.StatusResponse(w, http.StatusOK, filter.Notifications)
}

func (h filterHandler) updateFilterNotifications(w http.ResponseWriter, r *http.Request) {
	filterID, err := strconv.Atoi(chi.URLParam(r, "filterID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	var notifications []domain.FilterNotification
	if err := json.NewDecoder(r.Body).Decode(&notifications); err != nil {
		h.encoder.Error(w, err)
		return
	}

	// Use UpdatePartial to update just the notifications
	update := domain.FilterUpdate{
		ID:            filterID,
		Notifications: notifications,
	}

	if err := h.service.UpdatePartial(r.Context(), update); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
