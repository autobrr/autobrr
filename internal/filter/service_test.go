package filter

import (
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/stretchr/testify/assert"
)

func Test_checkFilterStrings(t *testing.T) {
	type args struct {
		name       string
		filterList string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test_01",
			args: args{
				name:       "The End",
				filterList: "The End, Other movie",
			},
			want: true,
		},
		{
			name: "test_02",
			args: args{
				name:       "The Simpsons S12",
				filterList: "The End, Other movie",
			},
			want: false,
		},
		{
			name: "test_03",
			args: args{
				name:       "The.Simpsons.S12",
				filterList: "The?Simpsons*, Other movie",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkFilterStrings(tt.args.name, tt.args.filterList)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_service_checkFilter(t *testing.T) {
	type args struct {
		filter   domain.Filter
		announce domain.Announce
	}

	svcMock := &service{
		repo:       nil,
		actionRepo: nil,
		indexerSvc: nil,
	}

	tests := []struct {
		name     string
		args     args
		expected bool
	}{
		{
			name: "freeleech",
			args: args{
				announce: domain.Announce{
					Freeleech: true,
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						Freeleech: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "scene",
			args: args{
				announce: domain.Announce{
					Scene: true,
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						Scene: true,
					},
				},
			},
			expected: true,
		},
		{
			name: "not_scene",
			args: args{
				announce: domain.Announce{
					Scene: false,
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						Scene: true,
					},
				},
			},
			expected: false,
		},
		{
			name: "shows_1",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Shows: "That show",
					},
				},
			},
			expected: true,
		},
		{
			name: "shows_2",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Shows: "That show, The Other show",
					},
				},
			},
			expected: true,
		},
		{
			name: "shows_3",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Shows: "That?show*, The?Other?show",
					},
				},
			},
			expected: true,
		},
		{
			name: "shows_4",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Shows: "The Other show",
					},
				},
			},
			expected: false,
		},
		{
			name: "shows_5",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Shows: "*show*",
					},
				},
			},
			expected: true,
		},
		{
			name: "shows_6",
			args: args{
				announce: domain.Announce{
					TorrentName: "That.Show.S06.1080p.BluRay.DD5.1.x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Shows: "*show*",
					},
				},
			},
			expected: true,
		},
		{
			name: "shows_7",
			args: args{
				announce: domain.Announce{
					TorrentName: "That.Show.S06.1080p.BluRay.DD5.1.x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Shows: "That?show*",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_releases_single",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						MatchReleases: "That show",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_releases_single_wildcard",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						MatchReleases: "That show*",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_releases_multiple",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						MatchReleases: "That show*, Other one",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_release_groups",
			args: args{
				announce: domain.Announce{
					TorrentName:  "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
					ReleaseGroup: "GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						MatchReleaseGroups: "GROUP1",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_release_groups_multiple",
			args: args{
				announce: domain.Announce{
					TorrentName:  "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
					ReleaseGroup: "GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						MatchReleaseGroups: "GROUP1,GROUP2",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_release_groups_dont_match",
			args: args{
				announce: domain.Announce{
					TorrentName:  "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
					ReleaseGroup: "GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						MatchReleaseGroups: "GROUP2",
					},
				},
			},
			expected: false,
		},
		{
			name: "except_release_groups",
			args: args{
				announce: domain.Announce{
					TorrentName:  "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
					ReleaseGroup: "GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterP2P: domain.FilterP2P{
						ExceptReleaseGroups: "GROUP1",
					},
				},
			},
			expected: false,
		},
		{
			name: "match_uploaders",
			args: args{
				announce: domain.Announce{
					Uploader: "Uploader1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						MatchUploaders: "Uploader1",
					},
				},
			},
			expected: true,
		},
		{
			name: "non_match_uploaders",
			args: args{
				announce: domain.Announce{
					Uploader: "Uploader2",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						MatchUploaders: "Uploader1",
					},
				},
			},
			expected: false,
		},
		{
			name: "except_uploaders",
			args: args{
				announce: domain.Announce{
					Uploader: "Uploader1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						ExceptUploaders: "Uploader1",
					},
				},
			},
			expected: false,
		},
		{
			name: "resolutions_1080p",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 1080p BluRay DD5.1 x264-GROUP1",
					Resolution:  "1080p",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Resolutions: []string{"1080p"},
					},
				},
			},
			expected: true,
		},
		{
			name: "resolutions_2160p",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 2160p BluRay DD5.1 x264-GROUP1",
					Resolution:  "2160p",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Resolutions: []string{"2160p"},
					},
				},
			},
			expected: true,
		},
		{
			name: "resolutions_no_match",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 2160p BluRay DD5.1 x264-GROUP1",
					Resolution:  "2160p",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Resolutions: []string{"1080p"},
					},
				},
			},
			expected: false,
		},
		{
			name: "codecs_1_match",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 2160p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Codecs: []string{"x264"},
					},
				},
			},
			expected: true,
		},
		{
			name: "codecs_2_no_match",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 2160p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Codecs: []string{"h264"},
					},
				},
			},
			expected: false,
		},
		{
			name: "sources_1_match",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 2160p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Sources: []string{"BluRay"},
					},
				},
			},
			expected: true,
		},
		{
			name: "sources_2_no_match",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 2160p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Sources: []string{"WEB"},
					},
				},
			},
			expected: false,
		},
		{
			name: "years_1",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Years: "2020",
					},
				},
			},
			expected: true,
		},
		{
			name: "years_2",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Years: "2020,1990",
					},
				},
			},
			expected: true,
		},
		{
			name: "years_3_no_match",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Years: "1990",
					},
				},
			},
			expected: false,
		},
		{
			name: "years_4_no_match",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Show S06 2160p BluRay DD5.1 x264-GROUP1",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterTVMovies: domain.FilterTVMovies{
						Years: "2020",
					},
				},
			},
			expected: false,
		},
		{
			name: "match_categories_1",
			args: args{
				announce: domain.Announce{
					Category: "TV",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						MatchCategories: "TV",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_categories_2",
			args: args{
				announce: domain.Announce{
					Category: "TV :: HD",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						MatchCategories: "*TV*",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_categories_3",
			args: args{
				announce: domain.Announce{
					Category: "TV :: HD",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						MatchCategories: "*TV*, *HD*",
					},
				},
			},
			expected: true,
		},
		{
			name: "match_categories_4_no_match",
			args: args{
				announce: domain.Announce{
					Category: "TV :: HD",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						MatchCategories: "Movies",
					},
				},
			},
			expected: false,
		},
		{
			name: "except_categories_1",
			args: args{
				announce: domain.Announce{
					Category: "Movies",
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						ExceptCategories: "Movies",
					},
				},
			},
			expected: false,
		},
		{
			name: "match_multiple_fields_1",
			args: args{
				announce: domain.Announce{
					TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
					Category:    "Movies",
					Freeleech:   true,
				},
				filter: domain.Filter{
					Enabled: true,
					FilterAdvanced: domain.FilterAdvanced{
						MatchCategories: "Movies",
					},
					FilterTVMovies: domain.FilterTVMovies{
						Resolutions: []string{"2160p"},
						Sources:     []string{"BluRay"},
						Years:       "2020",
					},
					FilterP2P: domain.FilterP2P{
						MatchReleaseGroups: "GROUP1",
						MatchReleases:      "That movie",
						Freeleech:          true,
					},
				},
			},
			expected: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svcMock.checkFilter(tt.args.filter, tt.args.announce)
			assert.Equal(t, tt.expected, got)
		})
	}
}
