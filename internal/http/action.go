package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/go-chi/chi"
)

type actionService interface {
	Fetch() ([]domain.Action, error)
	Store(ctx context.Context, action domain.Action) (*domain.Action, error)
	Delete(actionID int) error
	ToggleEnabled(actionID int) error
}

type actionHandler struct {
	encoder encoder
	service actionService
}

func newActionHandler(encoder encoder, service actionService) *actionHandler {
	return &actionHandler{
		encoder: encoder,
		service: service,
	}
}

func (h actionHandler) Routes(r chi.Router) {
	r.Get("/", h.getActions)
	r.Post("/", h.storeAction)
	r.Delete("/{id}", h.deleteAction)
	r.Put("/{id}", h.updateAction)
	r.Patch("/{id}/toggleEnabled", h.toggleActionEnabled)
}

func (h actionHandler) getActions(w http.ResponseWriter, r *http.Request) {
	actions, err := h.service.Fetch()
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(r.Context(), w, actions, http.StatusOK)
}

func (h actionHandler) storeAction(w http.ResponseWriter, r *http.Request) {
	var (
		data domain.Action
		ctx  = r.Context()
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	action, err := h.service.Store(ctx, data)
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, action, http.StatusCreated)
}

func (h actionHandler) updateAction(w http.ResponseWriter, r *http.Request) {
	var (
		data domain.Action
		ctx  = r.Context()
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	action, err := h.service.Store(ctx, data)
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, action, http.StatusCreated)
}

func (h actionHandler) deleteAction(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()

	actionID, err := parseInt(chi.URLParam(r, "id"))
	if err != nil {
		h.encoder.StatusResponse(ctx, w, errors.New("bad param id"), http.StatusBadRequest)
	}

	if err := h.service.Delete(actionID); err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func (h actionHandler) toggleActionEnabled(w http.ResponseWriter, r *http.Request) {
	var ctx = r.Context()

	actionID, err := parseInt(chi.URLParam(r, "id"))
	if err != nil {
		h.encoder.StatusResponse(ctx, w, errors.New("bad param id"), http.StatusBadRequest)
	}

	if err := h.service.ToggleEnabled(actionID); err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusCreated)
}

func parseInt(s string) (int, error) {
	u, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(u), nil
}
