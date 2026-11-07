// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/matrix"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
	"github.com/r3labs/sse/v2"
)

type matrixService interface {
	ListNetworks(ctx context.Context) ([]domain.MatrixNetwork, error)
	GetNetworkByID(ctx context.Context, id int64) (*domain.MatrixNetwork, error)
	StoreNetwork(ctx context.Context, network *domain.MatrixNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.MatrixNetwork) error
	DeleteNetwork(ctx context.Context, id int64) error
	RestartNetwork(ctx context.Context, id int64) error

	StartLogin(ctx context.Context, id int64) (*domain.MatrixLogin, error)
	SetAccessToken(ctx context.Context, req *domain.MatrixSetTokenRequest) error
	Logout(ctx context.Context, id int64) error

	ManualProcessAnnounce(ctx context.Context, req *domain.MatrixManualProcessRequest) error
	GetMessageHistory(ctx context.Context, networkID int64, room string) ([]domain.MatrixMessage, error)
}

type matrixHandler struct {
	encoder encoder
	sse     *sse.Server
	service matrixService
}

func newMatrixHandler(encoder encoder, sse *sse.Server, service matrixService) *matrixHandler {
	return &matrixHandler{
		encoder: encoder,
		sse:     sse,
		service: service,
	}
}

func (h matrixHandler) Routes(r chi.Router) {
	read := requireScope(domain.APIResourceMatrix, domain.APIAccessRead)
	invoke := requireScope(domain.APIResourceMatrix, domain.APIAccessInvoke)
	write := requireScope(domain.APIResourceMatrix, domain.APIAccessWrite)

	r.With(read).Get("/", h.listNetworks)
	r.With(write).Post("/", h.storeNetwork)

	r.Route("/network/{networkID}", func(r chi.Router) {
		r.With(read).Get("/", h.getNetworkByID)
		r.With(write).Put("/", h.updateNetwork)
		r.With(write).Delete("/", h.deleteNetwork)

		r.With(invoke).Post("/restart", h.restartNetwork)

		r.With(write).Post("/login", h.startLogin)
		r.With(write).Post("/token", h.setAccessToken)
		r.With(write).Post("/logout", h.logout)

		r.With(read).Get("/room/history", h.getMessageHistory)
		r.With(write).Post("/announce/process", h.announceProcess)
	})

	r.With(read).HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		// the SSE server also carries the irc, logs and notifications streams, which the matrix scope does not cover
		if r.URL.Query().Get("stream") != matrix.EventStreamKey {
			h.encoder.BadRequestErr(w, errors.New("unsupported stream: '%s'", r.URL.Query().Get("stream")))
			return
		}

		// headers are set once at startup; writing them per request races with other streams
		h.sse.ServeHTTP(w, r)
	})
}

// getMessageHistory takes the room as a query param because aliases and room ids contain '#', '!' and ':'.
func (h matrixHandler) getMessageHistory(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	room := r.URL.Query().Get("room")
	if room == "" {
		h.encoder.BadRequestErr(w, errors.New("room must not be empty"))
		return
	}

	messages, err := h.service.GetMessageHistory(r.Context(), int64(networkID), room)
	if err != nil {
		if errors.Is(err, domain.ErrMatrixRoomNotFound) {
			h.encoder.NotFoundErr(w, err)
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, messages)
}

func (h matrixHandler) listNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := h.service.ListNetworks(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, networks)
}

func (h matrixHandler) getNetworkByID(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	network, err := h.service.GetNetworkByID(r.Context(), int64(networkID))
	if err != nil {
		h.networkError(w, err, networkID)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, network)
}

func (h matrixHandler) storeNetwork(w http.ResponseWriter, r *http.Request) {
	var data domain.MatrixNetwork
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	data.ID = 0

	if err := data.Validate(); err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.StoreNetwork(r.Context(), &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusCreated, data)
}

func (h matrixHandler) updateNetwork(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	var data domain.MatrixNetwork
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	data.ID = int64(networkID)

	if err := data.Validate(); err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.UpdateNetwork(r.Context(), &data); err != nil {
		h.networkError(w, err, networkID)
		return
	}

	h.encoder.NoContent(w)
}

func (h matrixHandler) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.DeleteNetwork(r.Context(), int64(networkID)); err != nil {
		h.networkError(w, err, networkID)
		return
	}

	h.encoder.NoContent(w)
}

func (h matrixHandler) restartNetwork(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.RestartNetwork(r.Context(), int64(networkID)); err != nil {
		h.networkError(w, err, networkID)
		return
	}

	h.encoder.NoContent(w)
}

func (h matrixHandler) startLogin(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	login, err := h.service.StartLogin(r.Context(), int64(networkID))
	if err != nil {
		h.networkError(w, err, networkID)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, login)
}

func (h matrixHandler) setAccessToken(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	var data domain.MatrixSetTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if data.AccessToken == "" {
		h.encoder.BadRequestErr(w, errors.New("access token must not be empty"))
		return
	}

	data.NetworkID = int64(networkID)

	if err := h.service.SetAccessToken(r.Context(), &data); err != nil {
		h.networkError(w, err, networkID)
		return
	}

	h.encoder.NoContent(w)
}

func (h matrixHandler) logout(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	if err := h.service.Logout(r.Context(), int64(networkID)); err != nil {
		h.networkError(w, err, networkID)
		return
	}

	h.encoder.NoContent(w)
}

func (h matrixHandler) announceProcess(w http.ResponseWriter, r *http.Request) {
	networkID, err := parseURLParamInt(r, "networkID")
	if err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	var data domain.MatrixManualProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	data.NetworkID = int64(networkID)

	if err := h.service.ManualProcessAnnounce(r.Context(), &data); err != nil {
		h.encoder.BadRequestErr(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h matrixHandler) networkError(w http.ResponseWriter, err error, networkID int) {
	if errors.Is(err, domain.ErrRecordNotFound) {
		h.encoder.NotFoundErr(w, errors.New("network with id %d not found", networkID))
		return
	}

	h.encoder.Error(w, err)
}
