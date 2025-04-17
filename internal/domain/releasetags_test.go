// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseReleaseTags(t *testing.T) {
	t.Parallel()
	type args struct {
		tags []string
	}
	tests := []struct {
		name string
		args args
		want ReleaseTags
	}{
		{name: "test_1", args: args{tags: []string{"CD", "FLAC", "Lossless"}}, want: ReleaseTags{Audio: []string{"FLAC", "Lossless"}, AudioBitrate: "Lossless", AudioFormat: "FLAC", Source: "CD"}},
		{name: "test_2", args: args{tags: []string{"MP4", "2160p", "BluRay", "DV"}}, want: ReleaseTags{Source: "BluRay", Resolution: "2160p", Container: "mp4", HDR: []string{"DV"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ParseReleaseTags(tt.args.tags), "ParseReleaseTags(%v)", tt.args.tags)
		})
	}
}

func TestParseReleaseTagString(t *testing.T) {
	t.Parallel()
	type args struct {
		tags string
	}
	tests := []struct {
		name string
		args args
		want ReleaseTags
	}{
		{name: "music_1", args: args{tags: "FLAC / Lossless / Log / 80% / Cue / CD"}, want: ReleaseTags{Audio: []string{"Cue", "FLAC", "Lossless", "Log"}, AudioBitrate: "Lossless", AudioFormat: "FLAC", Source: "CD", HasCue: true}},
		{name: "music_2", args: args{tags: "FLAC Lossless Log 80% Cue CD"}, want: ReleaseTags{Audio: []string{"Cue", "FLAC", "Lossless", "Log80", "Log"}, AudioBitrate: "Lossless", AudioFormat: "FLAC", Source: "CD", HasLog: true, LogScore: 80, HasCue: true}},
		{name: "music_3", args: args{tags: "FLAC Lossless Log 100% Cue CD"}, want: ReleaseTags{Audio: []string{"Cue", "FLAC", "Lossless", "Log100", "Log"}, AudioBitrate: "Lossless", AudioFormat: "FLAC", Source: "CD", HasLog: true, LogScore: 100, HasCue: true}},
		{name: "music_4", args: args{tags: "FLAC 24bit Lossless Log 100% Cue CD"}, want: ReleaseTags{Audio: []string{"24BIT Lossless", "Cue", "FLAC", "Log100", "Log"}, AudioBitrate: "24BIT Lossless", AudioFormat: "FLAC", Source: "CD", HasLog: true, LogScore: 100, HasCue: true}},
		{name: "music_5", args: args{tags: "MP3 320 WEB"}, want: ReleaseTags{Audio: []string{"320", "MP3"}, AudioBitrate: "320", AudioFormat: "MP3", Source: "WEB"}},
		{name: "music_6", args: args{tags: "FLAC Lossless Log (100%) Cue CD"}, want: ReleaseTags{Audio: []string{"Cue", "FLAC", "Lossless", "Log100", "Log"}, AudioBitrate: "Lossless", AudioFormat: "FLAC", Source: "CD", HasCue: true, HasLog: true, LogScore: 100}},
		{name: "music_7", args: args{tags: "DSD / DSD64 / WEB"}, want: ReleaseTags{Audio: []string{"DSD", "DSD64"}, AudioBitrate: "DSD64", AudioFormat: "DSD", Source: "WEB"}},
		{name: "music_8", args: args{tags: "DSD / DSD128 / WEB"}, want: ReleaseTags{Audio: []string{"128", "DSD", "DSD128"}, AudioBitrate: "DSD128", AudioFormat: "DSD", Source: "WEB"}},
		{name: "music_9", args: args{tags: "DSD / DSD256 / WEB"}, want: ReleaseTags{Audio: []string{"256", "DSD", "DSD256"}, AudioBitrate: "DSD256", AudioFormat: "DSD", Source: "WEB"}},
		{name: "music_10", args: args{tags: "DSD / DSD512 / WEB"}, want: ReleaseTags{Audio: []string{"DSD", "DSD512"}, AudioBitrate: "DSD512", AudioFormat: "DSD", Source: "WEB"}},
		{name: "movies_1", args: args{tags: "x264 Blu-ray MKV 1080p"}, want: ReleaseTags{Codec: "x264", Source: "BluRay", Resolution: "1080p", Container: "mkv"}},
		{name: "movies_2", args: args{tags: "HEVC HDR Blu-ray mp4 2160p"}, want: ReleaseTags{Codec: "HEVC", Source: "BluRay", Resolution: "2160p", Container: "mp4", HDR: []string{"HDR"}}},
		{name: "movies_3", args: args{tags: "HEVC HDR DV Blu-ray mp4 2160p"}, want: ReleaseTags{Codec: "HEVC", Source: "BluRay", Resolution: "2160p", Container: "mp4", HDR: []string{"HDR", "DV"}}},
		{name: "movies_4", args: args{tags: "H.264, Blu-ray/HD DVD"}, want: ReleaseTags{Codec: "H.264", Source: "BluRay"}},
		{name: "movies_5", args: args{tags: "H.264, Remux"}, want: ReleaseTags{Codec: "H.264", Other: []string{"REMUX"}}},
		{name: "movies_6", args: args{tags: "H.264, DVD"}, want: ReleaseTags{Codec: "H.264", Source: "DVD"}},
		{name: "movies_7", args: args{tags: "H.264, DVD, Freeleech"}, want: ReleaseTags{Codec: "H.264", Source: "DVD", Bonus: []string{"Freeleech"}}},
		{name: "movies_8", args: args{tags: "H.264, DVD, Freeleech!"}, want: ReleaseTags{Codec: "H.264", Source: "DVD", Bonus: []string{"Freeleech"}}},
		{name: "anime_1", args: args{tags: "Web / MKV / h264 / 1080p / AAC 2.0 / Softsubs (SubsPlease) / Episode 22 / Freeleech"}, want: ReleaseTags{Audio: []string{"AAC"}, AudioBitrate: "", AudioFormat: "AAC", Bonus: []string{"Freeleech"}, Channels: "2.0", Codec: "H.264", Container: "mkv", Resolution: "1080p", Source: "WEB"}},
		{name: "anime_2", args: args{tags: "Web | ISO | h264 | 1080p | AAC 2.0 | Softsubs (SubsPlease) | Episode 22 | Freeleech"}, want: ReleaseTags{Audio: []string{"AAC"}, AudioBitrate: "", AudioFormat: "AAC", Bonus: []string{"Freeleech"}, Channels: "2.0", Codec: "H.264", Container: "iso", Resolution: "1080p", Source: "WEB"}}, {name: "tv_1", args: args{tags: "MKV | H.264 | WEB-DL | 1080p | Internal | FastTorrent"}, want: ReleaseTags{Source: "WEB-DL", Codec: "H.264", Resolution: "1080p", Container: "mkv", Origin: "Internal"}},
		{name: "tv_2", args: args{tags: "MKV | H.264 | WEB-DL | 1080p | Scene | FastTorrent"}, want: ReleaseTags{Source: "WEB-DL", Codec: "H.264", Resolution: "1080p", Container: "mkv", Origin: "Scene"}},
		{name: "tv_3", args: args{tags: "MKV | H.264 | WEB-DL | 1080p | P2P | FastTorrent"}, want: ReleaseTags{Source: "WEB-DL", Codec: "H.264", Resolution: "1080p", Container: "mkv", Origin: "P2P"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ParseReleaseTagString(tt.args.tags), "ParseReleaseTagString(%v)", tt.args.tags)
		})
	}
}

func Test_cleanReleaseTags(t *testing.T) {
	t.Parallel()
	type args struct {
		tagString string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "1", args: args{tagString: "FLAC / Lossless / Log / 100% / Cue / CD"}, want: "FLAC Lossless Log 100% Cue CD"},
		{name: "2", args: args{tagString: "FLAC/Lossless/Log 100%/Cue/CD"}, want: "FLAC Lossless Log 100% Cue CD"},
		{name: "3", args: args{tagString: "FLAC | Lossless | Log | 100% | Cue | CD"}, want: "FLAC Lossless Log 100% Cue CD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, CleanReleaseTags(tt.args.tagString), "cleanReleaseTags(%v)", tt.args.tagString)
		})
	}
}
