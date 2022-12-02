package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/autobrr/autobrr/test/mockindexer/irc"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	s, err := irc.NewServer(&irc.ServerOptions{
		BotName: "_AnnounceBot_",
		Channel: "#announces",
	})

	if err != nil {
		log.Fatalf("Err: %v", err)
	}

	go s.Run()

	log.Print("autobrr MockIndexer running")

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html><form method=\"POST\" action=\"/send\">Send an announce line to the channel<br><input style=\"width: 100%; margin-top: 5px; margin-bottom: 5px;\" name=\"line\" type=\"text\"><br><button type=\"submit\">Send to channel</button></form></html>"))
	})

	r.Get("/file/{fileId}/{fileName}", func(w http.ResponseWriter, r *http.Request) {
		f, err := os.Open("files/" + chi.URLParam(r, "fileId") + ".torrent")

		if err != nil {
			log.Fatalf("Err: %v", err)
		}

		w.Header().Set("Content-Disposition", "attachment; filename="+chi.URLParam(r, "fileName"))
		w.Header().Set("Content-Type", "application/x-bittorrent")

		io.Copy(w, bufio.NewReader(f))
	})

	r.Post("/send", func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()

		line := r.Form.Get("line")
		s.SendAll(line)

		http.Redirect(w, r, "/", 302)
	})

	http.ListenAndServe("localhost:3999", r)
}
