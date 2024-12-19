// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"html/template"
	"io"
	"io/fs"
	"net/http"
	filePath "path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type webLegacyHandler struct {
	log          zerolog.Logger
	embedFS      fs.FS
	baseUrl      string
	assetBaseURL string
	version      string

	files map[string]string
}

func newWebLegacyHandler(log zerolog.Logger, embedFS fs.FS, version, baseURL, assetBaseURL string) *webLegacyHandler {
	return &webLegacyHandler{
		log:          log.With().Str("module", "web-assets").Logger(),
		embedFS:      embedFS,
		baseUrl:      baseURL,
		assetBaseURL: assetBaseURL,
		version:      version,
		files:        make(map[string]string),
	}
}

// registerAssets walks the FS Dist dir and registers each file as a route
func (h *webLegacyHandler) registerAssets(r *chi.Mux) {
	err := fs.WalkDir(h.embedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			//h.log.Trace().Msgf("web assets: skip dir: %s", d.Name())
			return nil
		}

		h.log.Trace().Msgf("web assets: found path: %s", path)

		// ignore index.html, so we can render it as a template and inject variables
		if path == "index.html" || path == "manifest.webmanifest" || path == ".gitkeep" {
			return nil
		}

		// use old path.Join to not be os specific
		FileFS(r, filePath.Join("/", path), path, h.embedFS)

		h.files[path] = path

		return nil
	})

	if err != nil {
		return
	}
}

func (h *webLegacyHandler) RegisterRoutes(r *chi.Mux) {
	h.registerAssets(r)

	// Serve static files without a prefix
	assets, err := fs.Sub(h.embedFS, "assets")
	if err != nil {
		h.log.Error().Err(err).Msg("could not load assets sub dir")
	}

	StaticFS(r, "/assets", assets)

	p := IndexParams{
		Title:        "Dashboard",
		Version:      h.version,
		BaseUrl:      h.baseUrl,
		AssetBaseUrl: h.assetBaseURL,
	}

	// serve on base route
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if err := h.RenderIndex(w, p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// handle manifest
	r.Get("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		if err := h.RenderManifest(w, p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	// handle all other routes and files
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		file := strings.TrimPrefix(r.RequestURI, h.baseUrl)

		if strings.Contains(file, "/assets") {
			if strings.HasPrefix(file, "/assets/") {
				fsFile(w, r, file, h.embedFS)
				return
			}

			parts := strings.SplitAfter(file, "/assets/")
			if len(parts) > 1 {
				fsFile(w, r, "assets/"+parts[1], h.embedFS)
				return
			}
			return
		}

		if strings.Contains(file, "favicon.ico") {
			fsFile(w, r, "favicon.ico", h.embedFS)
			return
		}

		if strings.Contains(file, "Inter-Variable.woff2") {
			fsFile(w, r, "Inter-Variable.woff2", h.embedFS)
			return
		}

		// if valid web route then serve html
		if validWebRoute(file) || file == "index.html" {
			if err := h.RenderIndex(w, p); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			return
		}

		// if not valid web route then try and serve files
		fsFile(w, r, file, h.embedFS)
	})
}

func (h *webLegacyHandler) RenderIndex(w io.Writer, p IndexParams) error {
	return h.parseIndex().Execute(w, p)
}

func (h *webLegacyHandler) parseIndex() *template.Template {
	return template.Must(template.New("index.html").ParseFS(h.embedFS, "index.html"))
}

func (h *webLegacyHandler) RenderManifest(w io.Writer, p IndexParams) error {
	return h.parseManifest().Execute(w, p)
}

func (h *webLegacyHandler) parseManifest() *template.Template {
	return template.Must(template.New("manifest.webmanifest").ParseFS(h.embedFS, "manifest.webmanifest"))
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
