package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/go-chi/chi"
)

type actionService interface {
	Fetch() ([]domain.Action, error)
	Store(action domain.Action) (*domain.Action, error)
	Delete(actionID int) error
	ToggleEnabled(actionID int) error
}

type actionHandler struct {
	encoder       encoder
	actionService actionService
}

func (h actionHandler) Routes(r chi.Router) {
	r.Get("/", h.getActions)
	r.Post("/", h.storeAction)
	r.Delete("/{actionID}", h.deleteAction)
	r.Put("/{actionID}", h.updateAction)
	r.Patch("/{actionID}/toggleEnabled", h.toggleActionEnabled)
}

func (h actionHandler) getActions(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	actions, err := h.actionService.Fetch()
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, actions, http.StatusOK)
}

func (h actionHandler) storeAction(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Action
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	action, err := h.actionService.Store(data)
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, action, http.StatusCreated)
}

func (h actionHandler) updateAction(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Action
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	action, err := h.actionService.Store(data)
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, action, http.StatusCreated)
}

func (h actionHandler) deleteAction(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		actionID = chi.URLParam(r, "actionID")
	)

	// if !actionID return error

	id, _ := strconv.Atoi(actionID)

	if err := h.actionService.Delete(id); err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func (h actionHandler) toggleActionEnabled(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		actionID = chi.URLParam(r, "actionID")
	)

	// if !actionID return error

	id, _ := strconv.Atoi(actionID)

	if err := h.actionService.ToggleEnabled(id); err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusCreated)
}
