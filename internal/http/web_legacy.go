// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"github.com/autobrr/autobrr/web"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type webLegacyHandler struct {
	log          zerolog.Logger
	embedFS      fs.FS
	baseUrl      string
	assetBaseURL string
	version      string
}

func newWebLegacyHandler(log zerolog.Logger, embedFS fs.FS, version, baseURL, assetBaseURL string) *webLegacyHandler {
	return &webLegacyHandler{
		log:          log.With().Str("module", "web-assets").Logger(),
		embedFS:      embedFS,
		baseUrl:      baseURL,
		assetBaseURL: assetBaseURL,
		version:      version,
	}
}

func (h *webLegacyHandler) RegisterRoutes(r *chi.Mux) {
	// Serve static files without a prefix
	assets, err := fs.Sub(web.DistDirFS, "assets")
	if err != nil {
		h.log.Error().Err(err).Msg("could not load assets sub dir")
	}

	static, err := fs.Sub(web.DistDirFS, "static")
	if err != nil {
		h.log.Error().Err(err).Msg("could not load static sub dir")
	}

	StaticFS(r, "/assets", assets)
	StaticFS(r, "/static", static)
	//StaticFSNew(r, h.baseUrl, "/static", static)
	//StaticFSNew(r, h.baseUrl, "/assets", assets)

	p := IndexParams{
		Title:        "Dashboard",
		Version:      h.version,
		BaseUrl:      h.baseUrl,
		AssetBaseUrl: h.assetBaseURL,
	}

	// serve on base route
	//c.Get(baseUrl, func(w http.ResponseWriter, r *http.Request) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		//h.RenderIndex(w, p)
		if err := h.RenderIndex(w, p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// handle manifest
	r.Get("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		//h.RenderManifest(w, p)
		if err := h.RenderManifest(w, p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// handle all other routes
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		file := strings.TrimPrefix(r.RequestURI, h.baseUrl)

		// if valid web route then serve html
		if validWebRoute(file) || file == "index.html" {
			if err := h.RenderIndex(w, p); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		// if not valid web route then try and serve files
		fsFile(w, r, file, web.DistDirFS)
	})
}

func (h *webLegacyHandler) RenderIndex(w io.Writer, p IndexParams) error {
	return h.parseIndex().Execute(w, p)
}

func (h *webLegacyHandler) parseIndex() *template.Template {
	return template.Must(template.New("index.html").ParseFS(web.Dist, "dist/index.html"))
}

func (h *webLegacyHandler) RenderManifest(w io.Writer, p IndexParams) error {
	return h.parseManifest().Execute(w, p)
}

func (h *webLegacyHandler) parseManifest() *template.Template {
	return template.Must(template.New("manifest.webmanifest").ParseFS(web.Dist, "dist/manifest.webmanifest"))
}

func (h *webLegacyHandler) RenderFallbackIndex(w io.Writer) error {
	p := IndexParams{
		Title:        "autobrr Dashboard",
		Version:      h.version,
		BaseUrl:      h.baseUrl,
		AssetBaseUrl: h.assetBaseURL,
	}
	return h.parseFallbackIndex().Execute(w, p)
}

func (h *webLegacyHandler) parseFallbackIndex() *template.Template {
	return template.Must(template.New("fallback-index").Parse(`<!DOCTYPE html>
<html>
  <head>
    <title>autobrr</title>
    <style>
      @media (prefers-color-scheme: dark) {
		body {
		  color:#fff;
		  background:#333333
        }

       a {
         color: #3d70ea
       }
      }
    </style>
  </head>
  <body>
    <span>Must use base url: <a href="{{.BaseUrl}}">{{.BaseUrl}}</a></span>
  </body>
</html>
`))
}

// StaticFS registers a new route with path prefix to serve static files from the provided file system.
func StaticFS(r *chi.Mux, pathPrefix string, filesystem fs.FS) {
	r.Handle(pathPrefix+"*", http.StripPrefix(pathPrefix, http.FileServer(http.FS(filesystem))))
}
