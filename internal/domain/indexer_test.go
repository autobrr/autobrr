package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexerParse_ParseMatch(t *testing.T) {
	type fields struct {
		Type          string
		ForceSizeUnit string
		Lines         []IndexerParseExtract
		Match         IndexerParseMatch
	}
	type args struct {
		vars      map[string]string
		extraVars map[string]string
		release   *Release
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		expect  *Release
		wantErr bool
	}{
		{
			name: "test_01",
			fields: fields{
				Type:          "",
				ForceSizeUnit: "",
				Lines: []IndexerParseExtract{
					{
						Test:    nil,
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
				Match: IndexerParseMatch{
					TorrentURL:  "{{ .baseUrl }}rss/download/{{ .torrentId }}/{{ .rsskey }}/{{ .torrentName }}.torrent",
					TorrentName: "",
					Encode:      []string{"torrentName"},
				},
			},
			args: args{
				vars: map[string]string{
					"category":    "TV :: Episodes HD",
					"torrentName": "The Show 2019 S03E08 2160p DV WEBRip 6CH x265 HEVC-GROUP",
					"uploader":    "Anonymous",
					"freeleech":   "",
					"baseUrl":     "https://mock.org/",
					"torrentId":   "240860011",
				},
				extraVars: map[string]string{
					"rsskey": "00000000000000000000",
				},
				release: &Release{
					Indexer:        "mock",
					FilterStatus:   ReleaseStatusFilterPending,
					Rejections:     []string{},
					Protocol:       ReleaseProtocolTorrent,
					Implementation: ReleaseImplementationIRC,
				},
			},
			expect: &Release{
				Indexer:        "mock",
				FilterStatus:   ReleaseStatusFilterPending,
				Rejections:     []string{},
				Protocol:       ReleaseProtocolTorrent,
				Implementation: ReleaseImplementationIRC,
				TorrentURL:     "https://mock.org/rss/download/240860011/00000000000000000000/The+Show+2019+S03E08+2160p+DV+WEBRip+6CH+x265+HEVC-GROUP.torrent",
			},
			wantErr: false,
		},
		{
			name: "test_02",
			fields: fields{
				Type:          "",
				ForceSizeUnit: "",
				Lines: []IndexerParseExtract{
					{
						Test:    nil,
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
				Match: IndexerParseMatch{
					TorrentURL:  "{{ .baseUrl }}/torrent/{{ .torrentId }}/download/{{ .passkey }}",
					TorrentName: `{{ if .releaseGroup }}[{{ .releaseGroup }}] {{ end }}{{ .torrentName }} [{{ .year }}] {{ if .releaseEpisode }}{{ printf "- %02s " .releaseEpisode }}{{ end }}{{ print "[" .releaseTags "]" | replace " / " "][" }}`,
					Encode:      nil,
				},
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
					"baseUrl":        "https://mock.org",
					"torrentId":      "240860011",
					"tags":           "comedy, drama, school.life, sports",
					"uploader":       "Uploader",
				},
				extraVars: map[string]string{
					"passkey": "00000000000000000000",
				},
				release: &Release{
					Indexer:        "mock",
					FilterStatus:   ReleaseStatusFilterPending,
					Rejections:     []string{},
					Protocol:       ReleaseProtocolTorrent,
					Implementation: ReleaseImplementationIRC,
				},
			},
			expect: &Release{
				Indexer:        "mock",
				FilterStatus:   ReleaseStatusFilterPending,
				Rejections:     []string{},
				Protocol:       ReleaseProtocolTorrent,
				Implementation: ReleaseImplementationIRC,
				TorrentURL:     "https://mock.org/torrent/240860011/download/00000000000000000000",
				TorrentName:    "[Softsubs] Great BluRay SoftSubbed Anime [2020] [Blu-ray][MKV][h264 10-bit][1080p][FLAC 2.0][Dual Audio][Softsubs (Sub Group)][Freeleech]",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &IndexerParse{
				Type:          tt.fields.Type,
				ForceSizeUnit: tt.fields.ForceSizeUnit,
				Lines:         tt.fields.Lines,
				Match:         tt.fields.Match,
			}

			p.ParseMatch(tt.args.vars, tt.args.extraVars, tt.args.release)
			assert.Equal(t, tt.expect, tt.args.release)
		})
	}
}
