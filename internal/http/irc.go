// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/pkg/errors"

	"github.com/go-chi/chi/v5"
	"github.com/r3labs/sse/v2"
)

type ircService interface {
	ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error)
	GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error)
	DeleteNetwork(ctx context.Context, id int64) error
	GetNetworkByID(ctx context.Context, id int64) (*domain.IrcNetwork, error)
	StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error
	StoreChannel(ctx context.Context, networkID int64, channel *domain.IrcChannel) error
	RestartNetwork(ctx context.Context, id int64) error
	SendCmd(ctx context.Context, req *domain.SendIrcCmdRequest) error
	ManualProcessAnnounce(ctx context.Context, req *domain.IRCManualProcessRequest) error
}

type ircHandler struct {
	encoder encoder
	sse     *sse.Server

	service ircService
}

func newIrcHandler(encoder encoder, sse *sse.Server, service ircService) *ircHandler {
	return &ircHandler{
		encoder: encoder,
		sse:     sse,
		service: service,
	}
}

func (h ircHandler) Routes(r chi.Router) {
	r.Get("/", h.listNetworks)
	r.Post("/", h.storeNetwork)

	r.Route("/network/{networkID}", func(r chi.Router) {
		r.Put("/", h.updateNetwork)
		r.Get("/", h.getNetworkByID)
		r.Delete("/", h.deleteNetwork)

		r.Post("/cmd", h.sendCmd)
		r.Post("/channel", h.storeChannel)
		r.Get("/restart", h.restartNetwork)

		r.Post("/channel/{channel}/announce/process", h.announceProcess)
	})

	r.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {

		// inject CORS headers to bypass checks
		h.sse.Headers = map[string]string{
			"Content-Type":      "text/event-stream",
			"Cache-Control":     "no-cache",
			"Connection":        "keep-alive",
			"X-Accel-Buffering": "no",
		}

		h.sse.ServeHTTP(w, r)
	})
}

func (h ircHandler) listNetworks(w http.ResponseWriter, r *http.Request) {
	networks, err := h.service.GetNetworksWithHealth(r.Context())
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, networks)
}

func (h ircHandler) getNetworkByID(w http.ResponseWriter, r *http.Request) {
	networkID, err := strconv.Atoi(chi.URLParam(r, "networkID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	network, err := h.service.GetNetworkByID(r.Context(), int64(networkID))
	if err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("network with id %d not found", networkID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.StatusResponse(w, http.StatusOK, network)
}

func (h ircHandler) restartNetwork(w http.ResponseWriter, r *http.Request) {
	networkID, err := strconv.Atoi(chi.URLParam(r, "networkID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.RestartNetwork(r.Context(), int64(networkID)); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("network with id %d not found", networkID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h ircHandler) storeNetwork(w http.ResponseWriter, r *http.Request) {
	var data domain.IrcNetwork
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.StoreNetwork(r.Context(), &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h ircHandler) updateNetwork(w http.ResponseWriter, r *http.Request) {
	var data domain.IrcNetwork
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.UpdateNetwork(r.Context(), &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h ircHandler) sendCmd(w http.ResponseWriter, r *http.Request) {
	networkID, err := strconv.Atoi(chi.URLParam(r, "networkID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	var data domain.SendIrcCmdRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	data.NetworkId = int64(networkID)

	if err := h.service.SendCmd(r.Context(), &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

// announceProcess manually trigger announce process
func (h ircHandler) announceProcess(w http.ResponseWriter, r *http.Request) {
	networkID, err := strconv.Atoi(chi.URLParam(r, "networkID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	var data domain.IRCManualProcessRequest
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	data.NetworkId = int64(networkID)
	data.Channel = chi.URLParam(r, "channel")

	// we cant pass # as an url parameter so the frontend has to strip it
	if !strings.HasPrefix(data.Channel, "#") {
		data.Channel = fmt.Sprintf("#%s", data.Channel)
	}

	if err := h.service.ManualProcessAnnounce(r.Context(), &data); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("network with id %d not found", data.NetworkId))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h ircHandler) storeChannel(w http.ResponseWriter, r *http.Request) {
	networkID, err := strconv.Atoi(chi.URLParam(r, "networkID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	var data domain.IrcChannel
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.StoreChannel(r.Context(), int64(networkID), &data); err != nil {
		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}

func (h ircHandler) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	networkID, err := strconv.Atoi(chi.URLParam(r, "networkID"))
	if err != nil {
		h.encoder.Error(w, err)
		return
	}

	if err := h.service.DeleteNetwork(r.Context(), int64(networkID)); err != nil {
		if errors.Is(err, domain.ErrRecordNotFound) {
			h.encoder.NotFoundErr(w, errors.New("network with id %d does not exist", networkID))
			return
		}

		h.encoder.Error(w, err)
		return
	}

	h.encoder.NoContent(w)
}
