package http

import (
	"net/http"

	"github.com/autobrr/autobrr/internal/config"

	"github.com/go-chi/chi"
)

type configJson struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	LogLevel string `json:"log_level"`
	LogPath  string `json:"log_path"`
	BaseURL  string `json:"base_url"`
}

type configHandler struct {
	encoder encoder
}

func (h configHandler) Routes(r chi.Router) {
	r.Get("/", h.getConfig)
}

func (h configHandler) getConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	c := config.Config

	conf := configJson{
		Host:     c.Host,
		Port:     c.Port,
		LogLevel: c.LogLevel,
		LogPath:  c.LogPath,
		BaseURL:  c.BaseURL,
	}

	h.encoder.StatusResponse(ctx, w, conf, http.StatusOK)
}
