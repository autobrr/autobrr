// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexerIRCParse_ParseMatch(t *testing.T) {
	type fields struct {
		Type          string
		ForceSizeUnit string
		Lines         []IndexerIRCParseLine
		Match         IndexerIRCParseMatch
	}
	type args struct {
		baseURL string
		vars    map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *IndexerIRCParseMatched
		wantErr bool
	}{
		{
			name: "test_01",
			fields: fields{
				Type:          "",
				ForceSizeUnit: "",
				Lines: []IndexerIRCParseLine{
					{
						Pattern: "New Torrent Announcement:\\s*<([^>]*)>\\s*Name:'(.*)' uploaded by '([^']*)'\\s*(freeleech)*\\s*-\\s*(https?\\:\\/\\/[^\\/]+\\/)torrent\\/(\\d+)",
						Vars: []string{
							"category",
							"torrentName",
							"uploader",
							"freeleech",
							"baseUrl",
							"torrentId",
						},
					},
				},
				Match: IndexerIRCParseMatch{
					TorrentURL: "rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .torrentName }}.torrent",
					Encode:     []string{"torrentName"},
				},
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
			},
			want: &IndexerIRCParseMatched{
				TorrentURL: "https://mock.local/rss/download/240860011/00000000000000000000/The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP.torrent",
			},
			wantErr: false,
		},
		{
			name: "test_02",
			fields: fields{
				Type:          "",
				ForceSizeUnit: "",
				Lines: []IndexerIRCParseLine{
					{
						Pattern: `(.*?)(?: - )?(Visual Novel|Light Novel|TV.*|Movie|Manga|OVA|ONA|DVD Special|BD Special|Oneshot|Anthology|Manhwa|Manhua|Artbook|Game|Live Action.*|)[\s\p{Zs}]{2,}\[(\d+)\] :: (.*?)(?: \/ (?:RAW|Softsubs|Hardsubs|Translated)\s\((.+)\)(?:.*Episode\s(\d+))?(?:.*(Freeleech))?.*)? \|\| (https.*)\/torrents.*\?id=\d+&torrentid=(\d+) \|\| (.+?(?:(?:\|\| Uploaded by|$))?) (?:\|\| Uploaded by: (.*))?$`,
						Vars: []string{
							"torrentName",
							"category",
							"year",
							"releaseTags",
							"releaseGroup",
							"releaseEpisode",
							"freeleech",
							"baseUrl",
							"torrentId",
							"tags",
							"uploader",
						},
					},
				},
				Match: IndexerIRCParseMatch{
					TorrentURL:  "/torrent/{{ .torrentId }}/download/{{ .passkey }}",
					TorrentName: `{{ if .releaseGroup }}[{{ .releaseGroup }}] {{ end }}{{ .torrentName }} [{{ .year }}] {{ if .releaseEpisode }}{{ printf "- %02s " .releaseEpisode }}{{ end }}{{ print "[" .releaseTags "]" | replace " / " "][" }}`,
					Encode:      nil,
				},
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
			},
			want: &IndexerIRCParseMatched{
				TorrentURL:  "https://mock.local/torrent/240860011/download/00000000000000000000",
				TorrentName: "[Softsubs] Great BluRay SoftSubbed Anime [2020] [Blu-ray][MKV][h264 10-bit][1080p][FLAC 2.0][Dual Audio][Softsubs (Sub Group)][Freeleech]",
			},
			wantErr: false,
		},
		{
			name: "test_03",
			fields: fields{
				Type:          "",
				ForceSizeUnit: "",
				Lines: []IndexerIRCParseLine{
					{
						Pattern: "New Torrent Announcement:\\s*<([^>]*)>\\s*Name:'(.*)' uploaded by '([^']*)'\\s*(freeleech)*\\s*-\\s*(https?\\:\\/\\/[^\\/]+\\/)torrent\\/(\\d+)",
						Vars: []string{
							"category",
							"torrentName",
							"uploader",
							"freeleech",
							"baseUrl",
							"torrentId",
						},
					},
				},
				Match: IndexerIRCParseMatch{
					TorrentURL: "{{ .baseUrl }}rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .torrentName }}.torrent",
					Encode:     []string{"torrentName"},
				},
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
			},
			want: &IndexerIRCParseMatched{
				TorrentURL: "https://mock.local/rss/download/240860011/00000000000000000000/The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP.torrent",
			},
			wantErr: false,
		},
		{
			name: "test_04",
			fields: fields{
				Type:          "",
				ForceSizeUnit: "",
				Lines: []IndexerIRCParseLine{
					{
						Pattern: "New Torrent Announcement:\\s*<([^>]*)>\\s*Name:'(.*)' uploaded by '([^']*)'\\s*(freeleech)*\\s*-\\s*(https?\\:\\/\\/[^\\/]+\\/)torrent\\/(\\d+)",
						Vars: []string{
							"category",
							"torrentName",
							"uploader",
							"freeleech",
							"baseUrl",
							"torrentId",
						},
					},
				},
				Match: IndexerIRCParseMatch{
					TorrentURL: "https://mock.local/rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .torrentName }}.torrent",
					Encode:     []string{"torrentName"},
				},
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
			},
			want: &IndexerIRCParseMatched{
				TorrentURL: "https://mock.local/rss/download/240860011/00000000000000000000/The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP.torrent",
			},
			wantErr: false,
		},
		{
			name: "test_04",
			fields: fields{
				Type:          "",
				ForceSizeUnit: "",
				Lines: []IndexerIRCParseLine{
					{
						Pattern: "New Torrent in category \\[([^\\]]*)\\] (.*) \\(([^\\)]*)\\) uploaded! Download\\: (https?\\:\\/\\/[^\\/]+\\/).+id=(.+)",
						Vars: []string{
							"category",
							"torrentName",
							"uploader",
							"freeleech",
							"baseUrl",
							"torrentId",
						},
					},
				},
				Match: IndexerIRCParseMatch{
					TorrentURL: "/rss/?action=download&key={{ .key }}&token={{ .token }}&hash={{ .torrentId }}&title={{ .torrentName }}",
					Encode:     []string{"torrentName"},
				},
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
			},
			want: &IndexerIRCParseMatched{
				TorrentURL: "https://mock.local/rss/?action=download&key=KEY&token=TOKEN&hash=240860011&title=The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &IndexerIRCParse{
				Type:          tt.fields.Type,
				ForceSizeUnit: tt.fields.ForceSizeUnit,
				Lines:         tt.fields.Lines,
				Match:         tt.fields.Match,
			}

			got, _ := p.ParseMatch(tt.args.baseURL, tt.args.vars)
			assert.Equal(t, tt.want, got)
		})
	}
}
