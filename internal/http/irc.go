package http

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"

	"github.com/autobrr/autobrr/internal/domain"
)

type ircService interface {
	ListNetworks(ctx context.Context) ([]domain.IrcNetwork, error)
	GetNetworksWithHealth(ctx context.Context) ([]domain.IrcNetworkWithHealth, error)
	DeleteNetwork(ctx context.Context, id int64) error
	GetNetworkByID(id int64) (*domain.IrcNetwork, error)
	StoreNetwork(ctx context.Context, network *domain.IrcNetwork) error
	UpdateNetwork(ctx context.Context, network *domain.IrcNetwork) error
	StoreChannel(networkID int64, channel *domain.IrcChannel) error
	StopNetwork(name string) error
}

type ircHandler struct {
	encoder encoder
	service ircService
}

func newIrcHandler(encoder encoder, service ircService) *ircHandler {
	return &ircHandler{
		encoder: encoder,
		service: service,
	}
}

func (h ircHandler) Routes(r chi.Router) {
	r.Get("/", h.listNetworks)
	r.Post("/", h.storeNetwork)
	r.Put("/network/{networkID}", h.updateNetwork)
	r.Post("/network/{networkID}/channel", h.storeChannel)
	r.Get("/network/{networkID}/stop", h.stopNetwork)
	r.Get("/network/{networkID}", h.getNetworkByID)
	r.Delete("/network/{networkID}", h.deleteNetwork)
}

func (h ircHandler) listNetworks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	networks, err := h.service.GetNetworksWithHealth(ctx)
	if err != nil {
		//
	}

	h.encoder.StatusResponse(ctx, w, networks, http.StatusOK)
}

func (h ircHandler) getNetworkByID(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		networkID = chi.URLParam(r, "networkID")
	)

	id, _ := strconv.Atoi(networkID)

	network, err := h.service.GetNetworkByID(int64(id))
	if err != nil {
		//
	}

	h.encoder.StatusResponse(ctx, w, network, http.StatusOK)
}

func (h ircHandler) storeNetwork(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.IrcNetwork
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		return
	}

	err := h.service.StoreNetwork(ctx, &data)
	if err != nil {
		//
		h.encoder.StatusResponse(ctx, w, nil, http.StatusBadRequest)
		return
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusCreated)
}

func (h ircHandler) updateNetwork(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.IrcNetwork
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		return
	}

	err := h.service.UpdateNetwork(ctx, &data)
	if err != nil {
		//
		h.encoder.StatusResponse(ctx, w, nil, http.StatusBadRequest)
		return
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusCreated)
}

func (h ircHandler) storeChannel(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		data      domain.IrcChannel
		networkID = chi.URLParam(r, "networkID")
	)

	id, _ := strconv.Atoi(networkID)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		return
	}

	err := h.service.StoreChannel(int64(id), &data)
	if err != nil {
		//
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusCreated)
}

func (h ircHandler) stopNetwork(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		networkID = chi.URLParam(r, "networkID")
	)

	err := h.service.StopNetwork(networkID)
	if err != nil {
		//
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusCreated)
}

func (h ircHandler) deleteNetwork(w http.ResponseWriter, r *http.Request) {
	var (
		ctx       = r.Context()
		networkID = chi.URLParam(r, "networkID")
	)

	id, _ := strconv.Atoi(networkID)

	err := h.service.DeleteNetwork(ctx, int64(id))
	if err != nil {
		//
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}
