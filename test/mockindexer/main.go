// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/autobrr/autobrr/test/mockindexer/irc"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/spf13/pflag"
)

func main() {
	var (
		port      int
		channel   string
		announcer string
	)

	pflag.IntVarP(&port, "port", "p", 3999, "http port. Default: 3999")
	pflag.StringVar(&channel, "irc-channel", "#announces", "Announce channel. Default: #announces")
	pflag.StringVar(&announcer, "irc-announcer", "_AnnounceBot_", "Announcer. Default: _AnnounceBot_")

	pflag.Parse()

	log.Print("MockIndexer starting..")

	options := &irc.ServerOptions{
		BotName: announcer,
		Channel: channel,
	}

	ircServer, err := irc.NewServer(options)
	if err != nil {
		log.Fatalf("Err: %v", err)
	}

	go ircServer.Run()

	api := NewAPI(ircServer)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	if err := api.ListenAndServe(fmt.Sprintf(":%d", port)); err != nil {
		log.Fatalf("could not start mock indexer api server, %v", err)
	}

	for sig := range sigCh {
		log.Printf("received signal: %v", sig)
		os.Exit(0)
	}
}

type Api struct {
	router *chi.Mux

	ircServer *irc.Server

	announces []string
}

func NewAPI(ircServer *irc.Server) *Api {
	a := &Api{
		ircServer: ircServer,
		announces: make([]string, 0),
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", a.indexHandler)
	r.Post("/send", a.formHandler)

	r.Get("/file/{fileId}/{fileName}", a.fileDownloadHandler)
	r.Get("/torrent/download/{torrentId}", a.torrentDownloadHandler)
	r.Get("/feeds/{name}", a.feedHandler)
	r.Post("/webhook", a.webhookHandler)

	a.router = r

	return a
}

func (a *Api) ListenAndServe(addr string) error {
	go func() {
		if err := http.ListenAndServe(addr, a.router); err != nil {
			log.Printf("err: %q", err)
			//return err
		}
	}()

	log.Printf("API running on %q", addr)

	return nil
}

// formHandler takes in an HTTP response writer and request and handles the logic
// for processing form data. It parses the form data using r.ParseForm() and checks for any errors.
// If there is an error parsing the form data, it returns an internal server error response.
// It then retrieves the value of the "line" form field using r.Form.Get("line").
// If the "line" value is not empty, it appends it to the "announces" slice
// and sends the line to all clients using a.a.ircServer.SendAll(line).
// Finally, it redirects the user to the index page ("/") with a 302 status code.
func (a *Api) formHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		render.Status(r, http.StatusInternalServerError)
		return
	}

	line := r.Form.Get("line")
	if line != "" {
		a.announces = append(a.announces, line)

		a.ircServer.SendAll(line)
	}

	http.Redirect(w, r, "/", 302)
}

func (a *Api) indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("body").Parse(htmlBody)
	if err != nil {
		log.Fatal(err)
	}

	data := struct {
		Lines []string
	}{
		Lines: a.announces,
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		log.Fatal(err)
	}
}

// feedHandler takes in an HTTP response writer and request, and handles the logic
// for retrieving and serving an RSS feed based on the provided 'name' URL parameter.
// If the 'name' parameter is missing, it returns a bad request response.
// If there is an error reading the feed file, it returns an internal server error response.
// It sets the 'Content-Type' header of the response to 'application/rss+xml'.
// It writes the feed payload to the response body.
func (a *Api) feedHandler(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		render.Status(r, http.StatusBadRequest)
		render.PlainText(w, r, "bad request: missing param name")
		return
	}

	f, err := os.Open("feeds/" + name + ".xml")
	if err != nil {
		log.Printf("Err: %v", err)
		render.Status(r, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/rss+xml")
	io.Copy(w, bufio.NewReader(f))
}

// fileDownloadHandler takes in an HTTP response writer and request, and handles the logic
// for retrieving and serving a file based on the provided 'fileId' and 'fileName' URL parameters.
// If either parameter is missing, it returns a bad request response.
// It opens the file with the corresponding fileId and ".torrent" extension.
// If there is an error opening the file, it logs the error and terminates the program.
// It sets the 'Content-Disposition' header of the response to indicate the file should be downloaded with the provided fileName.
// It sets the 'Content-Type' header of the response to 'application/x-bittorrent'.
// It copies the file contents to the response body.
func (a *Api) fileDownloadHandler(w http.ResponseWriter, r *http.Request) {
	fileIdParam := chi.URLParam(r, "fileId")
	if fileIdParam == "" {
		render.Status(r, http.StatusBadRequest)
		render.PlainText(w, r, "bad request: missing param fileId")
		return
	}

	fileNameParam := chi.URLParam(r, "fileName")
	if fileNameParam == "" {
		render.Status(r, http.StatusBadRequest)
		render.PlainText(w, r, "bad request: missing param fileName")
		return
	}

	f, err := os.Open("files/" + fileIdParam + ".torrent")
	if err != nil {
		log.Printf("Err: %v", err)
		render.Status(r, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+fileNameParam)
	w.Header().Set("Content-Type", "application/x-bittorrent")

	io.Copy(w, bufio.NewReader(f))
}

// torrentDownloadHandler takes in an HTTP response writer and request, and handles the logic
// for retrieving and serving a file based on the provided 'torrentId' URL parameters.
// If either parameter is missing, it returns a bad request response.
// It opens the file with the corresponding fileId and ".torrent" extension.
// If there is an error opening the file, it logs the error and terminates the program.
// It sets the 'Content-Disposition' header of the response to indicate the file should be downloaded with the provided fileName.
// It sets the 'Content-Type' header of the response to 'application/x-bittorrent'.
// It copies the file contents to the response body.
func (a *Api) torrentDownloadHandler(w http.ResponseWriter, r *http.Request) {
	fileIdParam := chi.URLParam(r, "torrentId")
	if fileIdParam == "" {
		render.Status(r, http.StatusBadRequest)
		render.PlainText(w, r, "bad request: missing param fileId")
		return
	}

	f, err := os.Open("files/" + fileIdParam + ".torrent")
	if err != nil {
		log.Printf("Err: %v", err)
		render.Status(r, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+fileIdParam+".torrent")
	w.Header().Set("Content-Type", "application/x-bittorrent")

	io.Copy(w, bufio.NewReader(f))
}

// webhookHandler takes in an HTTP response writer and request, and handles the logic
// for processing webhook requests.
// If the 'timeout' query parameter is provided, it sleeps for the specified duration.
// If the 'status' query parameter is provided, it sets the response status to the provided value.
// If none of the query parameters are provided, it sets the response status to http.StatusOK by default.
func (a *Api) webhookHandler(w http.ResponseWriter, r *http.Request) {
	body, err := httputil.DumpRequest(r, true)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		return
	}

	log.Println(string(body))

	if timeout := r.URL.Query().Get("timeout"); timeout != "" {
		t, err := strconv.Atoi(timeout)
		if err != nil || t <= 0 || t > 60 { // Set a maximum limit for timeout
			render.Status(r, http.StatusInternalServerError)
			return
		}

		time.Sleep(time.Duration(t) * time.Second) // Changed t to time.Duration(t) to match type
	}

	if status := r.URL.Query().Get("status"); status != "" {
		s, err := strconv.Atoi(status)
		if err != nil {
			log.Printf("Err: %v", err)
			render.Status(r, http.StatusInternalServerError)
			return
		}

		render.Status(r, s)
		return
	}

	render.Status(r, http.StatusOK)
}

var htmlBody = `
<!DOCTYPE html>
<html>
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <script src="https://cdn.tailwindcss.com"></script>
</head>
<body>
  <div class="min-h-full">
    <div class="py-10">
      <main>
        <div class="mx-auto max-w-7xl sm:px-6 lg:px-8">
          <!-- Your content -->
          <form method="POST" action="/send">
            Send an announce line to the channel<br>
            <div class="flex">
              <input style="width: 100%; margin-top: 5px; margin-bottom: 5px;" name="line" type="text"
                class="block w-full rounded-md border-0 py-1.5 text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-indigo-600 sm:text-sm sm:leading-6" />
              <button type="submit"
                class="rounded bg-indigo-600 ml-2 px-2 py-1 text-sm font-semibold text-white shadow-sm hover:bg-indigo-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-600">
                Send
              </button>
            </div>
          </form>
          {{range .Lines}}
          <div class="mb-2">
            <form method="POST" action="/send" class="truncate">
              <button type="submit"
                class="rounded bg-white px-2 py-1 text-sm font-semibold text-gray-900 shadow-sm ring-1 ring-inset ring-gray-300 hover:bg-gray-50">
                Re-send
              </button>
              <label for="line">{{.}}</label>
              <input name="line" id="line" value="{{.}}" hidden />
            </form>
          </div>
          {{end}}
        </div>
      </main>
    </div>
  </div>
</body>
</html>
`
