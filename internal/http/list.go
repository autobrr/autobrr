package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
)

type listService interface {
	Store(ctx context.Context, list *domain.List) error
	Update(ctx context.Context, list *domain.List) error
	Delete(ctx context.Context, id int64) error
	FindByID(ctx context.Context, id int64) (*domain.List, error)
	List(ctx context.Context) ([]*domain.List, error)
	RefreshList(ctx context.Context, id int64) error
	RefreshAll(ctx context.Context) error
}

type listHandler struct {
	encoder encoder
	listSvc listService
}

func newListHandler(encoder encoder, service listService) *listHandler {
	return &listHandler{encoder: encoder, listSvc: service}
}

func (h listHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.store)
	r.Post("/refresh", h.refreshAll)

	r.Route("/{listID}", func(r chi.Router) {
		r.Post("/refresh", h.refreshList)
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
}

func (h listHandler) list(w http.ResponseWriter, r *http.Request) {
	//data := []domain.List{
	//	{
	//		ID:      1,
	//		Name:    "test",
	//		Type:    "RADARR",
	//		Filters: []int{1},
	//	},
	//}
	data, err := h.listSvc.List(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
}

func (h listHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.List
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.listSvc.Store(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusCreatedData(w, data)
}

func (h listHandler) update(w http.ResponseWriter, r *http.Request) {
	var data *domain.List
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.listSvc.Update(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h listHandler) delete(w http.ResponseWriter, r *http.Request) {
	listID, err := strconv.Atoi(chi.URLParam(r, "listID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.listSvc.Delete(r.Context(), int64(listID)); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h listHandler) refreshAll(w http.ResponseWriter, r *http.Request) {
	if err := h.listSvc.RefreshAll(r.Context()); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h listHandler) refreshList(w http.ResponseWriter, r *http.Request) {
	listID, err := strconv.Atoi(chi.URLParam(r, "listID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.listSvc.RefreshList(r.Context(), int64(listID)); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
