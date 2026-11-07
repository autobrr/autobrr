// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"net/http"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
)

type webhookHandler struct {
	encoder encoder
	listSvc listService
}

func newWebhookHandler(encoder encoder, listSvc listService) *webhookHandler {
	return &webhookHandler{encoder: encoder, listSvc: listSvc}
}

func (h *webhookHandler) Routes(r chi.Router) {
	// the webhooks only trigger list refreshes, which is what lists:invoke grants
	invoke := requireScope(domain.APIResourceLists, domain.APIAccessInvoke)

	r.Route("/lists", func(r chi.Router) {
		r.With(invoke).Post("/trigger", h.refreshAll)
		r.With(invoke).Post("/trigger/arr", h.refreshArr)
		r.With(invoke).Post("/trigger/lists", h.refreshLists)

		r.With(invoke).Get("/trigger", h.refreshAll)
		r.With(invoke).Get("/trigger/arr", h.refreshArr)
		r.With(invoke).Get("/trigger/lists", h.refreshLists)

		r.With(invoke).Post("/trigger/{listID}", h.refreshByID)
	})
}

func (h *webhookHandler) refreshAll(w http.ResponseWriter, r *http.Request) {
	go h.listSvc.RefreshAll(context.Background())

	h.encoder.NoContent(w)
}

func (h *webhookHandler) refreshByID(w http.ResponseWriter, r *http.Request) {
	listID, err := parseURLParamInt(r, "listID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.listSvc.RefreshList(context.Background(), int64(listID)); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("could not find list with id %d", listID))
			return
		}

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
