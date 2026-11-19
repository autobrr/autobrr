// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// Push is one release a mock download client received.
type Push struct {
	Name     string    `json:"name"`
	Category string    `json:"category,omitempty"`
	Approved bool      `json:"approved"`
	Error    string    `json:"error,omitempty"`
	At       time.Time `json:"at"`
}

// recorder collects pushes and requests to endpoints a mock does not implement,
// which usually means autobrr started calling something new.
type recorder struct {
	m       sync.Mutex
	pushes  []Push
	unknown []string
}

func (r *recorder) record(p Push) {
	r.m.Lock()
	defer r.m.Unlock()

	r.pushes = append(r.pushes, p)
}

func (r *recorder) recordUnknown(req *http.Request) {
	r.m.Lock()
	defer r.m.Unlock()

	r.unknown = append(r.unknown, req.Method+" "+req.URL.Path)
}

// Pushes returns everything received so far.
func (r *recorder) Pushes() []Push {
	r.m.Lock()
	defer r.m.Unlock()

	return slices.Clone(r.pushes)
}

// Unknown lists requests to endpoints the mock does not implement.
func (r *recorder) Unknown() []string {
	r.m.Lock()
	defer r.m.Unlock()

	return slices.Clone(r.unknown)
}

const (
	qbitUsername = "admin"
	qbitPassword = "simulation"
	qbitSID      = "simulation-sid"
)

// Qbittorrent mocks the qBittorrent WebUI API endpoints autobrr uses to add torrents.
type Qbittorrent struct {
	recorder

	server *httptest.Server
}

// StartQbittorrent serves the mock until the test ends.
func StartQbittorrent(t testing.TB) *Qbittorrent {
	t.Helper()

	q := &Qbittorrent{}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v2/auth/login", q.login)
	mux.HandleFunc("POST /api/v2/torrents/add", q.add)
	mux.HandleFunc("GET /api/v2/torrents/trackers", q.trackers)
	mux.HandleFunc("GET /api/v2/app/version", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("v5.0.0"))
	})
	mux.HandleFunc("GET /api/v2/app/webapiVersion", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("2.11.0"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q.recordUnknown(r)
		http.NotFound(w, r)
	})

	q.server = httptest.NewServer(mux)
	t.Cleanup(q.server.Close)

	return q
}

// URL is the WebUI address to configure the download client with.
func (q *Qbittorrent) URL() string {
	return q.server.URL
}

func (q *Qbittorrent) login(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if r.PostForm.Get("username") != qbitUsername || r.PostForm.Get("password") != qbitPassword {
		_, _ = w.Write([]byte("Fails."))
		return
	}

	http.SetCookie(w, &http.Cookie{Name: "SID", Value: qbitSID, Path: "/"})
	_, _ = w.Write([]byte("Ok."))
}

func (q *Qbittorrent) add(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie("SID"); err != nil || c.Value != qbitSID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	category := r.FormValue("category")

	files := r.MultipartForm.File["torrents"]
	if len(files) == 0 {
		q.record(Push{Error: "no torrents field in request", Category: category, At: time.Now()})
		http.Error(w, "no torrents", http.StatusUnsupportedMediaType)
		return
	}

	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		name, ok := torrentName(data)
		if !ok {
			q.record(Push{Error: "unparseable torrent", Category: category, At: time.Now()})
			http.Error(w, "torrent file not valid", http.StatusUnsupportedMediaType)
			return
		}

		q.record(Push{Name: name, Category: category, Approved: true, At: time.Now()})
	}

	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	_, _ = w.Write([]byte("Ok."))
}

// trackers reports a working tracker, which ends autobrr's reannounce loop on its first check.
func (q *Qbittorrent) trackers(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]map[string]any{
		{"url": "** [DHT] **", "status": 0, "msg": ""},
		{"url": q.server.URL + "/announce", "status": 2, "msg": ""},
	})
}

// Arr mocks the Sonarr/Radarr v3 endpoints autobrr pushes releases to. Like a
// real arr it fetches the pushed download url, so a broken url shows up as an
// error rather than a silent approval.
type Arr struct {
	recorder

	APIKey string

	name   string
	server *httptest.Server
	client *http.Client
}

// StartArr serves a mock arr named name until the test ends.
func StartArr(t testing.TB, name string) *Arr {
	t.Helper()

	a := &Arr{
		APIKey: name + "-simulation-apikey",
		name:   name,
		client: &http.Client{Timeout: 10 * time.Second},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v3/release/push", a.push)
	mux.HandleFunc("GET /api/v3/system/status", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{"appName": name, "version": "4.0.0"})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		a.recordUnknown(r)
		http.NotFound(w, r)
	})

	a.server = httptest.NewServer(mux)
	t.Cleanup(a.server.Close)

	return a
}

// URL is the address to configure the arr client with.
func (a *Arr) URL() string {
	return a.server.URL
}

type arrPushRequest struct {
	Title       string `json:"title"`
	DownloadURL string `json:"downloadUrl"`
	Indexer     string `json:"indexer"`
}

type arrPushResponse struct {
	Approved   bool     `json:"approved"`
	Rejected   bool     `json:"rejected"`
	Rejections []string `json:"rejections"`
}

func (a *Arr) push(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Api-Key") != a.APIKey {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var req arrPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	p := Push{Name: req.Title, At: time.Now()}

	if err := a.fetchTorrent(req); err != "" {
		p.Error = err
		a.record(p)

		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode([]map[string]string{{"propertyName": "DownloadUrl", "errorMessage": err}})
		return
	}

	resp := arrPushResponse{Approved: true, Rejections: []string{}}

	if strings.HasSuffix(req.Title, "-"+RejectGroup) {
		resp = arrPushResponse{Rejected: true, Rejections: []string{"Unknown " + a.name + " item"}}
	}

	p.Approved = resp.Approved
	a.record(p)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]arrPushResponse{resp})
}

func (a *Arr) fetchTorrent(req arrPushRequest) string {
	if req.DownloadURL == "" {
		return "empty downloadUrl"
	}

	resp, err := a.client.Get(req.DownloadURL)
	if err != nil {
		return "could not fetch downloadUrl: " + err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "downloadUrl returned " + resp.Status
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "could not read downloadUrl: " + err.Error()
	}

	name, ok := torrentName(data)
	if !ok {
		return "downloadUrl did not return a torrent"
	}

	if name != req.Title {
		return "downloadUrl returned torrent " + name
	}

	return ""
}
