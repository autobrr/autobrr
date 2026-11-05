// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/test/irc/ircd"
	"github.com/autobrr/autobrr/test/mockdiscord"
)

// IndexerKind is how an indexer delivers releases to autobrr.
type IndexerKind string

const (
	IndexerKindIRC     IndexerKind = "irc"
	IndexerKindRSS     IndexerKind = "rss"
	IndexerKindTorznab IndexerKind = "torznab"
)

// Indexer is one mock indexer and the autobrr records created for it.
type Indexer struct {
	Key     string
	Name    string
	Kind    IndexerKind
	Passkey string

	Tracker *Tracker

	IRC       *ircd.Server
	Channel   string
	Announcer string

	ID         int64
	Identifier string
	FeedID     int
}

// Client names a mock download client a filter action targets.
type Client string

const (
	ClientQbittorrent Client = "qbittorrent"
	ClientSonarr      Client = "sonarr"
	ClientRadarr      Client = "radarr"
)

func (c Client) isArr() bool {
	return c == ClientSonarr || c == ClientRadarr
}

// FilterSpec is a filter the scenario creates in autobrr, plus the ground-truth
// predicate the harness uses to predict what that filter will catch.
type FilterSpec struct {
	Name     string
	Priority int32
	Indexers []string
	Client   Client
	Category string

	Matches   func(rls *Release) bool
	Configure func(f map[string]any)

	ID int
}

// ActionName is the name of the filter's single action, as it shows up in notifications.
func (f *FilterSpec) ActionName() string {
	return f.Name + " -> " + string(f.Client)
}

// Scenario owns every mock the simulation runs against autobrr.
type Scenario struct {
	Indexers []*Indexer
	Filters  []*FilterSpec

	Qbittorrent *Qbittorrent
	Sonarr      *Arr
	Radarr      *Arr
	Discord     *mockdiscord.Server

	discordServer  *httptest.Server
	definitionsDir string
	clientIDs      map[Client]int32
}

func isKind(kinds ...Kind) func(rls *Release) bool {
	return func(rls *Release) bool {
		return slices.Contains(kinds, rls.Kind)
	}
}

// NewScenario starts the mocks and writes the custom definitions autobrr loads
// for the IRC indexers. It does not touch autobrr, see Configure.
func NewScenario(t testing.TB) *Scenario {
	t.Helper()

	s := &Scenario{
		definitionsDir: t.TempDir(),
		clientIDs:      map[Client]int32{},
		Indexers: []*Indexer{
			{Key: "alpha", Name: "SimAlpha", Kind: IndexerKindIRC, Channel: "#alpha-announce", Announcer: "AlphaBot"},
			{Key: "bravo", Name: "SimBravo", Kind: IndexerKindIRC, Channel: "#bravo-announce", Announcer: "BravoBot"},
			{Key: "charlie", Name: "SimCharlie RSS", Kind: IndexerKindRSS},
			{Key: "delta", Name: "SimDelta Torznab", Kind: IndexerKindTorznab},
		},
	}

	s.Filters = []*FilterSpec{
		{
			Name:     "TV HD",
			Priority: 40,
			Indexers: []string{"alpha", "bravo", "charlie", "delta"},
			Client:   ClientSonarr,
			Matches:  isKind(KindTVHD),
			Configure: func(f map[string]any) {
				f["shows"] = strings.Join(Shows, ",")
				f["resolutions"] = []string{"1080p"}
			},
		},
		{
			Name:     "Movies UHD",
			Priority: 30,
			Indexers: []string{"alpha", "bravo", "charlie", "delta"},
			Client:   ClientRadarr,
			Matches:  isKind(KindMovieUHD),
			Configure: func(f map[string]any) {
				f["resolutions"] = []string{"2160p"}
				f["years"] = "2000-2030"
			},
		},
		{
			// catches what Sonarr rejected from TV HD, as well as SD episodes
			Name:     "TV Fallback",
			Priority: 20,
			Indexers: []string{"alpha", "bravo", "charlie", "delta"},
			Client:   ClientQbittorrent,
			Category: "tv-fallback",
			Matches:  isKind(KindTVHD, KindTVSD),
			Configure: func(f map[string]any) {
				f["shows"] = strings.Join(Shows, ",")
			},
		},
		{
			// scoped to the IRC indexers, so ISOs from the feeds must not match
			Name:     "Linux ISOs",
			Priority: 10,
			Indexers: []string{"alpha", "bravo"},
			Client:   ClientQbittorrent,
			Category: "isos",
			Matches:  isKind(KindLinuxISO),
			Configure: func(f map[string]any) {
				f["match_releases"] = "*Linux*ISO*"
			},
		},
		{
			// RSS carries no freeleech flag, so only IRC and Torznab can feed this one
			Name:     "Freeleech FLAC",
			Priority: 5,
			Indexers: []string{"alpha", "delta"},
			Client:   ClientQbittorrent,
			Category: "music-fl",
			Matches: func(rls *Release) bool {
				return rls.Kind == KindMusic && rls.Freeleech
			},
			Configure: func(f map[string]any) {
				f["match_releases"] = "*FLAC*"
				f["freeleech"] = true
			},
		},
	}

	for _, idx := range s.Indexers {
		idx.Passkey = idx.Key + "simulationpasskey"
		idx.Tracker = StartTracker(t, idx)

		if idx.Kind != IndexerKindIRC {
			continue
		}

		idx.IRC = ircd.New(t)
		idx.IRC.AddChannel(idx.Channel, ircd.Announcer(idx.Announcer))
		idx.Identifier = "sim" + idx.Key

		def := fmt.Sprintf(definitionTemplate, idx.Name, idx.Identifier, idx.Name, idx.Tracker.URL(), "Sim"+idx.Key, idx.IRC.Port(), idx.Channel, idx.Announcer)

		if err := os.WriteFile(filepath.Join(s.definitionsDir, idx.Identifier+".yaml"), []byte(def), 0o644); err != nil {
			t.Fatalf("simulation: could not write definition: %v", err)
		}
	}

	s.Qbittorrent = StartQbittorrent(t)
	s.Sonarr = StartArr(t, "sonarr")
	s.Radarr = StartArr(t, "radarr")

	s.Discord = &mockdiscord.Server{}
	s.discordServer = httptest.NewServer(s.Discord.Handler())
	t.Cleanup(s.discordServer.Close)

	return s
}

// DefinitionsDir holds the generated IRC indexer definitions.
func (s *Scenario) DefinitionsDir() string {
	return s.definitionsDir
}

func (s *Scenario) indexer(key string) *Indexer {
	for _, idx := range s.Indexers {
		if idx.Key == key {
			return idx
		}
	}

	return nil
}

// Configure creates the indexers, IRC networks, feeds, download clients,
// filters and notification in autobrr over its API, the way the UI does.
//
// Request bodies are plain maps: the domain types redact their secrets in
// MarshalJSON, so encoding one would send "<redacted>" as the passkey.
func (s *Scenario) Configure(app *App) {
	for _, idx := range s.Indexers {
		s.configureIndexer(app, idx)
	}

	s.configureDownloader(app, ClientQbittorrent, map[string]any{
		"name":     "Sim qBittorrent",
		"type":     domain.DownloaderTypeQbittorrent,
		"enabled":  true,
		"host":     s.Qbittorrent.URL(),
		"username": qbitUsername,
		"password": qbitPassword,
	})

	s.configureDownloader(app, ClientSonarr, map[string]any{
		"name":     "Sim Sonarr",
		"type":     domain.DownloaderTypeSonarr,
		"enabled":  true,
		"host":     s.Sonarr.URL(),
		"settings": map[string]any{"apikey": s.Sonarr.APIKey},
	})

	s.configureDownloader(app, ClientRadarr, map[string]any{
		"name":     "Sim Radarr",
		"type":     domain.DownloaderTypeRadarr,
		"enabled":  true,
		"host":     s.Radarr.URL(),
		"settings": map[string]any{"apikey": s.Radarr.APIKey},
	})

	for _, spec := range s.Filters {
		s.configureFilter(app, spec)
	}

	app.Post("/api/notification", map[string]any{
		"name":    "Sim Discord",
		"type":    domain.NotificationTypeDiscord,
		"enabled": true,
		"webhook": s.discordServer.URL + "/api/webhooks/1/simulation-token",
		"events": []domain.NotificationEvent{
			domain.NotificationEventPushApproved,
			domain.NotificationEventPushRejected,
			domain.NotificationEventPushError,
		},
	}, nil)
}

func (s *Scenario) configureIndexer(app *App, idx *Indexer) {
	var created domain.Indexer

	switch idx.Kind {
	case IndexerKindIRC:
		app.Post("/api/indexer", map[string]any{
			"name":           idx.Name,
			"identifier":     idx.Identifier,
			"implementation": domain.IndexerImplementationIRC,
			"enabled":        true,
			"settings":       map[string]string{"passkey": idx.Passkey},
		}, &created)

		app.Post("/api/irc", map[string]any{
			"name":     "Sim" + idx.Key,
			"enabled":  true,
			"server":   idx.IRC.Host(),
			"port":     idx.IRC.Port(),
			"nick":     "autobrr-sim-" + idx.Key,
			"channels": []map[string]any{{"name": idx.Channel, "enabled": true}},
		}, nil)

	case IndexerKindRSS, IndexerKindTorznab:
		identifier, implementation, feedType, feedURL, apiKey := "rss", domain.IndexerImplementationRSS, domain.FeedTypeRSS, idx.Tracker.URL()+"rss", ""
		if idx.Kind == IndexerKindTorznab {
			identifier, implementation, feedType, feedURL, apiKey = "torznab", domain.IndexerImplementationTorznab, domain.FeedTypeTorznab, idx.Tracker.URL()+"api", idx.Passkey
		}

		app.Post("/api/indexer", map[string]any{
			"name":           idx.Name,
			"identifier":     identifier,
			"implementation": implementation,
			"enabled":        true,
			"base_url":       idx.Tracker.URL(),
		}, &created)

		// left disabled so the feed schedule can not fetch items ahead of the forced
		// batches Inject runs, forcerun does not check enabled
		app.Post("/api/feeds", map[string]any{
			"name":       idx.Name,
			"type":       feedType,
			"enabled":    false,
			"url":        feedURL,
			"api_key":    apiKey,
			"interval":   30,
			"timeout":    60,
			"indexer_id": created.ID,
		}, nil)

		var feeds []domain.Feed
		app.Get("/api/feeds", &feeds)

		for _, f := range feeds {
			if f.Indexer.ID == int(created.ID) {
				idx.FeedID = f.ID
			}
		}

		if idx.FeedID == 0 {
			app.t.Fatalf("simulation: could not find feed for indexer %s", idx.Name)
		}
	}

	idx.ID = created.ID
	idx.Identifier = created.Identifier
}

func (s *Scenario) configureDownloader(app *App, client Client, d map[string]any) {
	app.Post("/api/downloaders", d, nil)

	var all []domain.Downloader
	app.Get("/api/downloaders", &all)

	for _, c := range all {
		if c.Name == d["name"] {
			s.clientIDs[client] = c.ID
		}
	}

	if s.clientIDs[client] == 0 {
		app.t.Fatalf("simulation: could not find downloader %s after creating it", d["name"])
	}
}

func (s *Scenario) configureFilter(app *App, spec *FilterSpec) {
	indexers := make([]map[string]any, 0, len(spec.Indexers))
	for _, key := range spec.Indexers {
		idx := s.indexer(key)
		indexers = append(indexers, map[string]any{"id": idx.ID, "name": idx.Name, "identifier": idx.Identifier})
	}

	// the same create payload the add form sends, the rest goes in with the update below
	var created domain.Filter
	app.Post("/api/filters", map[string]any{
		"name":        spec.Name,
		"enabled":     false,
		"resolutions": []string{},
		"codecs":      []string{},
		"sources":     []string{},
		"containers":  []string{},
		"origins":     []string{},
	}, &created)

	if created.ID == 0 {
		app.t.Fatalf("simulation: filter %s was created without an id", spec.Name)
	}

	// round-trip the raw json like the UI does, decoding into domain.Filter would
	// drop the empty lists its omitempty tags hide and store them as NULL
	var filter map[string]any
	app.Get(fmt.Sprintf("/api/filters/%d", created.ID), &filter)

	fillEmptyLists(filter)

	filter["enabled"] = true
	filter["priority"] = spec.Priority
	filter["indexers"] = indexers
	filter["actions"] = []*domain.Action{
		{
			Name:           spec.ActionName(),
			Type:           domain.ActionType(strings.ToUpper(string(spec.Client))),
			Enabled:        true,
			ClientID:       s.clientIDs[spec.Client],
			Category:       spec.Category,
			ReAnnounceSkip: true,
		},
	}

	spec.Configure(filter)

	app.Put(fmt.Sprintf("/api/filters/%d", created.ID), filter, nil)

	spec.ID = created.ID
}

// fillEmptyLists sets every string list the filter json left out to an empty
// list, as the UI form does, since the update stores a missing list as NULL.
func fillEmptyLists(filter map[string]any) {
	typ := reflect.TypeFor[domain.Filter]()

	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Type != reflect.TypeFor[[]string]() {
			continue
		}

		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			continue
		}

		if filter[name] == nil {
			filter[name] = []string{}
		}
	}
}

// WaitReady blocks until autobrr monitors every announce channel.
func (s *Scenario) WaitReady(t testing.TB, app *App, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)

	for {
		var networks []domain.IrcNetworkWithHealth
		app.Get("/api/irc", &networks)

		pending := []string{}

		for _, idx := range s.Indexers {
			if idx.Kind != IndexerKindIRC {
				continue
			}

			monitoring := false
			for _, n := range networks {
				for _, ch := range n.Channels {
					if strings.EqualFold(ch.Name, idx.Channel) && ch.Monitoring {
						monitoring = true
					}
				}
			}

			if !monitoring {
				pending = append(pending, idx.Channel)
			}
		}

		if len(pending) == 0 {
			return
		}

		if time.Now().After(deadline) {
			t.Fatalf("simulation: autobrr never monitored %v\n%s", pending, app.LogTail(80))
		}

		time.Sleep(250 * time.Millisecond)
	}
}

// Inject delivers releases in order: IRC releases are announced as they come,
// feed releases are published and the feed force-run every batch items, so feed
// latency is bounded by the batch size rather than the feed interval.
func (s *Scenario) Inject(app *App, releases []*Release, interval time.Duration, batch int) map[int]time.Time {
	injected := make(map[int]time.Time, len(releases))
	pending := map[*Indexer][]int{}

	flush := func(idx *Indexer) {
		if len(pending[idx]) == 0 {
			return
		}

		now := time.Now()
		for _, id := range pending[idx] {
			injected[id] = now
		}

		app.Post(fmt.Sprintf("/api/feeds/%d/forcerun", idx.FeedID), nil, nil)
		pending[idx] = nil
	}

	for _, rls := range releases {
		idx := rls.Indexer

		switch idx.Kind {
		case IndexerKindIRC:
			idx.Tracker.Add(rls)
			injected[rls.ID] = time.Now()
			idx.IRC.Announce(idx.Channel, idx.Announcer, announceLine(idx, rls))

		default:
			idx.Tracker.Publish(rls)
			pending[idx] = append(pending[idx], rls.ID)

			if len(pending[idx]) >= batch {
				flush(idx)
			}
		}

		time.Sleep(interval)
	}

	for _, idx := range s.Indexers {
		flush(idx)
	}

	return injected
}

func announceLine(idx *Indexer, rls *Release) string {
	freeleech := "no"
	if rls.Freeleech {
		freeleech = "yes"
	}

	return fmt.Sprintf("New Torrent: %s | Category: %s | Size: %d MiB | Freeleech: %s | %storrent/%d",
		rls.Name, rls.Category, rls.Size/(1024*1024), freeleech, idx.Tracker.URL(), rls.ID)
}

// definitionTemplate is filled positionally with fmt rather than text/template,
// because the match section carries autobrr's own template syntax.
const definitionTemplate = `---
version: 2
name: %s
identifier: %s
description: Simulation indexer %s.
language: en-us
urls:
  - %s
privacy: private
protocol: torrent
supports:
  - irc
source: custom
settings:
  - name: passkey
    type: secret
    required: true
    label: Passkey

irc:
  network: %s
  server: 127.0.0.1
  port: %d
  tls: false
  settings:
    - name: nick
      type: text
      required: true
      label: Nick

  channels:
    - name: "%s"
      announcers:
        - %s
      parse:
        type: single
        lines:
          - pattern: 'New Torrent: (?P<releaseName>.+?) \| Category: (?P<category>.+?) \| Size: (?P<torrentSize>.+?) \| Freeleech: (?P<freeleech>\w+) \| (?P<baseUrl>https?://[^/]+/)torrent/(?P<torrentId>\d+)'
        match:
          infourl: "/torrent/{{ .torrentId }}"
          downloadurl: "/download/{{ .torrentId }}?passkey={{ .passkey }}"
`
