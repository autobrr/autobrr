// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package indexer

import (
	"io"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
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
								rejections: []string{"category not matching. got: Album want: Single"},
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
								rejections: []string{"albums not matching. got: Bogies & Alcohol want: Best album", "quality not matching. got: [FLAC Lossless] want: [24bit Lossless]"},
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
								rejections: []string{"albums not matching. got: Bogies & Alcohol want: Best album", "quality not matching. got: [Cue FLAC Lossless Log80 Log] want: [24bit Lossless]", "log score. got 80 want: 100"},
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
						announceLines: []string{"Artist - Albumname [2008] [Single] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.ch/torrents.php?id=0000000 / https://redacted.ch/torrents.php?action=download&id=0000000 - hip.hop,rhythm.and.blues,2000s"},
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
								rejections: []string{"category not matching. got: Album want: Single"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_2",
					args: args{
						announceLines: []string{"A really long name here - Concertos 5 and 6, Suite No 2 [1991] [Album] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.ch/torrents.php?id=0000000 / https://redacted.ch/torrents.php?action=download&id=0000000 - classical"},
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
								rejections: []string{"year not matching. got: 1991 want: 2024", "quality not matching. got: [Cue FLAC Lossless Log100 Log] want: [24bit Lossless]"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_3",
					args: args{
						announceLines: []string{"The best artist - Album No 2 [2024] [EP] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.ch/torrents.php?id=0000000 / https://redacted.ch/torrents.php?action=download&id=0000000 - classical"},
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
								rejections: []string{"quality not matching. got: [FLAC Lossless] want: [24bit Lossless]"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_4",
					args: args{
						announceLines: []string{"The best artist - Album No 2 [2024] [EP] - FLAC / Lossless / Log / 100% / Cue / CD - https://redacted.ch/torrents.php?id=0000000 / https://redacted.ch/torrents.php?action=download&id=0000000 - classical"},
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
								rejections: []string{"quality not matching. got: [FLAC Lossless] want: [24bit Lossless]"},
							},
						},
					},
					match: false,
				},
				{
					name: "announce_5",
					args: args{
						announceLines: []string{"The best artist - Album No 1 [2024] [EP] - FLAC / Lossless / Log / 87% / Cue / CD - https://redacted.ch/torrents.php?id=0000000 / https://redacted.ch/torrents.php?action=download&id=0000000 - classical"},
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
								rejections: []string{"log score. got: 87 want: 100"},
							},
							{
								filter: &domain.Filter{
									Name:            "filter_2",
									MatchCategories: "EP",
									PerfectFlac:     true,
								},
								match:      false,
								rejections: []string{"wanted: perfect flac. got: [Cue FLAC Lossless Log87 Log]"},
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
			//l := zerolog.New(io.Discard)
			//l := logger.Mock()

			i, err := OpenAndProcessDefinition("./definitions/" + tt.fields.identifier + ".yaml")
			assert.NoError(t, err)

			i.IdentifierExternal = tt.fields.identifierExternal
			i.SettingsMap = tt.fields.settings

			ll := zerolog.New(io.Discard)

			// indexer subtests
			for _, subT := range tt.subTests {
				t.Run(subT.name, func(t *testing.T) {

					// from announce/announce.go
					tmpVars := map[string]string{}
					parseFailed := false

					for idx, parseLine := range i.IRC.Parse.Lines {
						match, err := ParseLine(&ll, parseLine.Pattern, parseLine.Vars, tmpVars, subT.args.announceLines[idx], parseLine.Ignore)
						if err != nil {
							parseFailed = true
							break
						}

						if !match {
							parseFailed = true
							break
						}
					}

					if parseFailed {
						return
					}

					rls := domain.NewRelease(domain.IndexerMinimal{ID: i.ID, Name: i.Name, Identifier: i.Identifier, IdentifierExternal: i.IdentifierExternal})
					rls.Protocol = domain.ReleaseProtocol(i.Protocol)

					// on lines matched
					err = i.IRC.Parse.Parse(i, tmpVars, rls)
					assert.NoError(t, err)

					// release/service.go

					//ctx := context.Background()
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
							assert.Equal(t, rejections.Len(), len(filterT.rejections))
							assert.Equal(t, filterT.match, matchedFilter)
						})
					}
				})
			}
		})
	}
}
