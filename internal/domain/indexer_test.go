// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexerIRCV2ParseMatch_ParseUrls(t *testing.T) {
	t.Parallel()
	type fields struct {
		ReleaseName string
		DownloadURL string
		MagnetURI   string
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
				DownloadURL: "rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .releaseName }}.torrent",
				Encode:      []string{"releaseName"},
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"category":    "TV :: Episodes HD",
					"releaseName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
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
				DownloadURL: "/torrent/{{ .torrentId }}/download/{{ .passkey }}",
				Encode:      nil,
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"releaseName":    "Great BluRay SoftSubbed Anime",
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
				DownloadURL: "{{ .baseUrl }}rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .releaseName }}.torrent",
				Encode:      []string{"releaseName"},
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"category":    "TV :: Episodes HD",
					"releaseName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
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
				DownloadURL: "https://mock.local/rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .releaseName }}.torrent",
				Encode:      []string{"releaseName"},
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"category":    "TV :: Episodes HD",
					"releaseName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
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
				DownloadURL: "/rss/?action=download&key={{ .key }}&token={{ .token }}&hash={{ .torrentId }}&title={{ .releaseName }}",
				Encode:      []string{"releaseName"},
			},
			args: args{
				baseURL: "https://mock.local/",
				vars: map[string]string{
					"category":    "Movies/Remux",
					"releaseName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
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
		{
			name: "magnet_uri",
			fields: fields{
				MagnetURI: "magnet:?xt=urn:btih:{{ .torrentHash }}&dn={{ urlquery .releaseName }}",
			},
			args: args{
				vars: map[string]string{
					"torrentHash": "81c758d0eca5372d59e43879ecf2e2bce33a06c4",
					"releaseName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
				},
				rls: &Release{},
			},
			want: &Release{
				MagnetURI: "magnet:?xt=urn:btih:81c758d0eca5372d59e43879ecf2e2bce33a06c4&dn=The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &IndexerIRCV2ParseMatch{
				ReleaseName: tt.fields.ReleaseName,
				DownloadURL: tt.fields.DownloadURL,
				MagnetURI:   tt.fields.MagnetURI,
				InfoURL:     tt.fields.InfoURL,
				Encode:      tt.fields.Encode,
			}
			err := p.ParseURLs(tt.args.baseURL, tt.args.vars, tt.args.rls)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, tt.args.rls)
		})
	}
}

func TestIndexerIRCV2ParseMatch_ParseTorrentName(t *testing.T) {
	t.Parallel()
	type fields struct {
		ReleaseName string
		DownloadURL string
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
				ReleaseName: "",
			},
			args: args{
				vars: map[string]string{
					"releaseName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
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
				ReleaseName: `{{ if .releaseGroup }}[{{ .releaseGroup }}] {{ end }}{{ .releaseName }} [{{ .year }}] {{ if .releaseEpisode }}{{ printf "- %02s " .releaseEpisode }}{{ end }}{{ print "[" .releaseTags "]" | replace " / " "][" }}`,
			},
			args: args{
				vars: map[string]string{
					"releaseName":    "Great BluRay SoftSubbed Anime",
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
			p := &IndexerIRCV2ParseMatch{
				ReleaseName: tt.fields.ReleaseName,
				DownloadURL: tt.fields.DownloadURL,
				InfoURL:     tt.fields.InfoURL,
				Encode:      tt.fields.Encode,
			}
			err := p.ParseTorrentName(tt.args.vars, tt.args.rls)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, tt.args.rls)
		})
	}
}

func TestIndexerIRCV2Parse_MapCustomVariables(t *testing.T) {
	type fields struct {
		Type          string
		ForceSizeUnit string
		Lines         []IndexerIRCParseLine
		Match         IndexerIRCV2ParseMatch
		Mappings      IRCMappings
	}
	type args struct {
		vars       map[string]string
		expectVars map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "",
			fields: fields{
				Mappings: map[string]map[string]map[string]string{
					"announceType": {
						"0": map[string]string{
							"announceType": "NEW",
						},
						"1": map[string]string{
							"announceType": "PROMO",
						},
					},
					"categoryEnum": {
						"0": map[string]string{
							"category": "Feature Film",
						},
						"1": map[string]string{
							"category": "Short Film",
						},
						"2": map[string]string{
							"category": "Miniseries",
						},
						"3": map[string]string{
							"category": "Stand-up Comedy",
						},
						"4": map[string]string{
							"category": "Live Performance",
						},
						"5": map[string]string{
							"category": "Movie Collection",
						},
					},
					"freeleechEnum": {
						"0": map[string]string{
							"downloadVolumeFactor": "1.0",
							"uploadVolumeFactor":   "1.0",
						},
						"1": map[string]string{
							"downloadVolumeFactor": "0",
							"uploadVolumeFactor":   "1.0",
						},
						"2": map[string]string{
							"downloadVolumeFactor": "0.5",
							"uploadVolumeFactor":   "1.0",
						},
						"3": map[string]string{
							"downloadVolumeFactor": "0",
							"uploadVolumeFactor":   "0",
						},
					},
				},
			},
			args: args{
				vars: map[string]string{
					"announceType":  "1",
					"categoryEnum":  "0",
					"freeleechEnum": "1",
				},
				expectVars: map[string]string{
					"announceType":         "PROMO",
					"category":             "Feature Film",
					"categoryEnum":         "0",
					"freeleechEnum":        "1",
					"downloadVolumeFactor": "0",
					"uploadVolumeFactor":   "1.0",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &IndexerIRCV2Parse{
				Type:          tt.fields.Type,
				ForceSizeUnit: tt.fields.ForceSizeUnit,
				Lines:         tt.fields.Lines,
				Match:         tt.fields.Match,
				Mappings:      tt.fields.Mappings,
			}
			err := p.MapCustomVariables(tt.args.vars)
			assert.NoError(t, err)
			assert.Equal(t, tt.args.expectVars, tt.args.vars)
		})
	}
}
