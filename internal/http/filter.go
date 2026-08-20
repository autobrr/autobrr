// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
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
	UpdateNotifications(ctx context.Context, filterID int, notifications []domain.FilterNotification) error
	Duplicate(ctx context.Context, filterID int) (*domain.Filter, error)
	ToggleEnabled(ctx context.Context, filterID int, enabled bool) error
	PruneDeprecatedIndexers(ctx context.Context, identifiers []string) (int64, error)
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

	r.Post("/indexers/prune-deprecated", h.pruneDeprecatedIndexers)

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
	if sort != "" {
		field, order, found := strings.Cut(sort, "-")
		validField := field == "name" || field == "priority" || field == "created_at" || field == "updated_at"
		validOrder := order == "asc" || order == "desc"
		if !found || !validField || !validOrder {
			h.encoder.BadRequestErr(w, errors.New("sort parameter is invalid"))
			return
		}

		params.Sort[field] = order
	}

	params.Filters.Indexers = r.URL.Query()["indexer"]

	filters, err := h.service.Find(r.Context(), params)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, filters)
}

func (h filterHandler) getByID(w http.ResponseWriter, r *http.Request) {
	filterID, err := parseURLParamInt(r, "filterID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
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
	filterID, err := parseURLParamInt(r, "filterID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	filter, err := h.service.Duplicate(r.Context(), filterID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("filter with id %d not found", filterID))
			return
		}
		if errors.Is(err, domain.ErrIndexerNotFound) || errors.Is(err, domain.ErrIndexerArchived) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, filter)
}

func (h filterHandler) pruneDeprecatedIndexers(w http.ResponseWriter, r *http.Request) {
	var data struct {
		Identifiers []string `json:"identifiers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil && !errors.Is(err, io.EOF) {
		h.encoder.BadRequestErr(w, err)
		return
	}

	removed, err := h.service.PruneDeprecatedIndexers(r.Context(), data.Identifiers)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, map[string]int64{"removed": removed})
}

func (h filterHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.Filter
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Store(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrIndexerNotFound) || errors.Is(err, domain.ErrIndexerArchived) {
			h.encoder.BadRequestErr(w, err)
			return
		}
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
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("filter with id %d not found", data.ID))
			return
		}

		if errors.Is(err, domain.ErrIndexerNotFound) || errors.Is(err, domain.ErrIndexerArchived) || errors.Is(err, domain.ErrNotificationNotFound) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
}

func (h filterHandler) updatePartial(w http.ResponseWriter, r *http.Request) {
	var data domain.FilterUpdate
	filterID, err := parseURLParamInt(r, "filterID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}
	data.ID = filterID

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.UpdatePartial(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("filter with id %d not found", data.ID))
			return
		}

		if errors.Is(err, domain.ErrIndexerNotFound) || errors.Is(err, domain.ErrIndexerArchived) || errors.Is(err, domain.ErrNotificationNotFound) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h filterHandler) toggleEnabled(w http.ResponseWriter, r *http.Request) {
	filterID, err := parseURLParamInt(r, "filterID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
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
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("filter with id %d not found", filterID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h filterHandler) delete(w http.ResponseWriter, r *http.Request) {
	filterID, err := parseURLParamInt(r, "filterID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), filterID); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("filter with id %d not found", filterID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h filterHandler) getFilterNotifications(w http.ResponseWriter, r *http.Request) {
	filterID, err := parseURLParamInt(r, "filterID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
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

	// Return just the notifications array
	h.encoder.StatusResponse(w, http.StatusOK, filter.Notifications)
}

func (h filterHandler) updateFilterNotifications(w http.ResponseWriter, r *http.Request) {
	filterID, err := parseURLParamInt(r, "filterID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	var notifications []domain.FilterNotification
	if err := json.NewDecoder(r.Body).Decode(&notifications); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.UpdateNotifications(r.Context(), filterID, notifications); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("filter with id %d not found", filterID))
			return
		}

		if errors.Is(err, domain.ErrNotificationNotFound) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
