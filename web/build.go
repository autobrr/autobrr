// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

// Package web web/build.go
package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

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

	data, err := io.ReadAll(f)
	if err != nil {
		http.Error(w, "Failed to read the file", http.StatusInternalServerError)
		return
	}

	reader := bytes.NewReader(data)
	http.ServeContent(w, r, file, stat.ModTime(), reader)
}

func RegisterHandler(c *chi.Mux) {
	// Serve static files without a prefix
	assets, _ := fs.Sub(DistDirFS, "assets")
	static, _ := fs.Sub(DistDirFS, "static")
	StaticFS(c, "/assets", assets)
	StaticFS(c, "/static", static)

	c.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for unmatched routes
		fsFile(w, r, "dist/index.html", Dist)
	})
}

func Index(w io.Writer, p IndexParams) error {
	return parseIndex().Execute(w, p)
}

func parseIndex() *template.Template {
	return template.Must(template.New("index.html").ParseFS(Dist, "dist/index.html"))
}
