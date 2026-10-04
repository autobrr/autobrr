// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/notification/services/pushover"
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

	FindInbox(ctx context.Context, params domain.InboxQueryParams) (*domain.FindInboxResponse, error)
	MarkInboxRead(ctx context.Context, messageIDs []int64) error
	DeleteInboxMessages(ctx context.Context, messageIDs []int64) error
	DeleteInbox(ctx context.Context) error
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
	r.Get("/pushover/sounds", h.pushoverSounds)

	r.Route("/inbox", func(r chi.Router) {
		r.Get("/", h.findInbox)
		r.Delete("/", h.deleteInbox)
		r.Post("/read", h.markInboxRead)
		r.Post("/delete", h.deleteInboxMessages)
	})

	r.Route("/{notificationID}", func(r chi.Router) {
		r.Get("/", h.findByID)
		r.Put("/", h.update)
		r.Delete("/", h.delete)
	})
}

func (h notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	list, _, err := h.service.Find(r.Context(), domain.NotificationQueryParams{})
	if err != nil {
		h.encoder.Error(w, err)
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
		if errors.Is(err, domain.ErrNotificationBuiltin) || errors.Is(err, domain.ErrNotificationInvalid) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h notificationHandler) findByID(w http.ResponseWriter, r *http.Request) {
	notificationID, err := parseURLParamInt(r, "notificationID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	notif, err := h.service.FindByID(r.Context(), notificationID)
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("notification with id %d not found", notificationID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, notif)
}

func (h notificationHandler) update(w http.ResponseWriter, r *http.Request) {
	var data *domain.Notification
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Update(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("notification with id %d not found", data.ID))
			return
		}
		if errors.Is(err, domain.ErrNotificationBuiltin) || errors.Is(err, domain.ErrNotificationInvalid) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, data)
}

func (h notificationHandler) delete(w http.ResponseWriter, r *http.Request) {
	notificationID, err := parseURLParamInt(r, "notificationID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), notificationID); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("notification with id %d not found", notificationID))
			return
		}
		if errors.Is(err, domain.ErrNotificationBuiltin) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h notificationHandler) test(w http.ResponseWriter, r *http.Request) {
	var data *domain.Notification
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.Test(r.Context(), data); err != nil {
		if errors.Is(err, domain.ErrNotificationInvalid) {
			h.encoder.BadRequestErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h notificationHandler) pushoverSounds(w http.ResponseWriter, r *http.Request) {
	apiToken := r.URL.Query().Get("token")
	if apiToken == "" {
		h.encoder.BadRequestErr(w, errors.New("token parameter is required"))
		return
	}

	sounds, err := pushover.GetSounds(r.Context(), apiToken)
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, sounds)
}

func (h notificationHandler) findInbox(w http.ResponseWriter, r *http.Request) {
	limit, err := parseQueryParamInt(r, "limit", 50)
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	offset, err := parseQueryParamInt(r, "offset", 0)
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	resp, err := h.service.FindInbox(r.Context(), domain.InboxQueryParams{
		Limit:  uint64(max(limit, 1)),
		Offset: uint64(max(offset, 0)),
		Unread: r.URL.Query().Get("unread") == "true",
		Events: r.URL.Query()["event"],
	})
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, resp)
}

func (h notificationHandler) markInboxRead(w http.ResponseWriter, r *http.Request) {
	var data struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil && !errors.Is(err, io.EOF) {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.MarkInboxRead(r.Context(), data.IDs); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h notificationHandler) deleteInbox(w http.ResponseWriter, r *http.Request) {
	if err := h.service.DeleteInbox(r.Context()); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h notificationHandler) deleteInboxMessages(w http.ResponseWriter, r *http.Request) {
	var data struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if len(data.IDs) == 0 {
		h.encoder.BadRequestErr(w, errors.New("ids is required"))
		return
	}

	if err := h.service.DeleteInboxMessages(r.Context(), data.IDs); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
