// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		a     string
		b     string
		equal bool
	}{
		{name: "dots_spaces_case", a: "That.Show.2023.S01E01.1080p.WEB.h264-GRP", b: "that show 2023 S01E01 1080p WEB h264-GRP", equal: true},
		{name: "underscores", a: "ZORZA-Twilight_of_the_Golden_Star-WEB-2026-ENTiTLED", b: "ZORZA-Twilight.of.the.Golden.Star-WEB-2026-ENTiTLED", equal: true},
		{name: "ddp_spelled_dd+", a: "The.Cleaner.S02.1080p.AMZN.WEB-DL.DDP5.1.H.264-TEPES", b: "The Cleaner S02 1080p AMZN WEB-DL DD+ 5.1 H.264-TEPES", equal: true},
		{name: "codec_spelling", a: "Show.S01E01.1080p.WEB.h264-GRP", b: "Show.S01E01.1080p.WEB.H.264-GRP", equal: true},
		{name: "extension", a: "[SubsPlease] Frieren - 05 (1080p) [A1B2C3D4].mkv", b: "[SubsPlease] Frieren - 05 (1080p) [A1B2C3D4]", equal: true},
		{name: "anime_episodes", a: "[SubsPlease] Frieren - 05 (1080p) [A1B2C3D4].mkv", b: "[SubsPlease] Frieren - 06 (1080p) [A1B2C3D4].mkv"},
		{name: "f1_sessions", a: "Formula1.2025.Round04.Bahrain.FP1.SKY.F1TV.WEB-DL.1080p.h264.English-MWR", b: "Formula1.2025.Round04.Bahrain.FP2.SKY.F1TV.WEB-DL.1080p.h264.English-MWR"},
		{name: "hdr10_plus", a: "Movie.2020.2160p.UHD.BluRay.HDR10+.HEVC-GRP", b: "Movie.2020.2160p.UHD.BluRay.HDR10.HEVC-GRP"},
		{name: "web_vs_webdl", a: "Show.S01E01.1080p.WEB.H264-GRP", b: "Show.S01E01.1080p.WEB-DL.H264-GRP"},
		{name: "latin1_bytes", a: "Caf\xe9.Society.S01E01.1080p.WEB.H264-GRP", b: "Caf\xe9.Society.S01E02.1080p.WEB.H264-GRP"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, NameKey(tt.a) == NameKey(tt.b), "%q vs %q", NameKey(tt.a), NameKey(tt.b))
		})
	}
}

func TestTitleKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		a     string
		b     string
		equal bool
	}{
		{name: "colon", a: "Franchise: Example Show", b: "Franchise Example Show", equal: true},
		{name: "colon_no_space", a: "Franchise:Example Show", b: "Franchise Example Show", equal: true},
		{name: "dots", a: "Mr. Robot", b: "Mr.Robot", equal: true},
		{name: "accents", a: "Pokémon", b: "POKEMON", equal: true},
		{name: "hyphen", a: "Spider-Man", b: "Spiderman", equal: true},
		{name: "acronym", a: "S.W.A.T.", b: "SWAT", equal: true},
		{name: "apostrophe", a: "Journey's End", b: "Journeys End", equal: true},
		{name: "ampersand", a: "Law & Order", b: "Law and Order", equal: true},
		{name: "folded_letters", a: "Æon Flux", b: "Aeon Flux", equal: true},
		{name: "spin_off", a: "9-1-1", b: "9-1-1 Lone Star"},
		{name: "sequel_marker", a: "K-On!", b: "K-On!!"},
		{name: "sequel_question", a: "Kaguya-sama wa Kokurasetai", b: "Kaguya-sama wa Kokurasetai?"},
		{name: "wildcard_chars", a: "100% Wolf", b: "100 Little Wolf"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.equal, TitleKey(tt.a) == TitleKey(tt.b), "%q vs %q", TitleKey(tt.a), TitleKey(tt.b))
		})
	}

	assert.Empty(t, TitleKey(""))
	assert.Empty(t, TitleKey("!!"))
}
