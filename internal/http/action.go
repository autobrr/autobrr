// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
)

type actionService interface {
	List(ctx context.Context) ([]domain.Action, error)
	Store(ctx context.Context, action domain.Action) (*domain.Action, error)
	Delete(ctx context.Context, req *domain.DeleteActionRequest) error
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

	r.Route("/{actionID}", func(r chi.Router) {
		r.Delete("/", h.deleteAction)
		r.Put("/", h.updateAction)
		r.Patch("/toggleEnabled", h.toggleActionEnabled)
	})
}

func (h actionHandler) getActions(w http.ResponseWriter, r *http.Request) {
	actions, err := h.service.List(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, actions)
}

func (h actionHandler) storeAction(w http.ResponseWriter, r *http.Request) {
	var data domain.Action
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	action, err := h.service.Store(r.Context(), data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, action)
}

func (h actionHandler) updateAction(w http.ResponseWriter, r *http.Request) {
	var data domain.Action
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	action, err := h.service.Store(r.Context(), data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, action)
}

func (h actionHandler) deleteAction(w http.ResponseWriter, r *http.Request) {
	actionID, err := parseInt(chi.URLParam(r, "actionID"))
	if err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.New("bad param id"))
		return
	}

	if err := h.service.Delete(r.Context(), &domain.DeleteActionRequest{ActionId: actionID}); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h actionHandler) toggleActionEnabled(w http.ResponseWriter, r *http.Request) {
	actionID, err := parseInt(chi.URLParam(r, "actionID"))
	if err != nil {
		h.encoder.StatusError(w, http.StatusBadRequest, errors.New("bad param id"))
		return
	}

	if err := h.service.ToggleEnabled(actionID); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, nil)
}

func parseInt(s string) (int, error) {
	u, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, err
	}
	return int(u), nil
}
