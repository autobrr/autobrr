// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
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
	actions, err := h.service.List(r.Context())
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(w, http.StatusOK, actions)
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

	h.encoder.StatusResponse(w, http.StatusCreated, action)
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

	h.encoder.StatusResponse(w, http.StatusCreated, action)
}

func (h actionHandler) deleteAction(w http.ResponseWriter, r *http.Request) {
	actionID, err := parseInt(chi.URLParam(r, "id"))
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusBadRequest, errors.New("bad param id"))
	}

	if err := h.service.Delete(actionID); err != nil {
		// encode error
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h actionHandler) toggleActionEnabled(w http.ResponseWriter, r *http.Request) {
	actionID, err := parseInt(chi.URLParam(r, "id"))
	if err != nil {
		h.encoder.StatusResponse(w, http.StatusBadRequest, errors.New("bad param id"))
	}

	if err := h.service.ToggleEnabled(actionID); err != nil {
		// encode error
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
