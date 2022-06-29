package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseReleaseTags(t *testing.T) {
	type args struct {
		tags []string
	}
	tests := []struct {
		name string
		args args
		want ReleaseTags
	}{
		{name: "test_1", args: args{tags: []string{"CD", "FLAC", "Lossless"}}, want: ReleaseTags{Source: "CD", Audio: []string{"FLAC", "Lossless"}}},
		{name: "test_2", args: args{tags: []string{"MP4", "2160p", "BluRay", "DV"}}, want: ReleaseTags{Source: "BluRay", Resolution: "2160p", Container: "mp4", HDR: []string{"DV"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ParseReleaseTags(tt.args.tags), "ParseReleaseTags(%v)", tt.args.tags)
		})
	}
}

func TestParseReleaseTagString(t *testing.T) {
	type args struct {
		tags string
	}
	tests := []struct {
		name string
		args args
		want ReleaseTags
	}{
		{name: "music_1", args: args{tags: "FLAC / Lossless / Log / 80% / Cue / CD"}, want: ReleaseTags{Audio: []string{"Cue", "FLAC", "Lossless", "Log"}, Source: "CD"}},
		{name: "music_2", args: args{tags: "FLAC Lossless Log 80% Cue CD"}, want: ReleaseTags{Audio: []string{"Cue", "FLAC", "Lossless", "Log"}, Source: "CD"}},
		{name: "music_3", args: args{tags: "FLAC Lossless Log 100% Cue CD"}, want: ReleaseTags{Audio: []string{"Cue", "FLAC", "Lossless", "Log100", "Log"}, Source: "CD"}},
		{name: "music_4", args: args{tags: "FLAC 24bit Lossless Log 100% Cue CD"}, want: ReleaseTags{Audio: []string{"24BIT Lossless", "Cue", "FLAC", "Lossless", "Log100", "Log"}, Source: "CD"}},
		{name: "music_5", args: args{tags: "MP3 320 WEB"}, want: ReleaseTags{Audio: []string{"320", "MP3"}, Source: "WEB"}},
		{name: "movies_1", args: args{tags: "x264 Blu-ray MKV 1080p"}, want: ReleaseTags{Codec: "x264", Source: "BluRay", Resolution: "1080p", Container: "mkv"}},
		{name: "movies_2", args: args{tags: "HEVC HDR Blu-ray mp4 2160p"}, want: ReleaseTags{Codec: "HEVC", Source: "BluRay", Resolution: "2160p", Container: "mp4", HDR: []string{"HDR"}}},
		{name: "movies_3", args: args{tags: "HEVC HDR DV Blu-ray mp4 2160p"}, want: ReleaseTags{Codec: "HEVC", Source: "BluRay", Resolution: "2160p", Container: "mp4", HDR: []string{"HDR", "DV"}}},
		{name: "movies_4", args: args{tags: "H.264, Blu-ray/HD DVD"}, want: ReleaseTags{Codec: "H.264", Source: "BluRay"}},
		{name: "movies_5", args: args{tags: "H.264, Remux"}, want: ReleaseTags{Codec: "H.264", Other: []string{"REMUX"}}},
		{name: "movies_6", args: args{tags: "H.264, DVD"}, want: ReleaseTags{Codec: "H.264", Source: "DVD"}},
		{name: "movies_7", args: args{tags: "H.264, DVD, Freeleech"}, want: ReleaseTags{Codec: "H.264", Source: "DVD", Bonus: []string{"Freeleech"}}},
		{name: "movies_8", args: args{tags: "H.264, DVD, Freeleech!"}, want: ReleaseTags{Codec: "H.264", Source: "DVD", Bonus: []string{"Freeleech"}}},
		{name: "anime_1", args: args{tags: "Web / MKV / h264 / 1080p / AAC 2.0 / Softsubs (SubsPlease) / Episode 22 / Freeleech"}, want: ReleaseTags{Audio: []string{"AAC"}, Channels: "2.0", Source: "WEB", Resolution: "1080p", Container: "mkv", Group: "SubsPlease", Codec: "H.264", Bonus: []string{"Freeleech"}}},
		{name: "anime_2", args: args{tags: "Web | MKV | h264 | 1080p | AAC 2.0 | Softsubs (SubsPlease) | Episode 22 | Freeleech"}, want: ReleaseTags{Audio: []string{"AAC"}, Channels: "2.0", Source: "WEB", Resolution: "1080p", Container: "mkv", Group: "SubsPlease", Codec: "H.264", Bonus: []string{"Freeleech"}}},
		{name: "ln_1", args: args{tags: "Translated (Seven Seas Entertainment) / EPUB"}, want: ReleaseTags{Group: "Seven Seas Entertainment", Container: "epub"}},
		{name: "ln_2", args: args{tags: "Translated (Yen Press) | EPUB"}, want: ReleaseTags{Group: "Yen Press", Container: "epub"}},
		{name: "manga_1", args: args{tags: "Translated (Kodansha Comics) / Digital / Ongoing"}, want: ReleaseTags{Group: "Kodansha Comics"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, ParseReleaseTagString(tt.args.tags), "ParseReleaseTagString(%v)", tt.args.tags)
		})
	}
}
