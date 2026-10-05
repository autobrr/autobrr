// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"strings"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIndexersParseAndFilter(t *testing.T) {
	t.Parallel()
	type fields struct {
		identifier         string
		identifierExternal string
		settings           map[string]string
	}
	type filterTest struct {
		filter     *domain.Filter
		match      bool
		rejections []string
	}
	type args struct {
		announceLines []string
		filters       []filterTest
	}
	type subTest struct {
		name  string
		args  args
		match bool
	}
	tests := []struct {
		name     string
		fields   fields
		match    bool
		subTests []subTest
	}{
		{
			name: "ops",
			fields: fields{
				identifier:         "orpheus",
				identifierExternal: "Orpheus",
				settings: map[string]string{
					"torrent_pass": "pass",
					"api_key":      "key",
				},
			},
			subTests: []subTest{
				{
					name: "announce_1",
					args: args{
						announceLines: []string{"TORRENT: Dirty Dike – Bogies & Alcohol – [2008] [Album] CD/MP3/320 – hip.hop,uk.hip.hop,united.kingdom – https://orpheus.network/torrents.php?id=0000000 – https://orpheus.network/torrents.php?id=0000000&torrentid=0000000&action=download"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "Album",
									Years:           "2008",
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "Single",
									Years:           "2008",
								},
								match:      false,
								rejections: []string{"[match category] not matching: got Album want: Single"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_2",
					args: args{
						announceLines: []string{"TORRENT: Dirty Dike – Bogies & Alcohol – [2024] [EP] CD/FLAC/Lossless/Cue/Log/100 – hip.hop,uk.hip.hop,united.kingdom – https://orpheus.network/torrents.php?id=0000000 – https://orpheus.network/torrents.php?id=0000000&torrentid=0000000&action=download"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "EP,Album",
									Years:           "2024",
									Quality:         []string{"Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
									Artists:         "Dirty Dike",
									Albums:          "Bogies & Alcohol",
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "EP,Album",
									Years:           "2024",
									Quality:         []string{"Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
									Log:             true,
									LogScore:        100,
									PerfectFlac:     true,
									Artists:         "Dirty Dike",
									Albums:          "Bogies & Alcohol",
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "EP,Album",
									Years:           "2024",
									Quality:         []string{"24bit Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
									Albums:          "Best album",
								},
								match:      false,
								rejections: []string{"[albums] not matching: got Bogies & Alcohol want: Best album", "[quality] not matching: got [Cue FLAC Lossless Log100 Log] want: [24bit Lossless]"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_3",
					args: args{
						announceLines: []string{"TORRENT: Dirty Dike – Bogies & Alcohol – [2024] [EP] CD/FLAC/Lossless/Cue/Log/80 – hip.hop,uk.hip.hop,united.kingdom – https://orpheus.network/torrents.php?id=0000000 – https://orpheus.network/torrents.php?id=0000000&torrentid=0000000&action=download"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "EP,Album",
									Years:           "2024",
									Quality:         []string{"24bit Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
									Log:             true,
									LogScore:        100,
									Albums:          "Best album",
								},
								match:      false,
								rejections: []string{"[albums] not matching: got Bogies & Alcohol want: Best album", "[quality] not matching: got [Cue FLAC Lossless Log80 Log] want: [24bit Lossless]", "[log score] not matching: got 80 want: 100"},
							},
						},
					},
					match: false,
				},
			},
			match: true,
		},
		{
			name: "redacted",
			fields: fields{
				identifier:         "red",
				identifierExternal: "Redacted",
				settings: map[string]string{
					"authkey":      "key",
					"torrent_pass": "pass",
					"api_key":      "key",
				},
			},
			subTests: []subTest{
				{
					name: "announce_1",
					args: args{
						announceLines: []string{"Artist - Albumname [2008] [Single] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.sh/torrents.php?id=0000000 / https://redacted.sh/torrents.php?action=download&id=0000000 - hip.hop,rhythm.and.blues,2000s"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "Single",
									Years:           "2008",
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "Album",
								},
								match:      false,
								rejections: []string{"[match category] not matching: got Single want: Album"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_2",
					args: args{
						announceLines: []string{"A really long name here - Concertos 5 and 6, Suite No 2 [1991] [Album] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.sh/torrents.php?id=0000000 / https://redacted.sh/torrents.php?action=download&id=0000000 - classical"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "EP,Album",
									Years:           "1991",
									PerfectFlac:     true,
									//Quality:         []string{"Lossless"},
									//Sources:         []string{"CD"},
									//Formats:         []string{"FLAC"},
									Tags: "classical",
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "EP,Album",
									Years:           "2024",
									Quality:         []string{"24bit Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
								},
								match:      false,
								rejections: []string{"[year] not matching: got 1991 want: 2024", "[quality] not matching: got [Cue FLAC Lossless Log100 Log] want: [24bit Lossless]"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_3",
					args: args{
						announceLines: []string{"The best artist - Album No 2 [2024] [EP] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.sh/torrents.php?id=0000000 / https://redacted.sh/torrents.php?action=download&id=0000000 - classical"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "EP",
									Years:           "2024",
									Quality:         []string{"Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
									Log:             true,
									LogScore:        100,
									Cue:             true,
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "EP,Album",
									Years:           "2024",
									Quality:         []string{"24bit Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
								},
								match:      false,
								rejections: []string{"[quality] not matching: got [Cue FLAC Lossless Log100 Log] want: [24bit Lossless]"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_4",
					args: args{
						announceLines: []string{"The best artist - Album No 2 [2024] [EP] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.sh/torrents.php?id=0000000 / https://redacted.sh/torrents.php?action=download&id=0000000 - classical"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "EP",
									Years:           "2024",
									Quality:         []string{"Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
									Log:             true,
									LogScore:        100,
									Cue:             true,
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "EP,Album",
									Years:           "2024",
									Quality:         []string{"24bit Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
								},
								match:      false,
								rejections: []string{"[quality] not matching: got [Cue FLAC Lossless Log100 Log] want: [24bit Lossless]"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_5",
					args: args{
						announceLines: []string{"The best artist - Album No 1 [2024] [EP] - FLAC / Lossless / Log / 87% / Cue / CD - https://redacted.sh/torrents.php?id=0000000 / https://redacted.sh/torrents.php?action=download&id=0000000 - classical"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "EP",
									Years:           "2024",
									Quality:         []string{"Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
									Log:             true,
									LogScore:        100,
									Cue:             true,
								},
								match:      false,
								rejections: []string{"[log score] not matching: got 87 want: 100"},
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "EP",
									PerfectFlac:     true,
								},
								match:      false,
								rejections: []string{"[perfect flac] not matching: got CD FLAC Lossless (log: true, score: 87) want: wanted Log Score 100, got 87"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_6",
					args: args{
						announceLines: []string{"The best artist - Album No 1 [2017] [Album] - FLAC / Lossless / Log / Cue / CD - https://redacted.sh/torrents.php?id=0000000 / https://redacted.sh/torrents.php?action=download&id=0000000 - Hip.Hop,Estonian,2010s"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "Album",
									Years:           "2017",
									Quality:         []string{"Lossless"},
									Sources:         []string{"CD"},
									Formats:         []string{"FLAC"},
									Log:             true,
									Cue:             true,
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "Album",
									PerfectFlac:     true,
								},
								match:      false,
								rejections: []string{"[perfect flac] not matching: got CD FLAC Lossless (log: true, score: 0) want: wanted Log Score 100, got 0"},
							},
						},
					},
					match: false,
				},
			},
			match: true,
		},
		{
			name: "mam",
			fields: fields{
				identifier:         "myanonamouse",
				identifierExternal: "MyAnonamouse",
				settings: map[string]string{
					"cookie": "mam_id: key;",
				},
			},
			subTests: []subTest{
				{
					name: "announce_1",
					args: args{
						announceLines: []string{"The Long Game: A Playbook of the World's Most Enduring Companies By: Eric Becker [English] [Audiobook] [Non-Fiction] [m4b] [132.69 MiB] - Business/Money - https://www.myanonamouse.net/t/1262674 Normal"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:            "filter_1",
									MatchCategories: "Audiobook*",
									Containers:      []string{"m4b"},
								},
								match: true,
							},
						},
					},
					match: false,
				},
			},
			match: true,
		},
		{
			name: "aither",
			fields: fields{
				identifier:         "aither",
				identifierExternal: "Aither",
				settings: map[string]string{
					"rsskey": "key",
				},
			},
			subTests: []subTest{
				{
					name: "announce_resolution_not_in_name",
					args: args{
						announceLines: []string{"Category [TV] Type [WEB-DL] Name [The Show S01E01 NF WEB-DL DD+ 5.1 H.264-GRP] Resolution [1080p] Freeleech [0%] Internal [No] Double Upload [No] Size [1.38 GB] Uploader [Anonymous] Url [https://aither.cc/torrents/download/213123123]"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:        "filter_1",
									Resolutions: []string{"1080p"},
								},
								match: true,
							},
						},
					},
					match: false,
				},
			},
			match: true,
		},
		{
			name: "theoldschool",
			fields: fields{
				identifier:         "theoldschool",
				identifierExternal: "TheOldSchool",
				settings: map[string]string{
					"rsskey": "key",
				},
			},
			subTests: []subTest{
				{
					name: "announce_source_not_in_name",
					args: args{
						announceLines: []string{"[NEW] [Series] [Example.Show.S01E01.1080p.H264-GROUP] [WEB-DL] [487.27 MiB] [0%] par tester -> https://theoldschool.cc/torrents/00000"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:    "filter_1",
									Sources: []string{"WEB-DL"},
								},
								match: true,
							},
						},
					},
				},
				{
					name: "name_source_takes_priority",
					args: args{
						announceLines: []string{"[NEW] [Series] [Example.Show.S01E01.1080p.BluRay.H264-GROUP] [WEB-DL] [487.27 MiB] [0%] par tester -> https://theoldschool.cc/torrents/00000"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:    "filter_1",
									Sources: []string{"BluRay"},
								},
								match: true,
							},
						},
					},
				},
			},
		},
		{
			name: "sharewood",
			fields: fields{
				identifier:         "sharewood",
				identifierExternal: "Sharewood",
				settings: map[string]string{
					"passkey": "key",
				},
			},
			subTests: []subTest{
				{
					name: "announce_resolution_differs_from_name",
					args: args{
						announceLines: []string{"Vidéos | La Brea S02E08 MULTi 1080p WEB x264-FW | 1.27 GiB | 1080p/i <https://sharewood.tv/torrents/la-brea-s02e08-multi-1080p-web-x264-fw.66870>"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:        "filter_1",
									Resolutions: []string{"1080p"},
								},
								match: true,
							},
						},
					},
					match: false,
				},
			},
			match: true,
		},
		{
			name: "iptorrents",
			fields: fields{
				identifier:         "iptorrents",
				identifierExternal: "IPTorrents",
				settings: map[string]string{
					"passkey": "key",
				},
			},
			subTests: []subTest{
				{
					name: "announce_binary_size",
					args: args{
						announceLines: []string{"[TV/BD] Synthetic.Show.S01E01.2160p.WEB-GROUP - https://iptorrents.com/details.php?id=123 - 7.63 GB"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:    "filter_1",
									MinSize: "8GB",
								},
								match: true,
							},
							{
								filter: &domain.Filter{
									Name:    "filter_2",
									MaxSize: "8GB",
								},
								match:      false,
								rejections: []string{"release size 8192650117 bytes is larger than filter max size 8000000000 bytes", "[max size] not matching: got 8192650117 want: 8GB"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_binary_size_mb",
					args: args{
						announceLines: []string{"[TV/x264] Synthetic.Show.S01E01.720p.WEB-GROUP FREELEECH - https://iptorrents.com/details.php?id=124 - 716.22 MB"},
						filters: []filterTest{
							{
								filter: &domain.Filter{
									Name:    "filter_1",
									MinSize: "750MB",
								},
								match: true,
							},
						},
					},
					match: false,
				},
			},
			match: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var def *domain.IndexerDefinition
			defErr := OpenAndDecodeDefinition("./definitions/"+tt.fields.identifier+".yaml", &def)
			require.NoError(t, defErr)

			def.Prepare()

			def.IdentifierExternal = tt.fields.identifierExternal
			def.SettingsMap = tt.fields.settings

			// indexer subtests
			for _, subT := range tt.subTests {
				t.Run(subT.name, func(t *testing.T) {

					rls := domain.NewRelease(domain.IndexerMinimal{ID: def.ID, Name: def.Name, Identifier: def.Identifier, IdentifierExternal: def.IdentifierExternal})
					rls.Protocol = domain.ReleaseProtocol(def.Protocol)

					// from announce/announce.go
					tmpVars := map[string]string{}

					for _, channel := range def.IRC.ChannelsMap {
						require.Len(t, subT.args.announceLines, len(channel.Parse.Lines))

						for idx, parseLine := range channel.Parse.Lines {
							match, err := parseLine.ParseLine(tmpVars, subT.args.announceLines[idx], parseLine.Ignore)
							require.NoError(t, err)
							require.Truef(t, match, "announce line did not match pattern %q: %q", parseLine.Pattern, subT.args.announceLines[idx])
						}

						// on lines matched
						parseErr := channel.Parse.Parse(def, channel.Name, tmpVars, rls)
						require.NoError(t, parseErr)
					}

					// release/service.go

					//ctx := t.Context()
					//filterSvc := filter.NewService(l, nil, nil, nil, nil, nil)

					for _, filterT := range subT.args.filters {
						t.Run(filterT.filter.Name, func(t *testing.T) {
							filter := filterT.filter

							//l := s.log.With().Str("indexer", release.Indexer).Str("filter", filter.Name).Str("release", release.TorrentName).Logger()

							// save filter on release
							rls.Filter = filter
							rls.FilterName = filter.Name
							rls.FilterID = filter.ID

							// test filter
							//match, err := filterSvc.CheckFilter(ctx, filter, rls)

							rejections, matchedFilter := filter.CheckFilter(rls)
							assert.Equal(t, strings.Join(filterT.rejections, ", "), rejections.String())
							assert.Equal(t, filterT.match, matchedFilter)
						})
					}
				})
			}
		})
	}
}
