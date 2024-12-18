// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package http

import (
	"bufio"
	"bytes"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

type webHandler struct {
	log          zerolog.Logger
	embedFS      fs.FS
	baseUrl      string
	assetBaseURL string
	version      string
	files        map[string]string
}

func newWebHandler(log zerolog.Logger, embedFS fs.FS, version, baseURL, assetBaseURL string) *webHandler {
	return &webHandler{
		log:          log.With().Str("module", "web-assets").Logger(),
		embedFS:      embedFS,
		baseUrl:      baseURL,
		assetBaseURL: assetBaseURL,
		version:      version,
		files:        make(map[string]string),
	}
}

// registerAssets walks the FS Dist dir and registers each file as a route
func (h *webHandler) registerAssets(r *chi.Mux, baseUrl string) {
	err := fs.WalkDir(h.embedFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			//h.log.Trace().Msgf("web assets: skip dir: %s", d.Name())
			return nil
		}

		//h.log.Trace().Msgf("web assets: found path: %s", path)

		// ignore index.html, so we can render it as a template and inject variables
		if path == "index.html" || path == "manifest.webmanifest" || path == ".gitkeep" {
			return nil
		}

		FileFS(r, filepath.Join("/", path), path, h.embedFS)

		h.files[path] = path

		return nil
	})

	if err != nil {
		return
	}
}

func (h *webHandler) RegisterRoutes(r *chi.Mux) {
	h.registerAssets(r, h.baseUrl)

	// Serve static files without a prefix
	assets, err := fs.Sub(h.embedFS, "assets")
	if err != nil {
		h.log.Error().Err(err).Msg("could not load assets sub dir")
	}

	StaticFSNew(r, h.baseUrl, "/assets", assets)

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

	// handle all other routes
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		file := strings.TrimPrefix(r.RequestURI, h.baseUrl)

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

func (h *webHandler) RenderIndex(w io.Writer, p IndexParams) error {
	return h.parseIndex().Execute(w, p)
}

func (h *webHandler) parseIndex() *template.Template {
	return template.Must(template.New("index.html").ParseFS(h.embedFS, "index.html"))
}

func (h *webHandler) RenderManifest(w io.Writer, p IndexParams) error {
	return h.parseManifest().Execute(w, p)
}

func (h *webHandler) parseManifest() *template.Template {
	return template.Must(template.New("manifest.webmanifest").ParseFS(h.embedFS, "manifest.webmanifest"))
}

func (h *webHandler) RenderFallbackIndex(w io.Writer) error {
	p := IndexParams{
		Title:        "autobrr Dashboard",
		Version:      h.version,
		BaseUrl:      h.baseUrl,
		AssetBaseUrl: h.assetBaseURL,
	}
	return h.parseFallbackIndex().Execute(w, p)
}

func (h *webHandler) parseFallbackIndex() *template.Template {
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

type defaultFS struct {
	prefix string
	fs     fs.FS
}

type IndexParams struct {
	Title        string
	Version      string
	BaseUrl      string
	AssetBaseUrl string
}

func (fs defaultFS) Open(name string) (fs.File, error) {
	if fs.fs == nil {
		return os.Open(name)
	}
	return fs.fs.Open(name)
}

// FileFS registers a new route with path to serve a file from the provided file system.
func FileFS(r *chi.Mux, path, file string, filesystem fs.FS) {
	r.Get(path, StaticFileHandler(file, filesystem))
}

// StaticFileHandler creates a handler function to serve a file from the provided file system.
func StaticFileHandler(file string, filesystem fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fsFile(w, r, file, filesystem)
	}
}

//// StaticFS registers a new route with path prefix to serve static files from the provided file system.
//func StaticFS(r *chi.Mux, pathPrefix string, filesystem fs.FS) {
//	r.Handle(pathPrefix+"*", http.StripPrefix(pathPrefix, http.FileServer(http.FS(filesystem))))
//}

// StaticFSNew registers a new route with path prefix to serve static files from the provided file system.
func StaticFSNew(r *chi.Mux, baseUrl, pathPrefix string, filesystem fs.FS) {
	r.Handle(pathPrefix+"*", http.StripPrefix(path.Join(baseUrl, pathPrefix), http.FileServer(http.FS(filesystem))))
}

// fsFile is a helper function to serve a file from the provided file system.
func fsFile(w http.ResponseWriter, r *http.Request, file string, filesystem fs.FS) {
	f, err := filesystem.Open(file)
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	data, err := io.ReadAll(bufio.NewReader(f))
	if err != nil {
		http.Error(w, "Failed to read the file", http.StatusInternalServerError)
		return
	}

	reader := bytes.NewReader(data)
	http.ServeContent(w, r, file, stat.ModTime(), reader)
}

var validWebRoutes = []string{"filters", "releases", "settings", "logs", "onboard", "login", "logout"}

func validWebRoute(route string) bool {
	if route == "" || route == "/" {
		return true
	}
	for _, valid := range validWebRoutes {
		if strings.HasPrefix(route, valid) {
			return true
		}
	}

	return false
}
