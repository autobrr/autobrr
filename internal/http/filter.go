package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/autobrr/autobrr/internal/domain"
)

type filterService interface {
	ListFilters(ctx context.Context) ([]domain.Filter, error)
	FindByID(ctx context.Context, filterID int) (*domain.Filter, error)
	Store(ctx context.Context, filter domain.Filter) (*domain.Filter, error)
	Delete(ctx context.Context, filterID int) error
	Update(ctx context.Context, filter domain.Filter) (*domain.Filter, error)
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
	r.Get("/{filterID}", h.getByID)
	r.Get("/{filterID}/duplicate", h.duplicate)
	r.Post("/", h.store)
	r.Put("/{filterID}", h.update)
	r.Patch("/{filterID}", h.updatePartial)
	r.Put("/{filterID}/enabled", h.toggleEnabled)
	r.Delete("/{filterID}", h.delete)
}

func (h filterHandler) getFilters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	trackers, err := h.service.ListFilters(ctx)
	if err != nil {
		//
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(ctx, w, trackers, http.StatusOK)
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
		h.encoder.StatusNotFound(ctx, w)
		return
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusOK)
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
		h.encoder.StatusNotFound(ctx, w)
		return
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusOK)
}

func (h filterHandler) store(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Filter
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.Store(ctx, data)
	if err != nil {
		// encode error
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusCreatedData(w, filter)
}

func (h filterHandler) update(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Filter
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.Update(ctx, data)
	if err != nil {
		// encode error
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusOK)
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
		// encode error
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.UpdatePartial(ctx, data); err != nil {
		// encode error
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
		// encode error
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.ToggleEnabled(ctx, id, data.Enabled); err != nil {
		// encode error
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
		// return err
		h.encoder.Error(w, err)
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}
