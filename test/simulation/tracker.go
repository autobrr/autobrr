// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tracker is the HTTP side of one mock indexer: it serves .torrent files to
// autobrr and the arrs, and the RSS or Torznab feed for feed indexers.
type Tracker struct {
	indexer *Indexer
	server  *httptest.Server

	m         sync.Mutex
	releases  map[int]*Release
	published []*Release
	downloads map[int]int
	denied    []string
}

// StartTracker serves indexer's releases until the test ends.
func StartTracker(t testing.TB, indexer *Indexer) *Tracker {
	t.Helper()

	tr := &Tracker{
		indexer:   indexer,
		releases:  map[int]*Release{},
		downloads: map[int]int{},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /download/{id}", tr.download)
	mux.HandleFunc("GET /rss", tr.rss)
	mux.HandleFunc("GET /api", tr.torznab)

	tr.server = httptest.NewServer(mux)
	t.Cleanup(tr.server.Close)

	return tr
}

// URL is the tracker's base url, with a trailing slash like announce lines carry.
func (tr *Tracker) URL() string {
	return tr.server.URL + "/"
}

// Add registers rls so its torrent can be downloaded.
func (tr *Tracker) Add(rls *Release) {
	tr.m.Lock()
	defer tr.m.Unlock()

	tr.releases[rls.ID] = rls
}

// Publish adds rls to the feed served on the next fetch.
func (tr *Tracker) Publish(rls *Release) {
	tr.m.Lock()
	defer tr.m.Unlock()

	tr.releases[rls.ID] = rls
	tr.published = append(tr.published, rls)
}

// DownloadURL is where rls can be fetched with the indexer's passkey.
func (tr *Tracker) DownloadURL(rls *Release) string {
	return fmt.Sprintf("%sdownload/%d?passkey=%s", tr.URL(), rls.ID, tr.indexer.Passkey)
}

// Downloads returns how many times each release id was fetched.
func (tr *Tracker) Downloads() map[int]int {
	tr.m.Lock()
	defer tr.m.Unlock()

	out := make(map[int]int, len(tr.downloads))
	for k, v := range tr.downloads {
		out[k] = v
	}

	return out
}

// Denied lists requests rejected for a bad passkey or unknown id.
func (tr *Tracker) Denied() []string {
	tr.m.Lock()
	defer tr.m.Unlock()

	return slices.Clone(tr.denied)
}

func (tr *Tracker) download(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.PathValue("id"))

	tr.m.Lock()
	rls, ok := tr.releases[id]
	passkeyOK := r.URL.Query().Get("passkey") == tr.indexer.Passkey
	if !ok || !passkeyOK {
		tr.denied = append(tr.denied, r.URL.String())
	} else {
		tr.downloads[id]++
	}
	tr.m.Unlock()

	if !passkeyOK {
		http.Error(w, "invalid passkey", http.StatusForbidden)
		return
	}

	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/x-bittorrent")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", rls.Name+".torrent"))

	_, _ = w.Write(buildTorrent(rls.Name, tr.URL()+"announce"))
}

// feedItems returns the published releases newest first, like a tracker feed.
func (tr *Tracker) feedItems() []*Release {
	tr.m.Lock()
	defer tr.m.Unlock()

	items := slices.Clone(tr.published)
	slices.Reverse(items)

	return items
}

func (tr *Tracker) rss(w http.ResponseWriter, _ *http.Request) {
	now := time.Now().UTC()

	var b strings.Builder

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<rss version="2.0"><channel>`)
	fmt.Fprintf(&b, "<title>%s</title><link>%s</link>", xmlEscape(tr.indexer.Name), tr.URL())

	for _, rls := range tr.feedItems() {
		link := xmlEscape(tr.DownloadURL(rls))

		b.WriteString("<item>")
		fmt.Fprintf(&b, "<title>%s</title>", xmlEscape(rls.Name))
		fmt.Fprintf(&b, `<guid isPermaLink="false">sim-%s-%d</guid>`, tr.indexer.Key, rls.ID)
		fmt.Fprintf(&b, "<link>%s</link>", link)
		fmt.Fprintf(&b, "<pubDate>%s</pubDate>", now.Format(time.RFC1123Z))
		fmt.Fprintf(&b, "<category>%s</category>", rls.Category)
		fmt.Fprintf(&b, `<enclosure url="%s" length="%d" type="application/x-bittorrent"/>`, link, rls.Size)
		b.WriteString("</item>")
	}

	b.WriteString("</channel></rss>")

	w.Header().Set("Content-Type", "application/rss+xml")
	_, _ = w.Write([]byte(b.String()))
}

var torznabCategories = map[string]int{
	"Movies":   2000,
	"Music":    3000,
	"Software": 4000,
	"TV":       5000,
}

func (tr *Tracker) torznab(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("apikey") != tr.indexer.Passkey {
		tr.m.Lock()
		tr.denied = append(tr.denied, r.URL.String())
		tr.m.Unlock()

		http.Error(w, `<error code="100" description="Invalid API Key"/>`, http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/xml")

	if r.URL.Query().Get("t") == "caps" {
		_, _ = w.Write([]byte(torznabCaps))
		return
	}

	now := time.Now().UTC()

	var b strings.Builder

	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<rss version="2.0" xmlns:torznab="http://torznab.com/schemas/2015/feed"><channel>`)
	fmt.Fprintf(&b, "<title>%s</title>", xmlEscape(tr.indexer.Name))

	items := tr.feedItems()

	fmt.Fprintf(&b, `<torznab:response offset="0" total="%d"/>`, len(items))

	for _, rls := range items {
		link := tr.DownloadURL(rls)

		volumeFactor := "1"
		if rls.Freeleech {
			volumeFactor = "0"
		}

		b.WriteString("<item>")
		fmt.Fprintf(&b, "<title>%s</title>", xmlEscape(rls.Name))
		fmt.Fprintf(&b, "<guid>sim-%s-%d</guid>", tr.indexer.Key, rls.ID)
		fmt.Fprintf(&b, "<comments>%storrent/%d</comments>", tr.URL(), rls.ID)
		fmt.Fprintf(&b, "<pubDate>%s</pubDate>", now.Format(time.RFC1123Z))
		fmt.Fprintf(&b, "<size>%d</size>", rls.Size)
		fmt.Fprintf(&b, "<link>%s</link>", xmlEscape(link))
		fmt.Fprintf(&b, "<category>%d</category>", torznabCategories[rls.Category])
		fmt.Fprintf(&b, `<enclosure url="%s" length="%d" type="application/x-bittorrent"/>`, xmlEscape(link), rls.Size)
		fmt.Fprintf(&b, `<torznab:attr name="category" value="%d"/>`, torznabCategories[rls.Category])
		fmt.Fprintf(&b, `<torznab:attr name="seeders" value="%d"/>`, 10)
		fmt.Fprintf(&b, `<torznab:attr name="downloadvolumefactor" value="%s"/>`, volumeFactor)
		b.WriteString(`<torznab:attr name="uploadvolumefactor" value="1"/>`)
		b.WriteString("</item>")
	}

	b.WriteString("</channel></rss>")

	_, _ = w.Write([]byte(b.String()))
}

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))

	return b.String()
}

const torznabCaps = `<?xml version="1.0" encoding="UTF-8"?>
<caps>
  <server version="1.1" title="Simulation"/>
  <limits max="100" default="100"/>
  <searching>
    <search available="yes" supportedParams="q"/>
    <tv-search available="yes" supportedParams="q,season,ep"/>
    <movie-search available="yes" supportedParams="q"/>
  </searching>
  <categories>
    <category id="2000" name="Movies"/>
    <category id="3000" name="Audio"/>
    <category id="4000" name="PC"/>
    <category id="5000" name="TV"/>
  </categories>
</caps>`
