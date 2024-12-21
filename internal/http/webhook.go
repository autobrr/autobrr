package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

type webhookHandler struct {
	encoder encoder
	listSvc listService
}

func newWebhookHandler(encoder encoder, listSvc listService) *webhookHandler {
	return &webhookHandler{}
}

func (h *webhookHandler) Routes(r chi.Router) {
	r.Route("/lists", func(r chi.Router) {
		r.Post("/trigger", h.refreshAll)
		r.Post("/trigger/arr", h.refreshArr)
		r.Post("/trigger/lists", h.refreshLists)

		r.Get("/trigger", h.refreshAll)
		r.Get("/trigger/arr", h.refreshArr)
		r.Get("/trigger/lists", h.refreshLists)

		r.Post("/trigger/{listID}", h.refreshByID)
	})
}

func (h *webhookHandler) refreshAll(w http.ResponseWriter, r *http.Request) {
	go h.listSvc.RefreshAll(context.Background())

	h.encoder.NoContent(w)
}

func (h *webhookHandler) refreshByID(w http.ResponseWriter, r *http.Request) {
	listID, err := strconv.Atoi(chi.URLParam(r, "listID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.listSvc.RefreshList(context.Background(), int64(listID)); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h *webhookHandler) refreshArr(w http.ResponseWriter, r *http.Request) {
	if err := h.listSvc.RefreshArrLists(r.Context()); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h *webhookHandler) refreshLists(w http.ResponseWriter, r *http.Request) {
	if err := h.listSvc.RefreshOtherLists(r.Context()); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
