package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"

	"github.com/autobrr/autobrr/internal/domain"
)

type filterService interface {
	ListFilters() ([]domain.Filter, error)
	FindByID(filterID int) (*domain.Filter, error)
	Store(filter domain.Filter) (*domain.Filter, error)
	Delete(filterID int) error
	Update(filter domain.Filter) (*domain.Filter, error)
	//StoreFilterAction(action domain.Action) error
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
	r.Post("/", h.store)
	r.Put("/{filterID}", h.update)
	r.Delete("/{filterID}", h.delete)
}

func (h filterHandler) getFilters(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	trackers, err := h.service.ListFilters()
	if err != nil {
		//
	}

	h.encoder.StatusResponse(ctx, w, trackers, http.StatusOK)
}

func (h filterHandler) getByID(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "filterID")
	)

	id, _ := strconv.Atoi(filterID)

	filter, err := h.service.FindByID(id)
	if err != nil {
		h.encoder.StatusNotFound(ctx, w)
		return
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusOK)
}

func (h filterHandler) storeFilterAction(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "filterID")
	)

	id, _ := strconv.Atoi(filterID)

	filter, err := h.service.FindByID(id)
	if err != nil {
		//
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusCreated)
}

func (h filterHandler) store(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Filter
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	filter, err := h.service.Store(data)
	if err != nil {
		// encode error
		return
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusCreated)
}

func (h filterHandler) update(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Filter
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	filter, err := h.service.Update(data)
	if err != nil {
		// encode error
		return
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusOK)
}

func (h filterHandler) delete(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		filterID = chi.URLParam(r, "filterID")
	)

	id, _ := strconv.Atoi(filterID)

	if err := h.service.Delete(id); err != nil {
		// return err
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}
