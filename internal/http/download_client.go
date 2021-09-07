package http

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi"

	"github.com/autobrr/autobrr/internal/domain"
)

type downloadClientService interface {
	List() ([]domain.DownloadClient, error)
	Store(client domain.DownloadClient) (*domain.DownloadClient, error)
	Delete(clientID int) error
	Test(client domain.DownloadClient) error
}

type downloadClientHandler struct {
	encoder encoder
	service downloadClientService
}

func newDownloadClientHandler(encoder encoder, service downloadClientService) *downloadClientHandler {
	return &downloadClientHandler{
		encoder: encoder,
		service: service,
	}
}

func (h downloadClientHandler) Routes(r chi.Router) {
	r.Get("/", h.listDownloadClients)
	r.Post("/", h.store)
	r.Put("/", h.update)
	r.Post("/test", h.test)
	r.Delete("/{clientID}", h.delete)
}

func (h downloadClientHandler) listDownloadClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	clients, err := h.service.List()
	if err != nil {
		//
	}

	h.encoder.StatusResponse(ctx, w, clients, http.StatusOK)
}

func (h downloadClientHandler) store(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.DownloadClient
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	client, err := h.service.Store(data)
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, client, http.StatusCreated)
}

func (h downloadClientHandler) test(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.DownloadClient
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		h.encoder.StatusResponse(ctx, w, nil, http.StatusBadRequest)
		return
	}

	err := h.service.Test(data)
	if err != nil {
		// encode error
		h.encoder.StatusResponse(ctx, w, nil, http.StatusBadRequest)
		return
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}

func (h downloadClientHandler) update(w http.ResponseWriter, r *http.Request) {
	var (
		ctx  = r.Context()
		data domain.DownloadClient
	)

	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		// encode error
		return
	}

	client, err := h.service.Store(data)
	if err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, client, http.StatusCreated)
}

func (h downloadClientHandler) delete(w http.ResponseWriter, r *http.Request) {
	var (
		ctx      = r.Context()
		clientID = chi.URLParam(r, "clientID")
	)

	// if !clientID return error

	id, _ := strconv.Atoi(clientID)

	if err := h.service.Delete(id); err != nil {
		// encode error
	}

	h.encoder.StatusResponse(ctx, w, nil, http.StatusNoContent)
}
