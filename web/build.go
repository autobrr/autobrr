// Package web web/build.go
package web

import (
	"embed"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

//go:embed dist/*
var Assets embed.FS

// fsFunc is short-hand for constructing a http.FileSystem
// implementation
type fsFunc func(name string) (fs.File, error)

func (f fsFunc) Open(name string) (fs.File, error) {
	return f(name)
}

// AssetHandler returns a http.Handler that will serve files from
// the Assets embed.FS.  When locating a file, it will strip the given
// prefix from the request and prepend the root to the filesystem
// lookup: typical prefix might be /web/, and root would be build.
func AssetHandler(prefix, root string) http.Handler {
	handler := fsFunc(func(name string) (fs.File, error) {
		assetPath := path.Join(root, name)

		// If we can't find the asset, return the default index.html
		// content
		f, err := Assets.Open(assetPath)
		if os.IsNotExist(err) {
			if strings.HasPrefix(name, "/assets") || strings.HasPrefix(name, "/static") {
				return Assets.Open("dist" + name)
			} else {
				return Assets.Open("dist/index.html")
			}
		}

		// Otherwise, assume this is a legitimate request routed
		// correctly
		return f, err
	})

	return http.StripPrefix(prefix, http.FileServer(http.FS(handler)))
}

type IndexParams struct {
	Title   string
	Version string
	BaseUrl string
}

func Index(w io.Writer, p IndexParams) error {
	return parseIndex().Execute(w, p)
}

func parseIndex() *template.Template {
	return template.Must(
		template.New("index.html").ParseFS(Assets, "dist/index.html"))
}
