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
	authService           authService
	downloadClientService downloadClientService
	filterService         filterService
	indexerService        indexerService
	ircService            ircService
}

func NewServer(address string, baseUrl string, actionService actionService, authService authService, downloadClientSvc downloadClientService, filterSvc filterService, indexerSvc indexerService, ircSvc ircService) Server {
	return Server{
		address:               address,
		baseUrl:               baseUrl,
		actionService:         actionService,
		authService:           authService,
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

	r.Route("/api/auth", newAuthHandler(encoder, s.authService).Routes)

	r.Group(func(r chi.Router) {
		r.Use(IsAuthenticated)

		r.Route("/api", func(r chi.Router) {
			r.Route("/actions", newActionHandler(encoder, s.actionService).Routes)
			r.Route("/config", newConfigHandler(encoder).Routes)
			r.Route("/download_clients", newDownloadClientHandler(encoder, s.downloadClientService).Routes)
			r.Route("/filters", newFilterHandler(encoder, s.filterService).Routes)
			r.Route("/irc", newIrcHandler(encoder, s.ircService).Routes)
			r.Route("/indexer", newIndexerHandler(encoder, s.indexerService, s.ircService).Routes)
		})
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
