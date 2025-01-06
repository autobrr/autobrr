// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
)

type notificationService interface {
	Find(context.Context, domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, id int) (*domain.Notification, error)
	Store(ctx context.Context, notification *domain.Notification) error
	Update(ctx context.Context, notification *domain.Notification) error
	Delete(ctx context.Context, id int) error
	Test(ctx context.Context, notification *domain.Notification) error
}

type notificationHandler struct {
	encoder encoder
	service notificationService
}

func newNotificationHandler(encoder encoder, service notificationService) *notificationHandler {
	return &notificationHandler{
		encoder: encoder,
		service: service,
	}
}

func (h notificationHandler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.store)
	r.Post("/test", h.test)

	r.Route("/{notificationID}", func(r chi.Router) {
		r.Get("/", h.findByID)
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
}

func (h notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	list, _, err := h.service.Find(r.Context(), domain.NotificationQueryParams{})
	if err != nil {
		h.encoder.StatusNotFound(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, list)
}

func (h notificationHandler) store(w http.ResponseWriter, r *http.Request) {
	var data *domain.Notification
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	err := h.service.Store(r.Context(), data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h notificationHandler) findByID(w http.ResponseWriter, r *http.Request) {
	notificationID, err := strconv.Atoi(chi.URLParam(r, "notificationID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	notification, err := h.service.FindByID(r.Context(), notificationID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("notification with id %d not found", notificationID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, notification)
}

func (h notificationHandler) update(w http.ResponseWriter, r *http.Request) {
	var data *domain.Notification
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	err := h.service.Update(r.Context(), data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
}

func (h notificationHandler) delete(w http.ResponseWriter, r *http.Request) {
	notificationID, err := strconv.Atoi(chi.URLParam(r, "notificationID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), notificationID); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h notificationHandler) test(w http.ResponseWriter, r *http.Request) {
	var data *domain.Notification
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Test(r.Context(), data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
