// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

// Package web web/build.go
package web

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

type defaultFS struct {
	prefix string
	fs     fs.FS
}

type IndexParams struct {
	Title   string
	Version string
	BaseUrl string
}

var (
	//go:embed all:dist
	Dist embed.FS

	DistDirFS = MustSubFS(Dist, "dist")
)

func (fs defaultFS) Open(name string) (fs.File, error) {
	if fs.fs == nil {
		return os.Open(name)
	}
	return fs.fs.Open(name)
}

// MustSubFS creates sub FS from current filesystem or panic on failure.
// Panic happens when `fsRoot` contains invalid path according to `fs.ValidPath` rules.
//
// MustSubFS is helpful when dealing with `embed.FS` because for example `//go:embed assets/images` embeds files with
// paths including `assets/images` as their prefix. In that case use `fs := MustSubFS(fs, "rootDirectory") to
// create sub fs which uses necessary prefix for directory path.
func MustSubFS(currentFs fs.FS, fsRoot string) fs.FS {
	subFs, err := subFS(currentFs, fsRoot)
	if err != nil {
		panic(fmt.Errorf("can not create sub FS, invalid root given, err: %w", err))
	}
	return subFs
}

func subFS(currentFs fs.FS, root string) (fs.FS, error) {
	root = filepath.ToSlash(filepath.Clean(root)) // note: fs.FS operates only with slashes. `ToSlash` is necessary for Windows
	if dFS, ok := currentFs.(*defaultFS); ok {
		// we need to make exception for `defaultFS` instances as it interprets root prefix differently from fs.FS.
		// fs.Fs.Open does not like relative paths ("./", "../") and absolute paths.
		if !filepath.IsAbs(root) {
			root = filepath.Join(dFS.prefix, root)
		}
		return &defaultFS{
			prefix: root,
			fs:     os.DirFS(root),
		}, nil
	}
	return fs.Sub(currentFs, root)
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

// StaticFS registers a new route with path prefix to serve static files from the provided file system.
func StaticFS(r *chi.Mux, pathPrefix string, filesystem fs.FS) {
	r.Handle(pathPrefix+"*", http.StripPrefix(pathPrefix, http.FileServer(http.FS(filesystem))))
}

// fsFile is a helper function to serve a file from the provided file system.
func fsFile(w http.ResponseWriter, r *http.Request, file string, filesystem fs.FS) {
	//fmt.Printf("file: %s\n", file)
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

var validRoutes = []string{"/", "filters", "releases", "settings", "logs", "onboard", "login", "logout"}

func validRoute(route string) bool {
	for _, valid := range validRoutes {
		if strings.Contains(route, valid) {
			return true
		}
	}

	return false
}

// RegisterHandler register web routes and file serving
func RegisterHandler(c *chi.Mux, version, baseUrl string) {
	// Serve static files without a prefix
	assets, _ := fs.Sub(DistDirFS, "assets")
	static, _ := fs.Sub(DistDirFS, "static")
	StaticFS(c, "/assets", assets)
	StaticFS(c, "/static", static)

	p := IndexParams{
		Title:   "Dashboard",
		Version: version,
		BaseUrl: baseUrl,
	}

	// serve on base route
	c.Get("/", func(w http.ResponseWriter, r *http.Request) {
		Index(w, p)
	})

	// handle all other routes
	c.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		file := strings.TrimPrefix(r.RequestURI, "/")

		// if valid web route then serve html
		if validRoute(file) || file == "index.html" {
			Index(w, p)
			return
		}

		if strings.Contains(file, "manifest.webmanifest") {
			Manifest(w, p)
			return
		}

		// if not valid web route then try and serve files
		fsFile(w, r, file, DistDirFS)
	})
}

func Index(w io.Writer, p IndexParams) error {
	return parseIndex().Execute(w, p)
}

func parseIndex() *template.Template {
	return template.Must(template.New("index.html").ParseFS(Dist, "dist/index.html"))
}

func Manifest(w io.Writer, p IndexParams) error {
	return parseManifest().Execute(w, p)
}

func parseManifest() *template.Template {
	return template.Must(template.New("manifest.webmanifest").ParseFS(Dist, "dist/manifest.webmanifest"))
}
