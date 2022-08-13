package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type configJson struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	LogLevel string `json:"log_level"`
	LogPath  string `json:"log_path"`
	BaseURL  string `json:"base_url"`
	Version  string `json:"version"`
	Commit   string `json:"commit"`
	Date     string `json:"date"`
}

type configHandler struct {
	encoder encoder

	server Server
}

func newConfigHandler(encoder encoder, server Server) *configHandler {
	return &configHandler{
		encoder: encoder,
		server:  server,
	}
}

func (h configHandler) Routes(r chi.Router) {
	r.Get("/", h.getConfig)
}

func (h configHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	conf := configJson{
		Host:     h.server.config.Host,
		Port:     h.server.config.Port,
		LogLevel: h.server.config.LogLevel,
		LogPath:  h.server.config.LogPath,
		BaseURL:  h.server.config.BaseURL,
		Version:  h.server.version,
		Commit:   h.server.commit,
		Date:     h.server.date,
	}

	h.encoder.StatusResponse(ctx, w, conf, http.StatusOK)
}
