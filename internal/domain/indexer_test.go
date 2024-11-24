// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexerIRCParseMatch_ParseUrls(t *testing.T) {
	t.Parallel()
	type fields struct {
		TorrentURL  string
		TorrentName string
		InfoURL     string
		Encode      []string
	}
	type args struct {
		baseURL string
		vars    map[string]string
		rls     *Release
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Release
	}{
		{
			name: "",
			fields: fields{
				TorrentURL: "rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .torrentName }}.torrent",
				Encode:     []string{"torrentName"},
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"category":    "TV :: Episodes HD",
					"torrentName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
					"uploader":    "Anonymous",
					"freeleech":   "",
					"baseUrl":     "https://mock.local/",
					"torrentId":   "240860011",
					"rsskey":      "00000000000000000000",
				},
				rls: &Release{},
			},
			want: &Release{
				DownloadURL: "https://mock.local/rss/download/240860011/00000000000000000000/The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP.torrent",
			},
		},
		{
			name: "",
			fields: fields{
				TorrentURL: "/torrent/{{ .torrentId }}/download/{{ .passkey }}",
				Encode:     nil,
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"torrentName":    "Great BluRay SoftSubbed Anime",
					"category":       "TV Series",
					"year":           "2020",
					"releaseTags":    "Blu-ray / MKV / h264 10-bit / 1080p / FLAC 2.0 / Dual Audio / Softsubs (Sub Group) / Freeleech",
					"releaseGroup":   "Softsubs",
					"releaseEpisode": "",
					"freeleech":      "freeleech",
					"baseUrl":        "https://mock.local",
					"torrentId":      "240860011",
					"tags":           "comedy, drama, school.life, sports",
					"uploader":       "Uploader",
					"passkey":        "00000000000000000000",
				},
				rls: &Release{},
			},
			want: &Release{
				DownloadURL: "https://mock.local/torrent/240860011/download/00000000000000000000",
			},
		},
		{
			name: "",
			fields: fields{
				TorrentURL: "{{ .baseUrl }}rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .torrentName }}.torrent",
				Encode:     []string{"torrentName"},
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"category":    "TV :: Episodes HD",
					"torrentName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
					"uploader":    "Anonymous",
					"freeleech":   "",
					"baseUrl":     "https://mock.local/",
					"torrentId":   "240860011",
					"rsskey":      "00000000000000000000",
				},
				rls: &Release{},
			},
			want: &Release{
				DownloadURL: "https://mock.local/rss/download/240860011/00000000000000000000/The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP.torrent",
			},
		},
		{
			name: "",
			fields: fields{
				TorrentURL: "https://mock.local/rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .torrentName }}.torrent",
				Encode:     []string{"torrentName"},
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"category":    "TV :: Episodes HD",
					"torrentName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
					"uploader":    "Anonymous",
					"freeleech":   "",
					"baseUrl":     "https://mock.local/",
					"torrentId":   "240860011",
					"rsskey":      "00000000000000000000",
				},
				rls: &Release{},
			},
			want: &Release{
				DownloadURL: "https://mock.local/rss/download/240860011/00000000000000000000/The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP.torrent",
			},
		},
		{
			name: "",
			fields: fields{
				TorrentURL: "/rss/?action=download&key={{ .key }}&token={{ .token }}&hash={{ .torrentId }}&title={{ .torrentName }}",
				Encode:     []string{"torrentName"},
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"category":    "Movies/Remux",
					"torrentName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
					"uploader":    "Anonymous",
					"torrentSize": "",
					"baseUrl":     "https://mock.local/",
					"torrentId":   "240860011",
					"key":         "KEY",
					"token":       "TOKEN",
					"rsskey":      "00000000000000000000",
				},
				rls: &Release{},
			},
			want: &Release{
				DownloadURL: "https://mock.local/rss/?action=download&key=KEY&token=TOKEN&hash=240860011&title=The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &IndexerIRCParseMatch{
				TorrentURL:  tt.fields.TorrentURL,
				TorrentName: tt.fields.TorrentName,
				InfoURL:     tt.fields.InfoURL,
				Encode:      tt.fields.Encode,
			}
			p.ParseURLs(tt.args.baseURL, tt.args.vars, tt.args.rls)
			assert.Equal(t, tt.want, tt.args.rls)
		})
	}
}

func TestIndexerIRCParseMatch_ParseTorrentName(t *testing.T) {
	t.Parallel()
	type fields struct {
		TorrentURL  string
		TorrentName string
		InfoURL     string
		Encode      []string
	}
	type args struct {
		vars map[string]string
		rls  *Release
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Release
	}{
		{
			name: "",
			fields: fields{
				TorrentName: "",
			},
			args: args{
				vars: map[string]string{
					"torrentName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
				},
				rls: &Release{},
			},
			want: &Release{
				TorrentName: "",
			},
		},
		{
			name: "",
			fields: fields{
				TorrentName: `{{ if .releaseGroup }}[{{ .releaseGroup }}] {{ end }}{{ .torrentName }} [{{ .year }}] {{ if .releaseEpisode }}{{ printf "- %02s " .releaseEpisode }}{{ end }}{{ print "[" .releaseTags "]" | replace " / " "][" }}`,
			},
			args: args{
				vars: map[string]string{
					"torrentName":    "Great BluRay SoftSubbed Anime",
					"category":       "TV Series",
					"year":           "2020",
					"releaseTags":    "Blu-ray / MKV / h264 10-bit / 1080p / FLAC 2.0 / Dual Audio / Softsubs (Sub Group) / Freeleech",
					"releaseGroup":   "Softsubs",
					"releaseEpisode": "",
					"freeleech":      "freeleech",
					"baseUrl":        "https://mock.local",
					"torrentId":      "240860011",
					"tags":           "comedy, drama, school.life, sports",
					"uploader":       "Uploader",
					"passkey":        "00000000000000000000",
				},
				rls: &Release{},
			},
			want: &Release{
				TorrentName: "[Softsubs] Great BluRay SoftSubbed Anime [2020] [Blu-ray][MKV][h264 10-bit][1080p][FLAC 2.0][Dual Audio][Softsubs (Sub Group)][Freeleech]",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &IndexerIRCParseMatch{
				TorrentURL:  tt.fields.TorrentURL,
				TorrentName: tt.fields.TorrentName,
				InfoURL:     tt.fields.InfoURL,
				Encode:      tt.fields.Encode,
			}
			p.ParseTorrentName(tt.args.vars, tt.args.rls)
			assert.Equal(t, tt.want, tt.args.rls)
		})
	}
}

func TestIRCParserGazelleGames_Parse(t *testing.T) {
	t.Parallel()
	type args struct {
		rls  *Release
		vars map[string]string
	}
	type want struct {
		title   string
		release string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Trouble.in.Paradise-GROUP in Trouble in Paradise",
				},
			},
			want: want{
				title:   "Trouble in Paradise",
				release: "Trouble.in.Paradise-GROUP",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "F.I.L.F. Game Walkthrough v.0.18 in F.I.L.F.",
				},
			},
			want: want{
				title:   "F.I.L.F.",
				release: "F.I.L.F. Game Walkthrough v.0.18",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Ni no Kuni: Dominion of the Dark Djinn in Ni no Kuni: Dominion of the Dark Djinn",
				},
			},
			want: want{
				title:   "Ni no Kuni: Dominion of the Dark Djinn",
				release: "Ni no Kuni: Dominion of the Dark Djinn",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "GazelleGames", "ggn", "GazelleGames"}),
				vars: map[string]string{
					"torrentName": "Year 2 Remastered by Insaneintherainmusic",
					"category":    "OST",
				},
			},
			want: want{
				title:   "Year 2 Remastered by Insaneintherainmusic",
				release: "Year 2 Remastered by Insaneintherainmusic",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := IRCParserGazelleGames{}
			p.Parse(tt.args.rls, tt.args.vars)
			assert.Equal(t, tt.want.release, tt.args.rls.TorrentName)
			assert.Equal(t, tt.want.title, tt.args.rls.Title)
		})
	}
}

func TestIRCParserOrpheus_Parse(t *testing.T) {
	t.Parallel()
	type args struct {
		rls  *Release
		vars map[string]string
	}
	type want struct {
		title   string
		release string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "Orpheus", "ops", "Orpheus"}),
				vars: map[string]string{
					"torrentName": "Busta Rhymes – BEACH BALL (feat. BIA) – [2023] [Single] WEB/FLAC/24bit Lossless",
					"title":       "Busta Rhymes – BEACH BALL (feat. BIA)",
					"year":        "2023",
					"releaseTags": "WEB/FLAC/24bit Lossless",
				},
			},
			want: want{
				title:   "BEACH BALL",
				release: "Busta Rhymes - BEACH BALL (feat. BIA) [2023] (WEB FLAC 24BIT Lossless)",
			},
		},
		{
			name: "",
			args: args{
				rls: NewRelease(IndexerMinimal{0, "Orpheus", "ops", "Orpheus"}),
				vars: map[string]string{
					"torrentName": "Busta Rhymes – BEACH BALL (feat. BIA) – [2023] [Single] CD/FLAC/Lossless",
					"title":       "Busta Rhymes – BEACH BALL (feat. BIA)",
					"year":        "2023",
					"releaseTags": "CD/FLAC/Lossless",
				},
			},
			want: want{
				title:   "BEACH BALL",
				release: "Busta Rhymes - BEACH BALL (feat. BIA) [2023] (CD FLAC Lossless)",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := IRCParserOrpheus{}
			p.Parse(tt.args.rls, tt.args.vars)
			assert.Equal(t, tt.want.release, tt.args.rls.TorrentName)
			assert.Equal(t, tt.want.title, tt.args.rls.Title)
		})
	}
}
