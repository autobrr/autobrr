package http

import (
	"encoding/json"
	"net/http"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
)

type listService interface{}

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

	r.Route("/{listID}", func(r chi.Router) {
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
}

func (h listHandler) list(w http.ResponseWriter, r *http.Request) {
	data := []domain.List{
		{
			ID:      1,
			Name:    "test",
			Type:    "RADARR",
			Filters: []int{1},
		},
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
}

func (h listHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.Filter
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	//if err := h.service.Store(r.Context(), data); err != nil {
	//	h.encoder.Error(w, err)
	//	return
	//}

	h.encoder.StatusCreatedData(w, data)
}

func (h listHandler) update(w http.ResponseWriter, r *http.Request) {}

func (h listHandler) delete(w http.ResponseWriter, r *http.Request) {

}
