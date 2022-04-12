package http

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"

	"github.com/autobrr/autobrr/internal/database"
	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/web"

	"github.com/go-chi/chi"
	"github.com/gorilla/sessions"
	"github.com/r3labs/sse/v2"
	"github.com/rs/cors"
)

type Server struct {
	sse *sse.Server
	db  *database.DB

	config      domain.Config
	cookieStore *sessions.CookieStore

	version string
	commit  string
	date    string

	actionService         actionService
	authService           authService
	downloadClientService downloadClientService
	filterService         filterService
	indexerService        indexerService
	ircService            ircService
	notificationService   notificationService
	releaseService        releaseService
}

func NewServer(config domain.Config, sse *sse.Server, db *database.DB, version string, commit string, date string, actionService actionService, authService authService, downloadClientSvc downloadClientService, filterSvc filterService, indexerSvc indexerService, ircSvc ircService, notificationSvc notificationService, releaseSvc releaseService) Server {
	return Server{
		config:  config,
		sse:     sse,
		db:      db,
		version: version,
		commit:  commit,
		date:    date,

		cookieStore: sessions.NewCookieStore([]byte(config.SessionSecret)),

		actionService:         actionService,
		authService:           authService,
		downloadClientService: downloadClientSvc,
		filterService:         filterSvc,
		indexerService:        indexerSvc,
		ircService:            ircSvc,
		notificationService:   notificationSvc,
		releaseService:        releaseSvc,
	}
}

func (s Server) Open() error {
	addr := fmt.Sprintf("%v:%v", s.config.Host, s.config.Port)
	listener, err := net.Listen("tcp", addr)
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

	c := cors.New(cors.Options{
		AllowCredentials:   true,
		AllowedMethods:     []string{"HEAD", "OPTIONS", "GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowOriginFunc:    func(origin string) bool { return true },
		OptionsPassthrough: true,
		// Enable Debugging for testing, consider disabling in production
		Debug: false,
	})

	r.Use(c.Handler)

	//r.Get("/", index)
	//r.Get("/dashboard", dashboard)

	//handler := web.AssetHandler("/", "build")

	encoder := encoder{}

	assets, _ := fs.Sub(web.Assets, "build/static")
	r.HandleFunc("/static/*", func(w http.ResponseWriter, r *http.Request) {
		fileSystem := http.StripPrefix("/static/", http.FileServer(http.FS(assets)))
		fileSystem.ServeHTTP(w, r)
	})

	r.Route("/api/auth", newAuthHandler(encoder, s.config, s.cookieStore, s.authService).Routes)
	r.Route("/api/healthz", newHealthHandler(encoder, s.db).Routes)

	r.Group(func(r chi.Router) {
		r.Use(s.IsAuthenticated)

		r.Route("/api", func(r chi.Router) {
			r.Route("/actions", newActionHandler(encoder, s.actionService).Routes)
			r.Route("/config", newConfigHandler(encoder, s).Routes)
			r.Route("/download_clients", newDownloadClientHandler(encoder, s.downloadClientService).Routes)
			r.Route("/filters", newFilterHandler(encoder, s.filterService).Routes)
			r.Route("/irc", newIrcHandler(encoder, s.ircService).Routes)
			r.Route("/indexer", newIndexerHandler(encoder, s.indexerService, s.ircService).Routes)
			r.Route("/notification", newNotificationHandler(encoder, s.notificationService).Routes)
			r.Route("/release", newReleaseHandler(encoder, s.releaseService).Routes)

			r.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {

				// inject CORS headers to bypass checks
				s.sse.Headers = map[string]string{
					"Content-Type":      "text/event-stream",
					"Cache-Control":     "no-cache",
					"Connection":        "keep-alive",
					"X-Accel-Buffering": "no",
				}

				s.sse.ServeHTTP(w, r)
			})
		})
	})

	//r.HandleFunc("/*", handler.ServeHTTP)
	r.Get("/", s.index)
	r.Get("/*", s.index)

	return r
}

func (s Server) index(w http.ResponseWriter, r *http.Request) {
	p := web.IndexParams{
		Title:   "Dashboard",
		Version: s.version,
		BaseUrl: s.config.BaseURL,
	}
	web.Index(w, p)
}
