// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi/v5"
)

type notificationService interface {
	Find(context.Context, domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, id int64) (*domain.Notification, error)
	Store(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Update(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Delete(ctx context.Context, id int64) error
	Test(ctx context.Context, notification domain.Notification) error
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
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
}

func (h notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	list, _, err := h.service.Find(ctx, domain.NotificationQueryParams{})
	if err != nil {
		h.encoder.StatusNotFound(w)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, list)
}

func (h notificationHandler) store(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Notification
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.Store(ctx, data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, filter)
}

func (h notificationHandler) update(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Notification
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	filter, err := h.service.Update(ctx, data)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, filter)
}

func (h notificationHandler) delete(w http.ResponseWriter, r *http.Request) {
	var (
		ctx            = r.Context()
		notificationID = chi.URLParam(r, "notificationID")
	)

	id, _ := strconv.Atoi(notificationID)

	if err := h.service.Delete(ctx, int64(id)); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusNoContent, nil)
}

func (h notificationHandler) test(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Notification
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Test(ctx, data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
