// Copyright (c) 2021 - 2026, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build simulation

package simulation

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

// Kind is the ground truth for a generated release. Filters in the scenario
// are written so each kind is caught by a known filter, which lets the harness
// predict every push without reimplementing autobrr's matching.
type Kind string

const (
	KindTVHD     Kind = "tv-hd"
	KindTVSD     Kind = "tv-sd"
	KindMovieUHD Kind = "movie-uhd"
	KindMovieHD  Kind = "movie-hd"
	KindLinuxISO Kind = "linux-iso"
	KindMusic    Kind = "music"
)

// RejectGroup is the release group the mock arrs reject, standing in for an
// arr that does not know the series or movie.
const RejectGroup = "RJCT"

// Shows are the series the TV filters watch.
var Shows = []string{"Sim Show Alpha", "Sim Show Bravo", "Night Lab"}

var (
	movieAdjectives = []string{"Quiet", "Broken", "Hidden", "Last", "Silver", "Distant", "Burning", "Frozen", "Hollow", "Golden"}
	movieNouns      = []string{"Harbor", "Signal", "Orchard", "Engine", "Meridian", "Lantern", "Canyon", "Archive", "Comet", "Garden"}
	groups          = []string{"SIMGRP", "NTb", "FLUX", "HONE", "EDITH"}
)

// Release is one release the simulation announces or publishes in a feed.
type Release struct {
	ID        int
	Name      string
	Kind      Kind
	Indexer   *Indexer
	Rejected  bool
	Freeleech bool
	Size      uint64
	Category  string
}

// GenerateReleases builds n unique releases spread over indexers. The same seed
// always yields the same set, so a failing run can be replayed.
func GenerateReleases(seed uint64, n int, indexers []*Indexer) []*Release {
	rng := rand.New(rand.NewPCG(seed, seed^0x5eed))

	weights := []struct {
		kind   Kind
		weight int
	}{
		{KindTVHD, 30},
		{KindTVSD, 15},
		{KindMovieUHD, 20},
		{KindMovieHD, 10},
		{KindLinuxISO, 15},
		{KindMusic, 10},
	}

	total := 0
	for _, w := range weights {
		total += w.weight
	}

	seen := make(map[string]struct{}, n)
	releases := make([]*Release, 0, n)

	// bounded so a name space too small for n ends with fewer releases instead of spinning
	for id := 1; len(releases) < n && id <= n*50; id++ {
		pick := rng.IntN(total)

		var kind Kind
		for _, w := range weights {
			if pick < w.weight {
				kind = w.kind
				break
			}
			pick -= w.weight
		}

		rls := &Release{
			ID:        id,
			Kind:      kind,
			Indexer:   indexers[rng.IntN(len(indexers))],
			Freeleech: rng.IntN(4) == 0,
			Size:      uint64(rng.IntN(40_000)+500) * 1024 * 1024,
		}

		// random picks reach Freeleech FLAC in about 1 of 80 releases, so the first
		// release always does, keeping the filter covered in every run
		if id == 1 {
			for _, idx := range indexers {
				if idx.Key == "alpha" {
					kind = KindMusic
					rls.Kind, rls.Indexer, rls.Freeleech = kind, idx, true
				}
			}
		}

		group := groups[rng.IntN(len(groups))]

		if kind == KindTVHD || kind == KindMovieUHD {
			rls.Rejected = rng.IntN(4) == 0
			if rls.Rejected {
				group = RejectGroup
			}
		}

		switch kind {
		case KindTVHD, KindTVSD:
			resolution := "1080p"
			if kind == KindTVSD {
				resolution = "720p"
			}

			show := strings.ReplaceAll(Shows[rng.IntN(len(Shows))], " ", ".")
			rls.Name = fmt.Sprintf("%s.S%02dE%02d.%s.WEB-DL.DDP5.1.H.264-%s", show, rng.IntN(30)+1, rng.IntN(24)+1, resolution, group)
			rls.Category = "TV"

		case KindMovieUHD, KindMovieHD:
			resolution, source := "2160p", "UHD.BluRay.x265"
			if kind == KindMovieHD {
				resolution, source = "1080p", "BluRay.x264"
			}

			title := "The." + movieAdjectives[rng.IntN(len(movieAdjectives))] + "." + movieNouns[rng.IntN(len(movieNouns))]
			rls.Name = fmt.Sprintf("%s.%d.%s.%s-%s", title, 2010+rng.IntN(15), resolution, source, group)
			rls.Category = "Movies"

		case KindLinuxISO:
			rls.Name = fmt.Sprintf("Simux.Linux.%d.%02d.%d.amd64.DVD.ISO-SIMOS", 20+rng.IntN(6), 4+6*rng.IntN(2), rng.IntN(1000))
			rls.Category = "Software"

		case KindMusic:
			rls.Name = fmt.Sprintf("Sim_Artist_%d-Sim_Album_%d-WEB-FLAC-%d-%s", rng.IntN(500), rng.IntN(500), 2010+rng.IntN(15), group)
			rls.Category = "Music"
		}

		if _, ok := seen[rls.Name]; ok {
			continue
		}
		seen[rls.Name] = struct{}{}

		releases = append(releases, rls)
	}

	for i, rls := range releases {
		rls.ID = i + 1
	}

	return releases
}
