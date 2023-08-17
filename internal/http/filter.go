// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"
)

type filterService interface {
	ListFilters(ctx context.Context) ([]domain.Filter, error)
	FindByID(ctx context.Context, filterID int) (*domain.Filter, error)
	Find(ctx context.Context, params domain.FilterQueryParams) ([]domain.Filter, error)
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
	})
}

func (h filterHandler) getFilters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

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
		if s[0] == "name" || s[0] == "priority" {
			field = s[0]
		}

		if s[1] == "asc" || s[1] == "desc" {
			order = s[1]
		}

		params.Sort[field] = order
	}

	u, err := url.Parse(r.URL.String())
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusBadRequest, map[string]interface{}{
			"code":    "BAD_REQUEST_PARAMS",
			"message": "indexer parameter is invalid",
		})
		return
	}
	vals := u.Query()
	params.Filters.Indexers = vals["indexer"]

	trackers, err := h.service.Find(ctx, params)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, trackers)
}

func (h filterHandler) getByID(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "filterID")
	)

	id, err := strconv.Atoi(filterID)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.StatusNotFound(w)
			return
		}

		h.encoder.StatusInternalError(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, filter)
}

func (h filterHandler) duplicate(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "filterID")
	)

	id, err := strconv.Atoi(filterID)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.Duplicate(ctx, id)
	if err != nil {
		h.encoder.StatusInternalError(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, filter)
}

func (h filterHandler) store(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data *domain.Filter
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Store(ctx, data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusCreatedData(w, data)
}

func (h filterHandler) update(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data *domain.Filter
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Update(ctx, data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
}

func (h filterHandler) updatePartial(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		data     domain.FilterUpdate
		filterID = chi.URLParam(r, "filterID")
	)

	// set id from param and convert to int
	id, err := strconv.Atoi(filterID)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}
	data.ID = id

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.UpdatePartial(ctx, data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h filterHandler) toggleEnabled(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "filterID")
		data     struct {
			Enabled bool `json:"enabled"`
		}
	)

	id, err := strconv.Atoi(filterID)
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

func (h filterHandler) delete(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "filterID")
	)

	id, err := strconv.Atoi(filterID)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Delete(ctx, id); err != nil {
		h.encoder.Error(w, err)
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}
