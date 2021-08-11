package http

import (
	"io/fs"
	"net"
	"net/http"

	"github.com/autobrr/autobrr/internal/config"
	"github.com/autobrr/autobrr/web"

	"github.com/go-chi/chi"
)

type Server struct {
	address               string
	baseUrl               string
	actionService         actionService
	downloadClientService downloadClientService
	filterService         filterService
	indexerService        indexerService
	ircService            ircService
}

func NewServer(address string, baseUrl string, actionService actionService, downloadClientSvc downloadClientService, filterSvc filterService, indexerSvc indexerService, ircSvc ircService) Server {
	return Server{
		address:               address,
		baseUrl:               baseUrl,
		actionService:         actionService,
		downloadClientService: downloadClientSvc,
		filterService:         filterSvc,
		indexerService:        indexerSvc,
		ircService:            ircSvc,
	}
}

func (s Server) Open() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}

	server := http.Server{
		Handler: s.Handler(),
	}

	return server.Serve(listener)
}

func (s Server) Handler() http.Handler {
	r := chi.NewRouter()

	//r.Get("/", index)
	//r.Get("/dashboard", dashboard)

	//handler := web.AssetHandler("/", "build")

	encoder := encoder{}

	assets, _ := fs.Sub(web.Assets, "build/static")
	r.HandleFunc("/static/*", func(w http.ResponseWriter, r *http.Request) {
		fileSystem := http.StripPrefix("/static/", http.FileServer(http.FS(assets)))
		fileSystem.ServeHTTP(w, r)
	})

	r.Group(func(r chi.Router) {

		actionHandler := actionHandler{
			encoder:       encoder,
			actionService: s.actionService,
		}

		r.Route("/api/actions", actionHandler.Routes)

		downloadClientHandler := downloadClientHandler{
			encoder:               encoder,
			downloadClientService: s.downloadClientService,
		}

		r.Route("/api/download_clients", downloadClientHandler.Routes)

		filterHandler := filterHandler{
			encoder:       encoder,
			filterService: s.filterService,
		}

		r.Route("/api/filters", filterHandler.Routes)

		ircHandler := ircHandler{
			encoder:    encoder,
			ircService: s.ircService,
		}

		r.Route("/api/irc", ircHandler.Routes)

		indexerHandler := indexerHandler{
			encoder:        encoder,
			indexerService: s.indexerService,
		}

		r.Route("/api/indexer", indexerHandler.Routes)

		configHandler := configHandler{
			encoder: encoder,
		}

		r.Route("/api/config", configHandler.Routes)
	})

	//r.HandleFunc("/*", handler.ServeHTTP)
	r.Get("/", index)
	r.Get("/*", index)

	return r
}

func index(w http.ResponseWriter, r *http.Request) {
	p := web.IndexParams{
		Title:   "Dashboard",
		Version: "thisistheversion",
		BaseUrl: config.Config.BaseURL,
	}
	web.Index(w, p)
}
