package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/go-chi/chi"
)

type notificationService interface {
	Find(context.Context, domain.NotificationQueryParams) ([]domain.Notification, int, error)
	FindByID(ctx context.Context, id int) (*domain.Notification, error)
	Store(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Update(ctx context.Context, n domain.Notification) (*domain.Notification, error)
	Delete(ctx context.Context, id int) error
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
	r.Put("/{notificationID}", h.update)
	r.Delete("/{notificationID}", h.delete)
}

func (h notificationHandler) list(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	list, _, err := h.service.Find(ctx, domain.NotificationQueryParams{})
	if err != nil {
		h.encoder.StatusNotFound(ctx, w)
		return
	}

	h.encoder.StatusResponse(ctx, w, list, http.StatusOK)
}

func (h notificationHandler) store(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Notification
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	filter, err := h.service.Store(ctx, data)
	if err != nil {
		// encode error
		return
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusCreated)
}

func (h notificationHandler) update(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.Notification
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	filter, err := h.service.Update(ctx, data)
	if err != nil {
		// encode error
		return
	}

	h.encoder.StatusResponse(ctx, w, filter, http.StatusOK)
}

func (h notificationHandler) delete(w http.ResponseWriter, r *http.Request) {
	var (
		ctx            = r.Context()
		notificationID = chi.URLParam(r, "notificationID")
	)

	id, _ := strconv.Atoi(notificationID)

	if err := h.service.Delete(ctx, id); err != nil {
		// return err
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}
