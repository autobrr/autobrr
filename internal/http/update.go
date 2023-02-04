package http

import (
	"context"
	"net/http"

	"github.com/autobrr/autobrr/pkg/version"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type updateService interface {
	CheckUpdates(ctx context.Context)
	AvailableRelease(ctx context.Context) *version.Release
}

type updateHandler struct {
	encoder encoder
	service updateService
}

func newUpdateHandler(encoder encoder, service updateService) *updateHandler {
	return &updateHandler{
		encoder: encoder,
		service: service,
	}
}

func (h updateHandler) Routes(r chi.Router) {
	r.Get("/", h.getNewUpdates)
}

func (h updateHandler) getNewUpdates(w http.ResponseWriter, r *http.Request) {
	latest := h.service.AvailableRelease(r.Context())
	if latest != nil {
		render.Status(r, http.StatusOK)
		render.JSON(w, r, latest)
		return
	}

	render.Status(r, http.StatusNotFound)
	render.NoContent(w, r)
}
