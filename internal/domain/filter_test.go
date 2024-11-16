// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter_CheckFilter(t *testing.T) {
	type args struct {
		filter           Filter
		rejectionReasons *RejectionReasons
	}
	tests := []struct {
		name   string
		fields *Release
		args   args
		want   bool
	}{
		//{
		//	name:   "test_no_size",
		//	fields: &Release{Size: uint64(0)},
		//	args: args{
		//		filter: Filter{
		//			Enabled:       true,
		//			FilterGeneral: FilterGeneral{MinSize: "10 GB", MaxSize: "20GB"},
		//		},
		//	},
		//	want: false, // additional checks
		//},
		{
			name: "movie_parse_1",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001),
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "TV*,Movies*",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2020",
					MatchReleaseGroups: "GROUP1",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_parse_1",
			fields: &Release{
				TorrentName: "White Christmas 1954 2160p Remux DoVi HDR10 HEVC DTS-HD MA 5.1-VHS",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					Sources:            []string{"BluRay", "UHD.BluRay"},
					MatchReleaseGroups: "VHS",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},

		{
			name: "movie_parse_2",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p Blu-Ray DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001),
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "Movies",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2020",
					MatchReleaseGroups: "GROUP1",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_parse_3",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p WEBDL DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001),
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "Movies",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"2160p"},
					Sources:            []string{"WEB-DL"},
					Codecs:             []string{"x264"},
					Years:              "2020",
					MatchReleaseGroups: "GROUP1",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_parse_shows",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001),
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "Movies",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2020",
					MatchReleaseGroups: "GROUP1",
					Shows:              "That Movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_parse_shows_1",
			fields: &Release{
				TorrentName: "That.Movie.2020.2160p.BluRay.DD5.1.x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001),
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "Movies",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2020",
					MatchReleaseGroups: "GROUP1",
					Shows:              "That Movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_parse_multiple_shows",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001),
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "Movies",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2020",
					MatchReleaseGroups: "GROUP1",
					Shows:              "That Movie, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_parse_multiple_shows_1",
			fields: &Release{
				TorrentName: "That.Movie.2020.2160p.BluRay.DD5.1.x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001),
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "Movies",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2020",
					MatchReleaseGroups: "GROUP1",
					Shows:              "That Movie, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_parse_wildcard_shows",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001), // 30GB
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "Movies, tv",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2015,2018-2022",
					MatchReleaseGroups: "GROUP1,BADGROUP",
					Shows:              "*Movie*, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_except_category_1",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001), // 30GB
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					ExceptCategories:   "*movies*",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2015,2018-2022",
					MatchReleaseGroups: "GROUP1,BADGROUP",
					Shows:              "*Movie*, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except category", got: "Movies", want: "*movies*"}}},
			},
			want: false,
		},
		{
			name: "movie_except_category_1",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001), // 30GB
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					ExceptCategories:   "*tv*",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2015,2018-2022",
					MatchReleaseGroups: "GROUP1,BADGROUP",
					Shows:              "*Movie*, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_bad_category",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				Category:    "Movies",
				Freeleech:   true,
				Size:        uint64(30000000001), // 30GB
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2015,2018-2022",
					MatchReleaseGroups: "GROUP1,BADGROUP",
					Shows:              "*Movie*, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match category", got: "Movies", want: "*tv*"}}},
			},
			want: false,
		},
		{
			name: "movie_bad_category_2",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				//Category:    "Movies",
				Categories: []string{"Movies/HD", "2040"},
				Freeleech:  true,
				Size:       uint64(30000000001), // 30GB
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2015,2018-2022",
					MatchReleaseGroups: "GROUP1,BADGROUP",
					Shows:              "*Movie*, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match category", got: "Movies/HD,2040", want: "*tv*"}}},
			},
			want: false,
		},
		{
			name: "movie_category_2",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				//Category:    "Movies",
				Categories: []string{"Movies/HD", "2040"},
				Freeleech:  true,
				Size:       uint64(30000000001), // 30GB
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*Movies*",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2015,2018-2022",
					MatchReleaseGroups: "GROUP1,BADGROUP",
					Shows:              "*Movie*, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_category_3",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				//Category:    "Movies",
				Categories: []string{"Movies/HD", "2040"},
				Freeleech:  true,
				Size:       uint64(30000000001), // 30GB
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "2040",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2015,2018-2022",
					MatchReleaseGroups: "GROUP1,BADGROUP",
					Shows:              "*Movie*, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "movie_category_4",
			fields: &Release{
				TorrentName: "That Movie 2020 2160p BluRay DD5.1 x264-GROUP1",
				//Category:    "Movies",
				Categories: []string{"Movies/HD", "2040"},
				Freeleech:  true,
				Size:       uint64(30000000001), // 30GB
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*HD*",
					Freeleech:          true,
					MinSize:            "10 GB",
					MaxSize:            "40GB",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"BluRay"},
					Codecs:             []string{"x264"},
					Years:              "2015,2018-2022",
					MatchReleaseGroups: "GROUP1,BADGROUP",
					Shows:              "*Movie*, good story, bad movie",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "tv_match_season_episode",
			fields: &Release{
				TorrentName: "Good show S01E01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"WEB-DL"},
					Codecs:             []string{"HEVC"},
					MatchReleaseGroups: "GROUP1,GROUP2",
					Seasons:            "1,2",
					Episodes:           "1",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "tv_match_season",
			fields: &Release{
				TorrentName: "Good show S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"WEB-DL"},
					Codecs:             []string{"HEVC"},
					MatchReleaseGroups: "GROUP1,GROUP2",
					Seasons:            "1,2",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "tv_bad_match_season",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					Resolutions:        []string{"1080p", "2160p"},
					Sources:            []string{"WEB-DL"},
					Codecs:             []string{"HEVC"},
					MatchReleaseGroups: "GROUP1,GROUP2",
					Seasons:            "1",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "season", got: 2, want: "1"}}},
			},
			want: false,
		},
		{
			name: "match_uploader",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "except_uploader",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Anonymous",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					ExceptUploaders: "Anonymous",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except uploaders", got: "Anonymous", want: "Anonymous"}}},
			},
			want: false,
		},
		{
			name: "match_except_uploader",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1,Uploader2",
					ExceptUploaders: "Anonymous",
					Shows:           "Good show",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_tags_empty",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"tv"},
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1,Uploader2",
					ExceptUploaders: "Anonymous",
					Shows:           "Good show",
					Tags:            "tv",
					TagsMatchLogic:  "",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_tags_any",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"tv"},
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1,Uploader2",
					ExceptUploaders: "Anonymous",
					Shows:           "Good show",
					Tags:            "tv",
					TagsMatchLogic:  "ANY",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_tags_all",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"tv", "foreign"},
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1,Uploader2",
					ExceptUploaders: "Anonymous",
					Shows:           "Good show",
					Tags:            "tv,foreign",
					TagsMatchLogic:  "ALL",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_tags_any_bad",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"foreign"},
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1,Uploader2",
					ExceptUploaders: "Anonymous",
					Shows:           "Good show",
					Tags:            "tv",
					TagsMatchLogic:  "ANY",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match tags: ANY", got: []string{"foreign"}, want: "tv"}}},
			},
			want: false,
		},
		{
			name: "match_tags_all_bad",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"foreign"},
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1,Uploader2",
					ExceptUploaders: "Anonymous",
					Shows:           "Good show",
					Tags:            "tv,foreign",
					TagsMatchLogic:  "ALL",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match tags: ALL", got: []string{"foreign"}, want: "tv,foreign"}}},
			},
			want: false,
		},
		{
			name: "match_except_tags_any",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"foreign"},
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1,Uploader2",
					ExceptUploaders: "Anonymous",
					Shows:           "Good show",
					ExceptTags:      "tv",
					TagsMatchLogic:  "ANY",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_except_tags_all",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"foreign"},
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					MatchUploaders:  "Uploader1,Uploader2",
					ExceptUploaders: "Anonymous",
					Shows:           "Good show",
					ExceptTags:      "tv,foreign",
					TagsMatchLogic:  "ALL",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except tags: ANY", got: []string{"foreign"}, want: "tv,foreign"}}},
			},
			want: false,
		},
		{
			name: "match_except_tags_any_2",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"foreign"},
			},
			args: args{
				filter: Filter{
					Enabled:              true,
					MatchCategories:      "*tv*",
					MatchUploaders:       "Uploader1,Uploader2",
					ExceptUploaders:      "Anonymous",
					Shows:                "Good show",
					ExceptTags:           "foreign",
					ExceptTagsMatchLogic: "ANY",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except tags: ANY", got: []string{"foreign"}, want: "foreign"}}},
			},
			want: false,
		},
		{
			name: "match_except_tags_all_2",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "TV",
				Uploader:    "Uploader1",
				Tags:        []string{"tv", "foreign"},
			},
			args: args{
				filter: Filter{
					Enabled:              true,
					MatchCategories:      "*tv*",
					MatchUploaders:       "Uploader1,Uploader2",
					ExceptUploaders:      "Anonymous",
					Shows:                "Good show",
					ExceptTags:           "foreign,tv",
					ExceptTagsMatchLogic: "ALL",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except tags: ALL", got: []string{"tv", "foreign"}, want: "foreign,tv"}}},
			},
			want: false,
		},
		{
			name: "match_group_1",
			fields: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show",
					MatchReleaseGroups: "GROUP",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_group_potential_partial_1",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-ift",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "ift",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_group_potential_partial_2",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "ift",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match release groups", got: "GROUP", want: "ift"}}},
			},
			want: false,
		},
		//{
		//	name: "match_group_potential_partial_3",
		//	fields: &Release{
		//		TorrentName: "[AnimeGroup] Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC",
		//		Category:    "TV",
		//		Uploader:    "Uploader1",
		//	},
		//	args: args{
		//		filter: Filter{
		//			Enabled:            true,
		//			MatchCategories:    "*tv*",
		//			MatchUploaders:     "Uploader1,Uploader2",
		//			ExceptUploaders:    "Anonymous",
		//			Shows:              "Good show shift",
		//			MatchReleaseGroups: "[AnimeGroup]",
		//		},
		//	},
		//	want: true,
		//},
		{
			name: "except_release_1",
			fields: &Release{
				TorrentName: "Good show shift S02 NORDiC 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "Good show shift",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except releases", got: "Good show shift S02 NORDiC 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP", want: "Good show shift"}}},
			},
			want: false,
		},
		{
			name: "except_release_2",
			fields: &Release{
				TorrentName: "Good show shift S02 NORDiC 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except releases", got: "Good show shift S02 NORDiC 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP", want: "NORDiC"}}},
			},
			want: false,
		},
		{
			name: "except_release_3",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "except_release_4",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC,*shift*",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except releases", got: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP", want: "NORDiC,*shift*"}}},
			},
			want: false,
		},
		{
			name: "match_hdr_1",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
					MatchHDR:           []string{"DV", "HDR"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_hdr_2",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DoVi HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
					MatchHDR:           []string{"DV", "HDR"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_hdr_3",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DoVi HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
					ExceptHDR:          []string{"DV", "HDR", "DoVi"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except hdr", got: []string{"DV"}, want: []string{"DV", "HDR", "DoVi"}}}},
			},
			want: false,
		},
		{
			name: "match_hdr_4",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
					MatchHDR:           []string{"DV", "HDR", "DoVi"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match hdr", got: []string(nil), want: []string{"DV", "HDR", "DoVi"}}}},
			},
			want: false,
		},
		{
			name: "match_hdr_5",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
					ExceptHDR:          []string{"DV", "HDR", "DoVi"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_hdr_6",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos HDR HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
					ExceptHDR:          []string{"DV", "DoVi"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_hdr_7",
			fields: &Release{
				TorrentName: "Good show dvorak shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos HDR HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show dvorak shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
					ExceptHDR:          []string{"DV", "DoVi"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_hdr_8",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos HDR10+ HEVC-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:            true,
					MatchCategories:    "*tv*",
					MatchUploaders:     "Uploader1,Uploader2",
					ExceptUploaders:    "Anonymous",
					Shows:              "Good show shift",
					MatchReleaseGroups: "GROUP",
					ExceptReleases:     "NORDiC",
					MatchHDR:           []string{"DV", "DoVi", "HDR10+"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_hdr_9",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HDR HEVC-GROUP",
			},
			args: args{
				filter: Filter{
					Enabled:  true,
					MatchHDR: []string{"DV HDR"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_hdr_10",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HDR10 HEVC-GROUP",
			},
			args: args{
				filter: Filter{
					Enabled:  true,
					MatchHDR: []string{"DV HDR"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match hdr", got: []string{"DV", "HDR10"}, want: []string{"DV HDR"}}}},
			},
			want: false,
		},
		{
			name: "match_hdr_11",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos HDR10 HEVC-GROUP",
			},
			args: args{
				filter: Filter{
					Enabled:  true,
					MatchHDR: []string{"DV", "HDR"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match hdr", got: []string{"HDR10"}, want: []string{"DV", "HDR"}}}},
			},
			want: false,
		},
		{
			name: "match_music_1",
			fields: &Release{
				TorrentName: "Artist - Albumname FLAC CD",
				ReleaseTags: "FLAC / 24bit Lossless / Log / 100% / Cue / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "Album",
					Artists:         "Artist",
					Media:           []string{"CD"},
					Formats:         []string{"FLAC"},
					Quality:         []string{"24bit Lossless"},
					Log:             true,
					Cue:             true,
					//LogScore:        100,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_music_2",
			fields: &Release{
				TorrentName: "Artist-Albumname-SINGLE-WEB-2023-GROUP",
				ReleaseTags: "MP3 / 320 / WEB",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "Album",
					Artists:         "Artist",
					PerfectFlac:     true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "perfect flac", got: []string{"320", "MP3"}, want: "Cue, Log, Log Score 100, FLAC and 24bit Lossless"}}},
			},
			want: false,
		},
		{
			name: "match_music_3",
			fields: &Release{
				TorrentName: "Artist - Albumname FLAC CD",
				ReleaseTags: "FLAC / Lossless / Log / 100% / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "Album",
					Artists:         "Artist",
					PerfectFlac:     true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "perfect flac", got: []string{"FLAC", "Lossless", "Log100", "Log"}, want: "Cue, Log, Log Score 100, FLAC and 24bit Lossless"}}},
			},
			want: false,
		},
		{
			name: "match_music_4",
			fields: &Release{
				TorrentName: "Artist - Albumname FLAC CD",
				ReleaseTags: "FLAC / Lossless / Log / 100% / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "Album",
					Artists:         "Artist",
					Media:           []string{"CD"},
					Formats:         []string{"FLAC"},
					Quality:         []string{"24bit Lossless"},
					//PerfectFlac: true,
					Log:      true,
					LogScore: 100,
					Cue:      true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "quality", got: []string{"FLAC", "Lossless", "Log100", "Log"}, want: []string{"24bit Lossless"}}, {key: "cue", got: []string{"FLAC", "Lossless", "Log100", "Log"}, want: "Cue"}}},
			},
			want: false,
		},
		{
			name: "match_music_5",
			fields: &Release{
				//TorrentName: "Artist - Albumname FLAC CD",
				TorrentName: "Artist - Albumname [2022] [Album] (FLAC 24bit Lossless CD)",
				Year:        2022,
				ReleaseTags: "FLAC / 24bit Lossless / Log / 100% / Cue / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:           true,
					MatchReleaseTypes: []string{"Album"},
					Years:             "2020-2022",
					Artists:           "Artist",
					Media:             []string{"CD"},
					Formats:           []string{"FLAC"},
					Quality:           []string{"24bit Lossless"},
					//PerfectFlac:       true,
					//Log:               true,
					//LogScore:          100,
					Cue: true,
					//Cue: true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_music_6",
			fields: &Release{
				TorrentName: "Artist - Albumname FLAC CD",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:           true,
					MatchReleaseTypes: []string{"Single"},
					Artists:           "Artist",
					Media:             []string{"CD"},
					Formats:           []string{"FLAC"},
					Quality:           []string{"24bit Lossless", "Lossless"},
					PerfectFlac:       true,
					Log:               true,
					LogScore:          100,
					Cue:               true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "release type", got: "Album", want: []string{"Single"}}}},
			},
			want: false,
		},
		{
			name: "match_music_7",
			fields: &Release{
				TorrentName: "Artist - Albumname FLAC CD",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:           true,
					MatchReleaseTypes: []string{"Album"},
					Artists:           "Artiiiist",
					Media:             []string{"CD"},
					Formats:           []string{"FLAC"},
					Quality:           []string{"24bit Lossless", "Lossless"},
					PerfectFlac:       true,
					Log:               true,
					LogScore:          100,
					Cue:               true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "artists", got: "Artist", want: "Artiiiist"}}},
			},
			want: false,
		},
		{
			name: "match_music_8",
			fields: &Release{
				TorrentName: "Artist - Albumname FLAC CD",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:           true,
					MatchReleaseTypes: []string{"Album"},
					Artists:           "Artist",
					Albums:            "Albumname",
					Media:             []string{"CD"},
					Formats:           []string{"FLAC"},
					Quality:           []string{"24bit Lossless", "Lossless"},
					PerfectFlac:       true,
					Log:               true,
					//LogScore:          100,
					Cue: true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_music_9",
			fields: &Release{
				TorrentName: "Artist - Albumname [2022] [Album] (FLAC 24bit Lossless CD)",
				Year:        2022,
				ReleaseTags: "FLAC / 24bit Lossless / Log / 100% / Cue / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:           true,
					MatchReleaseTypes: []string{"Album"},
					Years:             "2020-2022",
					Artists:           "Artist",
					Media:             []string{"CD"},
					Formats:           []string{"FLAC"},
					Quality:           []string{"Lossless"},
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "quality", got: []string{"24BIT Lossless", "Cue", "FLAC", "Log100", "Log"}, want: []string{"Lossless"}}}},
			},
			want: false,
		},
		{
			name: "match_anime_1",
			fields: &Release{
				TorrentName: "Kaginado",
				ReleaseTags: "Web / MKV / h264 / 1080p / AAC 2.0 / Softsubs (SubsPlease) / Episode 22 / Freeleech",
			},
			args: args{
				filter: Filter{
					Enabled:   true,
					Freeleech: true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_anime_2",
			fields: &Release{
				TorrentName: "Kaginado",
				ReleaseTags: "Web / MKV / h264 / 1080p / AAC 2.0 / Softsubs (SubsPlease) / Episode 22",
			},
			args: args{
				filter: Filter{
					Enabled:   true,
					Freeleech: true,
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "freeleech", got: false, want: true}}},
			},
			want: false,
		},
		{
			name: "match_light_novel_1",
			fields: &Release{
				TorrentName: "[Group] -Name of a Novel Something Good-  [2012][Translated (Group)][EPUB]",
				Title:       "-Name of a Novel Something Good-",
				Category:    "Light Novel",
				Year:        2012,
				ReleaseTags: "Translated (Group) / EPUB",
				Group:       "Group",
			},
			args: args{
				filter: Filter{
					MatchReleases:      "(?:.*Something Good.*|.*Something Bad.*)",
					UseRegex:           true,
					MatchReleaseGroups: "Group",
					MatchCategories:    "Light Novel",
					MatchReleaseTags:   "*EPUB*",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "match_daily",
			fields: &Release{
				TorrentName: "Daily talk show 2022 04 20 Someone 1080p WEB-DL h264-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					Shows:           "Daily talk show",
					Years:           "2022",
					Months:          "04",
					Days:            "20",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{}},
			},
			want: true,
		},
		{
			name: "daily_dont_match",
			fields: &Release{
				TorrentName: "Daily talk show 2022 04 20 Someone 1080p WEB-DL h264-GROUP",
				Category:    "TV",
				Uploader:    "Uploader1",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "*tv*",
					Shows:           "Daaaaaily talk show",
					Years:           "2022",
					//Months:          "05",
				},
				rejectionReasons: &RejectionReasons{data: []Rejection{{key: "shows", got: "Daily talk show", want: "Daaaaaily talk show"}}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields // Release

			r.ParseString(tt.fields.TorrentName) // Parse TorrentName into struct
			rejections, got := tt.args.filter.CheckFilter(r)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.args.rejectionReasons, rejections)
		})
	}
}

func TestFilter_CheckFilter1(t *testing.T) {
	type fields Filter
	type args struct {
		r *Release
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		rejectionReasons *RejectionReasons
		wantMatch        bool
	}{
		{
			name: "test_1",
			fields: fields{
				Shows:              "WeCrashed",
				Seasons:            "1",
				Resolutions:        []string{"2160p"},
				Sources:            []string{"WEB-DL"},
				Codecs:             []string{"x265"},
				MatchReleaseGroups: "NOSiViD",
				MatchHDR:           []string{"DV", "HDR"},
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_2",
			fields: fields{
				Shows:              "WeCrashed",
				Seasons:            "1",
				Episodes:           "2-8",
				Resolutions:        []string{"2160p"},
				Sources:            []string{"WEB-DL"},
				Codecs:             []string{"x265"},
				MatchReleaseGroups: "NOSiViD",
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "episodes", got: 0, want: "2-8"}}},
			wantMatch:        false,
		},
		{
			name: "test_3",
			fields: fields{
				Shows:              "WeCrashed",
				Seasons:            "1",
				Resolutions:        []string{"2160p"},
				Sources:            []string{"WEB-DL"},
				Codecs:             []string{"x265"},
				MatchReleaseGroups: "NOSiViD",
				MatchHDR:           []string{"HDR"},
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match hdr", got: []string{"DV"}, want: []string{"HDR"}}}},
			wantMatch:        false,
		},
		{
			name: "test_4",
			fields: fields{
				Shows:              "WeCrashed",
				Seasons:            "1",
				Resolutions:        []string{"2160p"},
				Sources:            []string{"WEB-DL"},
				Codecs:             []string{"x265"},
				MatchReleaseGroups: "NOSiViD",
				ExceptHDR:          []string{"DV", "HDR"},
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except hdr", got: []string{"DV"}, want: []string{"DV", "HDR"}}}},
			wantMatch:        false,
		},
		{
			name: "test_5",
			fields: fields{
				Shows:              "WeWork",
				Seasons:            "1",
				Resolutions:        []string{"2160p"},
				Sources:            []string{"WEB-DL"},
				Codecs:             []string{"x265"},
				MatchReleaseGroups: "NOSiViD",
				ExceptHDR:          []string{"DV", "HDR"},
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "shows", got: "WeCrashed", want: "WeWork"}, {key: "except hdr", got: []string{"DV"}, want: []string{"DV", "HDR"}}}},
			wantMatch:        false,
		},
		{
			name: "test_6",
			fields: fields{
				Shows:               "WeWork",
				Seasons:             "1",
				Resolutions:         []string{"2160p"},
				Sources:             []string{"WEB-DL"},
				Codecs:              []string{"x265"},
				ExceptReleaseGroups: "NOSiViD",
				ExceptHDR:           []string{"DV", "HDR"},
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "shows", got: "WeCrashed", want: "WeWork"}, {key: "except release groups", got: "NOSiViD", want: "NOSiViD"}, {key: "except hdr", got: []string{"DV"}, want: []string{"DV", "HDR"}}}},
			wantMatch:        false,
		},
		{
			name: "test_7",
			fields: fields{
				Shows:               "WeWork",
				Seasons:             "1",
				Resolutions:         []string{"2160p"},
				Sources:             []string{"WEB-DL"},
				Codecs:              []string{"x265"},
				ExceptReleaseGroups: "NOSiViD",
				ExceptHDR:           []string{"DV", "HDR"},
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "shows", got: "WeCrashed", want: "WeWork"}, {key: "except release groups", got: "NOSiViD", want: "NOSiViD"}, {key: "source", got: "WEB", want: []string{"WEB-DL"}}, {key: "except hdr", got: []string{"DV"}, want: []string{"DV", "HDR"}}}},
			wantMatch:        false,
		},
		{
			name: "test_8",
			fields: fields{
				Shows:              "WeCrashed",
				Seasons:            "1",
				Resolutions:        []string{"2160p"},
				Sources:            []string{"WEB"},
				Codecs:             []string{"x265"},
				MatchReleaseGroups: "NOSiViD",
				MatchHDR:           []string{"DV", "HDR"},
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "source", got: "WEB-DL", want: []string{"WEB"}}}},
			wantMatch:        false,
		},
		{
			name: "test_9",
			fields: fields{
				Shows:              "WeCrashed",
				Seasons:            "1",
				Resolutions:        []string{"2160p"},
				Sources:            []string{"WEB"},
				Codecs:             []string{"x265"},
				MatchReleaseGroups: "NOSiViD",
				MatchHDR:           []string{"DV", "HDR"},
			},
			args:             args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.Blu-ray.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "source", got: "BluRay", want: []string{"WEB"}}}},
			wantMatch:        false,
		},
		{
			name: "test_10",
			fields: fields{
				Resolutions: []string{"2160p"},
				Sources:     []string{"BluRay"},
				Codecs:      []string{"x265", "HEVC"},
				MatchHDR:    []string{"DV", "HDR"},
				ExceptOther: []string{"REMUX", "HYBRID"},
			},
			args:             args{&Release{TorrentName: "Stranger Things S02 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-FraMeSToR"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "source", got: "UHD.BluRay", want: []string{"BluRay"}}, {key: "except other", got: []string{"HYBRiD", "REMUX"}, want: []string{"REMUX", "HYBRID"}}}},
			wantMatch:        false,
		},
		{
			name: "test_10",
			fields: fields{
				Resolutions: []string{"2160p"},
				Sources:     []string{"UHD.BluRay"},
				Codecs:      []string{"x265", "HEVC"},
				MatchHDR:    []string{"DV", "HDR"},
				MatchOther:  []string{"REMUX", "HYBRID"},
			},
			args:             args{&Release{TorrentName: "Stranger Things S02 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-FraMeSToR"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_11",
			fields: fields{
				Resolutions: []string{"1080p"},
				Sources:     []string{"BluRay"},
				Codecs:      []string{"HEVC"},
				//MatchHDR:    []string{"DV", "HDR"},
			},
			args:             args{&Release{TorrentName: "Food Wars!: Shokugeki no Soma S05 2020 1080p BluRay HEVC 10-Bit DD2.0 Dual Audio -ZR-"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_12",
			fields: fields{
				Resolutions: []string{"2160p"},
				Codecs:      []string{"h.265"},
			},
			args:             args{&Release{TorrentName: "The.First.Lady.S01E01.DV.2160p.WEB-DL.DD5.1.H265-GLHF"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},

		{
			name: "test_13",
			fields: fields{
				Resolutions: []string{"2160p"},
				Codecs:      []string{"h.265"},
			},
			args:             args{&Release{TorrentName: "The First Lady S01E01 DV 2160p WEB-DL DD5.1 H 265-GLHF"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_14",
			fields: fields{
				Sources: []string{"WEBRip"},
			},
			args:             args{&Release{TorrentName: "Halt and Catch Fire S04 1080p WEBRip x265-HiQVE"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_15",
			fields: fields{
				Sources: []string{"WEB"},
			},
			args:             args{&Release{TorrentName: "Dominik Walter-Cocktail Girl-(NS1083)-WEB-2022-AFO"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_16",
			fields: fields{
				Sources: []string{"ViNYL"},
			},
			args:             args{&Release{TorrentName: "Love Unlimited - Under the Influence of Love Unlimited [1973] [Album] - MP3 / V0 (VBR) / Vinyl"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_17",
			fields: fields{
				Resolutions: []string{"1080p"},
				Sources:     []string{"BluRay"},
			},
			args:             args{&Release{TorrentName: "A Movie [2015] - GROUP", ReleaseTags: "Type: Movie / 1080p / Encode / Freeleech: 100 Size: 7.00GB"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "source", got: "", want: []string{"BluRay"}}}},
			wantMatch:        false,
		},
		{
			name: "test_18",
			fields: fields{
				Resolutions: []string{"2160p"},
			},
			args:             args{&Release{TorrentName: "The Green Mile [1999] - playBD", ReleaseTags: "Type: Movie / 2160p / Remux / Freeleech: 100 Size: 72.78GB"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_19",
			fields: fields{
				Shows:       "Reacher",
				Seasons:     "1",
				Episodes:    "0",
				Resolutions: []string{"2160p"},
				Sources:     []string{"WEB-DL"},
				Codecs:      []string{"x265"},
			},
			args:             args{&Release{TorrentName: "Preacher.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "shows", got: "Preacher", want: "Reacher"}}},
			wantMatch:        false,
		},
		{
			name: "test_20",
			fields: fields{
				Shows:       "Atlanta",
				Resolutions: []string{"1080p"},
				Sources:     []string{"WEB-DL", "WEB"},
			},
			args:             args{&Release{TorrentName: "NBA.2022.04.19.Atlanta.Hawks.vs.Miami.Heat.1080p.WEB.H264-SPLASH"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "shows", got: "NBA", want: "Atlanta"}}},
			wantMatch:        false,
		},
		{
			name: "test_21",
			fields: fields{
				Formats: []string{"FLAC"},
				Quality: []string{"Lossless"},
				Media:   []string{"CD"},
				Log:     true,
				//LogScore: 100,
				Cue: true,
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_22",
			fields: fields{
				PerfectFlac: true,
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_23",
			fields: fields{
				Origins: []string{"Internal"},
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene", Origin: "Internal"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_24",
			fields: fields{
				Origins: []string{"P2P"},
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene", Origin: "Internal"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match origin", got: "Internal", want: []string{"P2P"}}}},
			wantMatch:        false,
		},
		{
			name: "test_25",
			fields: fields{
				Origins: []string{"O-SCENE"},
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene", Origin: "SCENE"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match origin", got: "SCENE", want: []string{"O-SCENE"}}}},
			wantMatch:        false,
		},
		{
			name: "test_26",
			fields: fields{
				Origins: []string{"SCENE"},
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene", Origin: "O-SCENE"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match origin", got: "O-SCENE", want: []string{"SCENE"}}}},
			wantMatch:        false,
		},
		{
			name: "test_26",
			fields: fields{
				Origins: []string{"SCENE"},
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_27",
			fields: fields{
				UseRegex:      true,
				MatchReleases: ".*1080p.+(group1|group3)",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match releases: REGEX", got: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", want: ".*1080p.+(group1|group3)"}}},
			wantMatch:        false,
		},
		{
			name: "test_28",
			fields: fields{
				UseRegex:      true,
				MatchReleases: ".*2160p.+(group1|group2)",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_29",
			fields: fields{
				UseRegex:      true,
				MatchReleases: "*2160p*",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match releases: REGEX", got: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", want: "*2160p*"}}},
			wantMatch:        false,
		},
		{
			name: "test_30",
			fields: fields{
				UseRegex:      true,
				MatchReleases: "2160p",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_31",
			fields: fields{
				UseRegex:      false,
				MatchReleases: "*2160p*",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_32",
			fields: fields{
				MaxDownloads:     1,
				MaxDownloadsUnit: FilterMaxDownloadsMonth,
				Downloads: &FilterDownloads{
					MonthCount: 0,
				},
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_33",
			fields: fields{
				MaxDownloads:     10,
				MaxDownloadsUnit: FilterMaxDownloadsMonth,
				Downloads: &FilterDownloads{
					TotalCount: 10,
					MonthCount: 10,
				},
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "max downloads", got: "Hour: 0, Day: 0, Week: 0, Month: 10, Total: 10", want: "reached 10 per MONTH", format: "[max downloads] reached 10 per MONTH"}}},
			wantMatch:        false,
		},
		{
			name: "test_34",
			fields: fields{
				MaxDownloads:     10,
				MaxDownloadsUnit: FilterMaxDownloadsMonth,
				Downloads: &FilterDownloads{
					TotalCount: 50,
					MonthCount: 50,
				},
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "max downloads", got: "Hour: 0, Day: 0, Week: 0, Month: 50, Total: 50", want: "reached 10 per MONTH", format: "[max downloads] reached 10 per MONTH"}}},
			wantMatch:        false,
		},
		{
			name: "test_35",
			fields: fields{
				MaxDownloads:     15,
				MaxDownloadsUnit: FilterMaxDownloadsHour,
				Downloads: &FilterDownloads{
					TotalCount: 50,
					MonthCount: 50,
					WeekCount:  50,
					DayCount:   25,
					HourCount:  20,
				},
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "max downloads", got: "Hour: 20, Day: 25, Week: 50, Month: 50, Total: 50", want: "reached 15 per HOUR", format: "[max downloads] reached 15 per HOUR"}}},
			wantMatch:        false,
		},
		{
			name: "test_36",
			fields: fields{
				MaxDownloads:     15,
				MaxDownloadsUnit: FilterMaxDownloadsHour,
				Downloads: &FilterDownloads{
					HourCount:  14,
					MonthCount: 50,
				},
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_37",
			fields: fields{
				ExceptOrigins: []string{"Internal"},
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", Origin: "Internal"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "except origin", got: "Internal", want: []string{"Internal"}}}},
			wantMatch:        false,
		},
		{
			name: "test_38",
			fields: fields{
				ExceptOrigins: []string{"Internal"},
			},
			args:             args{&Release{TorrentName: "Gillan - Future Shock", Origin: "Scene"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_39",
			fields: fields{
				UseRegexReleaseTags: true,
				MatchReleaseTags:    ".*1080p.+(group1|group3)",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", ReleaseTags: "MKV | x264 | WEB | P2P"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match release tags: REGEX", got: "MKV | x264 | WEB | P2P", want: ".*1080p.+(group1|group3)"}}},
			wantMatch:        false,
		},
		{
			name: "test_40",
			fields: fields{
				UseRegexReleaseTags: true,
				MatchReleaseTags:    "foreign - 16",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", ReleaseTags: "MKV | x264 | WEB | P2P | Foreign - 17"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match release tags: REGEX", got: "MKV | x264 | WEB | P2P | Foreign - 17", want: "foreign - 16"}}},
			wantMatch:        false,
		},
		{
			name: "test_41",
			fields: fields{
				UseRegexReleaseTags: true,
				MatchReleaseTags:    "foreign - 17",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", ReleaseTags: "MKV | x264 | WEB | P2P | Foreign - 17"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_42",
			fields: fields{
				UseRegexReleaseTags: true,
				MatchReleaseTags:    "foreign - 17",
			},
			args:             args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", ReleaseTags: ""}},
			rejectionReasons: &RejectionReasons{data: []Rejection{{key: "match release tags: REGEX", got: "", want: "foreign - 17"}}},
			wantMatch:        false,
		},
		{
			name: "test_43",
			fields: fields{
				Shows:       ",Dutchess, preacher",
				Seasons:     "1",
				Episodes:    "0",
				Resolutions: []string{"2160p"},
				Sources:     []string{"WEB-DL"},
				Codecs:      []string{"x265"},
			},
			args:             args{&Release{TorrentName: "Preacher.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
		{
			name: "test_44",
			fields: fields{
				MatchDescription: "*black?metal*",
			},
			args:             args{&Release{Description: "dog\ncat\r\nblack metalo\negg"}},
			rejectionReasons: &RejectionReasons{data: []Rejection{}},
			wantMatch:        true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Filter{
				ID:                   tt.fields.ID,
				Name:                 tt.fields.Name,
				Enabled:              tt.fields.Enabled,
				CreatedAt:            tt.fields.CreatedAt,
				UpdatedAt:            tt.fields.UpdatedAt,
				MinSize:              tt.fields.MinSize,
				MaxSize:              tt.fields.MaxSize,
				Delay:                tt.fields.Delay,
				Priority:             tt.fields.Priority,
				MaxDownloads:         tt.fields.MaxDownloads,
				MaxDownloadsUnit:     tt.fields.MaxDownloadsUnit,
				MatchReleases:        tt.fields.MatchReleases,
				ExceptReleases:       tt.fields.ExceptReleases,
				UseRegex:             tt.fields.UseRegex,
				MatchReleaseGroups:   tt.fields.MatchReleaseGroups,
				ExceptReleaseGroups:  tt.fields.ExceptReleaseGroups,
				MatchReleaseTags:     tt.fields.MatchReleaseTags,
				ExceptReleaseTags:    tt.fields.ExceptReleaseTags,
				UseRegexReleaseTags:  tt.fields.UseRegexReleaseTags,
				MatchDescription:     tt.fields.MatchDescription,
				ExceptDescription:    tt.fields.ExceptDescription,
				UseRegexDescription:  tt.fields.UseRegexDescription,
				Scene:                tt.fields.Scene,
				Origins:              tt.fields.Origins,
				ExceptOrigins:        tt.fields.ExceptOrigins,
				Freeleech:            tt.fields.Freeleech,
				FreeleechPercent:     tt.fields.FreeleechPercent,
				Shows:                tt.fields.Shows,
				Seasons:              tt.fields.Seasons,
				Episodes:             tt.fields.Episodes,
				Resolutions:          tt.fields.Resolutions,
				Codecs:               tt.fields.Codecs,
				Sources:              tt.fields.Sources,
				Containers:           tt.fields.Containers,
				MatchHDR:             tt.fields.MatchHDR,
				ExceptHDR:            tt.fields.ExceptHDR,
				Years:                tt.fields.Years,
				Months:               tt.fields.Months,
				Days:                 tt.fields.Days,
				Artists:              tt.fields.Artists,
				Albums:               tt.fields.Albums,
				MatchReleaseTypes:    tt.fields.MatchReleaseTypes,
				ExceptReleaseTypes:   tt.fields.ExceptReleaseTypes,
				Formats:              tt.fields.Formats,
				Quality:              tt.fields.Quality,
				Media:                tt.fields.Media,
				PerfectFlac:          tt.fields.PerfectFlac,
				Cue:                  tt.fields.Cue,
				Log:                  tt.fields.Log,
				LogScore:             tt.fields.LogScore,
				MatchOther:           tt.fields.MatchOther,
				ExceptOther:          tt.fields.ExceptOther,
				MatchCategories:      tt.fields.MatchCategories,
				ExceptCategories:     tt.fields.ExceptCategories,
				MatchUploaders:       tt.fields.MatchUploaders,
				ExceptUploaders:      tt.fields.ExceptUploaders,
				Tags:                 tt.fields.Tags,
				ExceptTags:           tt.fields.ExceptTags,
				TagsMatchLogic:       tt.fields.TagsMatchLogic,
				ExceptTagsMatchLogic: tt.fields.ExceptTagsMatchLogic,
				Actions:              tt.fields.Actions,
				Indexers:             tt.fields.Indexers,
				Downloads:            tt.fields.Downloads,
			}

			f.Sanitize()
			tt.args.r.ParseString(tt.args.r.TorrentName)
			rejections, match := f.CheckFilter(tt.args.r)
			assert.Equalf(t, tt.rejectionReasons, rejections, "CheckFilter(%v)", tt.args.r)
			assert.Equalf(t, tt.wantMatch, match, "CheckFilter(%v)", tt.args.r)
		})
	}
}

func Test_containsMatch(t *testing.T) {
	type args struct {
		tags    []string
		filters []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{tags: []string{"HDR", "DV"}, filters: []string{"DV"}}, want: true},
		{name: "test_2", args: args{tags: []string{"HDR", "DV"}, filters: []string{"HD*", "D*"}}, want: true},
		{name: "test_3", args: args{tags: []string{"HDR"}, filters: []string{"DV"}}, want: false},
		{name: "test_4", args: args{tags: []string{"HDR"}, filters: []string{"TEST*"}}, want: false},
		{name: "test_5", args: args{tags: []string{""}, filters: []string{"test,"}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, containsMatch(tt.args.tags, tt.args.filters), "containsMatch(%v, %v)", tt.args.tags, tt.args.filters)
		})
	}
}

func Test_containsAllMatch(t *testing.T) {
	type args struct {
		tags    []string
		filters []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{tags: []string{"HDR", "DV"}, filters: []string{"DV"}}, want: true},
		{name: "test_2", args: args{tags: []string{"HDR", "DV"}, filters: []string{"DV", "DoVI"}}, want: false},
		{name: "test_3", args: args{tags: []string{"HDR", "DV", "DoVI"}, filters: []string{"DV", "DoVI"}}, want: true},
		{name: "test_4", args: args{tags: []string{"HDR", "DV"}, filters: []string{"HD*", "D*"}}, want: true},
		{name: "test_5", args: args{tags: []string{"HDR", "DV"}, filters: []string{"HD*", "TEST*"}}, want: false},
		{name: "test_6", args: args{tags: []string{"HDR"}, filters: []string{"DV"}}, want: false},
		{name: "test_7", args: args{tags: []string{""}, filters: []string{"test,"}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, containsAllMatch(tt.args.tags, tt.args.filters), "containsAllMatch(%v, %v)", tt.args.tags, tt.args.filters)
		})
	}
}

func Test_contains(t *testing.T) {
	type args struct {
		tag    string
		filter string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{tag: "group1", filter: "group1,group2"}, want: true},
		{name: "test_2", args: args{tag: "group1", filter: "group1,group2"}, want: true},
		{name: "test_3", args: args{tag: "group1", filter: "group2,group3"}, want: false},
		{name: "test_4", args: args{tag: "something test something", filter: "*test*"}, want: true},
		{name: "test_5", args: args{tag: "something.test.something", filter: "*test*"}, want: true},
		{name: "test_6", args: args{tag: "that movie", filter: "that?movie"}, want: true},
		{name: "test_7", args: args{tag: "that.movie", filter: "that?movie"}, want: true},
		{name: "test_8", args: args{tag: "", filter: "that?movie,"}, want: false},
		{name: "test_9", args: args{tag: "", filter: ""}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, contains(tt.args.tag, tt.args.filter), "contains(%v, %v)", tt.args.tag, tt.args.filter)
		})
	}
}

func Test_containsSlice(t *testing.T) {
	type args struct {
		tag     string
		filters []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{tag: "group1", filters: []string{"group1", "group2"}}, want: true},
		{name: "test_2", args: args{tag: "group1", filters: []string{"group2", "group3"}}, want: false},
		{name: "test_3", args: args{tag: "2160p", filters: []string{"1080p", "2160p"}}, want: true},
		{name: "test_4", args: args{tag: "", filters: []string{""}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, containsSlice(tt.args.tag, tt.args.filters), "containsSlice(%v, %v)", tt.args.tag, tt.args.filters)
		})
	}
}

func Test_containsAny(t *testing.T) {
	type args struct {
		tags   []string
		filter string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{tags: []string{"HDR", "DV"}, filter: "DV"}, want: true},
		{name: "test_2", args: args{tags: []string{"HDR"}, filter: "DV"}, want: false},
		{name: "test_3", args: args{tags: []string{""}, filter: "test,"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, containsAny(tt.args.tags, tt.args.filter), "containsAny(%v, %v)", tt.args.tags, tt.args.filter)
		})
	}
}

func Test_containsAll(t *testing.T) {
	type args struct {
		tags   []string
		filter string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{tags: []string{"HDR", "DV"}, filter: "DV"}, want: true},
		{name: "test_2", args: args{tags: []string{"HDR", "DV"}, filter: "HDR,DV"}, want: true},
		{name: "test_2", args: args{tags: []string{"HDR", "DV"}, filter: "HD*,D*"}, want: true},
		{name: "test_3", args: args{tags: []string{"HDR", "DoVI"}, filter: "HDR,DV"}, want: false},
		{name: "test_4", args: args{tags: []string{"HDR", "DV", "HDR10+"}, filter: "HDR,DV"}, want: true},
		{name: "test_5", args: args{tags: []string{"HDR"}, filter: "DV"}, want: false},
		{name: "test_6", args: args{tags: []string{""}, filter: "test,"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, containsAll(tt.args.tags, tt.args.filter), "containsAll(%v, %v)", tt.args.tags, tt.args.filter)
		})
	}
}

func Test_sliceContainsSlice(t *testing.T) {
	type args struct {
		tags    []string
		filters []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{tags: []string{"HDR", "DV"}, filters: []string{"HDR", "DoVi"}}, want: true},
		{name: "test_2", args: args{tags: []string{"HDR10+", "DV"}, filters: []string{"HDR"}}, want: false},
		{name: "test_3", args: args{tags: []string{""}, filters: []string{"test,"}}, want: false},
		{name: "test_4", args: args{tags: []string{""}, filters: []string{","}}, want: false},
		{name: "test_5", args: args{tags: []string{""}, filters: []string{""}}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, sliceContainsSlice(tt.args.tags, tt.args.filters), "sliceContainsSlice(%v, %v)", tt.args.tags, tt.args.filters)
		})
	}
}

func Test_containsIntStrings(t *testing.T) {
	type args struct {
		value      int
		filterList string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{value: 2, filterList: "1,2,3"}, want: true},
		{name: "test_2", args: args{value: 2, filterList: "1-3"}, want: true},
		{name: "test_3", args: args{value: 2, filterList: "2"}, want: true},
		{name: "test_4", args: args{value: 2, filterList: "2-5"}, want: true},
		{name: "test_5", args: args{value: 2, filterList: "3-5"}, want: false},
		{name: "test_6", args: args{value: 2, filterList: "3-5"}, want: false},
		{name: "test_7", args: args{value: 0, filterList: "3-5"}, want: false},
		{name: "test_8", args: args{value: 0, filterList: "0"}, want: true},
		{name: "test_9", args: args{value: 100, filterList: "1-1000"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, containsIntStrings(tt.args.value, tt.args.filterList), "containsIntStrings(%v, %v)", tt.args.value, tt.args.filterList)
		})
	}
}

func Test_matchRegex(t *testing.T) {
	type args struct {
		tag    string
		filter string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "test_1", args: args{tag: "Some.show.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP1", filter: ".*2160p.+(group1|group2)"}, want: true},
		{name: "test_2", args: args{tag: "Some.show.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", filter: ".*1080p.+(group1|group3)"}, want: false},
		{name: "test_3", args: args{tag: "Some.show.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", filter: ".*1080p.+(group1|group3),.*2160p.+"}, want: true},
		{name: "test_4", args: args{tag: "Some.show.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", filter: ".*1080p.+(group1|group3),.*720p.+"}, want: false},
		{name: "test_5", args: args{tag: "Some.show.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2", filter: ".*1080p.+(group1|group3),.*720p.+,"}, want: false},
		{name: "test_6", args: args{tag: "[Group] -Name of a Novel Something Good-  [2012][Translated (Group)][EPUB]", filter: "(?:.*Something Good.*|.*Something Bad.*)"}, want: true},
		{name: "test_7", args: args{tag: "[Group] -Name of a Novel Something Good-  [2012][Translated (Group)][EPUB]", filter: "(?:.*Something Funny.*|.*Something Bad.*)"}, want: false},
		{name: "test_8", args: args{tag: ".s10E123.", filter: `\.[Ss]\d{1,2}[Ee]\d{1,3}\.`}, want: true},
		{name: "test_9", args: args{tag: "S1E1", filter: `\.[Ss]\d{1,2}[Ee]\d{1,3}\.`}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, matchRegex(tt.args.tag, tt.args.filter), "matchRegex(%v, %v)", tt.args.tag, tt.args.filter)
		})
	}
}

func Test_validation(t *testing.T) {
	tests := []struct {
		name   string
		filter Filter
		valid  bool
	}{
		{name: "empty name", filter: Filter{}, valid: false},
		{name: "empty filter, with name", filter: Filter{Name: "test"}, valid: true},
		{name: "valid size limit", filter: Filter{Name: "test", MaxSize: "12MB"}, valid: true},
		{name: "gibberish max size limit", filter: Filter{Name: "test", MaxSize: "asdf"}, valid: false},
		{name: "gibberish min size limit", filter: Filter{Name: "test", MinSize: "qwerty"}, valid: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.valid, tt.filter.Validate() == nil, "validation error \"%+v\" in test case %s", tt.filter.Validate(), tt.filter.Name)
		})
	}
}

func Test_checkSizeFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      Filter
		releaseSize uint64
		want        bool
		wantErr     string
	}{
		{name: "test_1", filter: Filter{MinSize: "1GB", MaxSize: ""}, releaseSize: 100, want: false},
		{name: "test_2", filter: Filter{MinSize: "1GB", MaxSize: ""}, releaseSize: 2000000000, want: true},
		{name: "test_3", filter: Filter{MinSize: "1GB", MaxSize: "2.2GB"}, releaseSize: 2000000000, want: true},
		{name: "test_4", filter: Filter{MinSize: "1GB", MaxSize: "2GIB"}, releaseSize: 2000000000, want: true},
		{name: "test_5", filter: Filter{MinSize: "1GB", MaxSize: "2GB"}, releaseSize: 2000000010, want: false},
		{name: "test_6", filter: Filter{MinSize: "1GB", MaxSize: "2GB"}, releaseSize: 2000000000, want: false},
		{name: "test_7", filter: Filter{MaxSize: "2GB"}, releaseSize: 2500000000, want: false},
		{name: "test_8", filter: Filter{MaxSize: "20GB"}, releaseSize: 2500000000, want: true},
		{name: "test_9", filter: Filter{MinSize: "unparseable", MaxSize: "20GB"}, releaseSize: 2500000000, want: false, wantErr: "could not parse filter min size: strconv.ParseFloat: parsing \"\": invalid syntax"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.filter.RejectReasons = NewRejectionReasons()
			got, err := tt.filter.CheckReleaseSize(tt.releaseSize)
			if tt.wantErr != "" && assert.Error(t, err) {
				assert.EqualErrorf(t, err, tt.wantErr, "Error should be: %v, got: %v", tt.wantErr, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_containsFuzzy(t *testing.T) {
	type args struct {
		tag    string
		filter string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "", args: args{tag: "this is a long text that should contain some random data one of them being black metal which should be in the middle of everything.", filter: "*black?metal*"}, want: true},
		{
			name: "",
			args: args{
				tag:    "Kategori: Music \\n Storlek: 132.78 MiB\\n                                       ?                                     ??                                     ??                                 ?????????                                     ??                                     ??                            ??       ??       ??           ????              ??      ??      ??              ????        ???     ?            ??      ??      ??            ?     ???      ?           ?         ??       ?       ??         ?           ???  ?                     ??       ??       ???                     ??   ?? ????                      ????      ??      ????                      ?????   ?????  ? ??           ??  ????    ??    ????  ??           ?? ?  ??????      ?????????????            ????????????            ?????????????                    ??????         ????         ??????     ?     ?? ???   ?   ??         ?         ??    ??   ??? ??     ?      ?     ??????  ??   ??          ??          ??    ??  ?????     ?       ??         ??                 ??                  ??         ??         ?????????     ?          ??????         ??      ?????????            ???         ?          ??????          ?          ??             ?         ??            ??           ???         ?             ?        ?              ??            ??        ?      ???    ?       ??               ?    ?         ?? ?????       ?        ??         ?   ?   ??         ??       ??? ???                   ???      ??    ??? ??      ???       ???       ?  ?????????              ????    ??     ?????    ? ??        ?????????????  ???????????            ???   ????     ???     ??               ??????????   ??       ??            ???    ??      ???     ???          ???? ??? ?  ??  ???        ?    ??  ?????????? ??  ??????????????? ????????????? ????    ????   ?????   ??? ??????? ???????? ????? ???????? ????? ?????    ??  ???      ???  ?   ?    ????   ???? ????      ???? ????     ???? ????    ??   ??       ???  ???        ???    ??? ???        ??? ???       ??? ???    ??    ??     ???  ???        ??    ??? ???        ??? ???       ??? ????????  ?    ??     ???   ???       ???    ??? ???        ??? ???       ??? ???      ???  ??    ???    ???      ???    ??? ???        ??? ???       ??? ???  ??????  ???    ???    ????     ??? ? ??? ???      ??? ??? ?   ??? ????   ?      ?   ???   ????      ????    ?? ??          ?? ??         ?? ??     ??   ??????? ????  ????    ?  ??      ? ?            ? ?           ? ?      ??????    ???????   ?????     ??       ? ?            ? ?           ? ?        ??      ??????     ????? ??         ? ?            ? ?           ? ?   sM!iMPURE    ???       ????           ? ?            ? ?           ? ?   ./\\\\\\\\//\\\\\\\\.   ????        ??             ?              ?             ?                 ??        ?                                                              ?        ?                                                              ?                     ??????????????????????????????????????   ??????????????????                                      ?????????????????? ??? ??                                                                  ?? ??? ???                       RELEASE INFORMATION for:                       ??? ??          Portae_Obscuritas-Sapientia_Occulta-WEB-2024-ENTiTLED           ?? ??                                                                          ?? ?   artist........ | Portae Obscuritas                                       ?     title......... | Sapientia Occulta     label......... | 6868317 Records DK     genre......... | Black Metal     url............| https://www.deezer.com/album/568832371     rip date...... | 2024-10-17     retail date... | 2024-04-03     runtime....... | 55:41     tracks........ | 7     size.......... | 132.14MB     source........ | WEB     quality....... | CBR 320kbps 44.1kHz Stereo ?   codec......... | MP3 (MPEG-1 Audio Layer 3)                               ? ??  encoder....... | LAME                                                    ?? ??                                                                           ?? ??                                                                        ?? ????                                                                      ????   ??????????????????                                      ?????????????????? ??? ?               ??????????????????????????????????????               ? ??   ??                                                                      ?? ????             -------------------------------------------              ???? ???                     ? ?\\u003c t r a c k . l i s t \\u003e? ?                      ??? ??              -------------------------------------------               ??? ??                                                                          ?? ??                                                                          ??     01 \\u003e Intro                                                     \\u003c 05:12     02 \\u003e In a Twilight Obscurity                                   \\u003c 09:15     03 \\u003e Manifestation of Acheronian Trinity                       \\u003c 09:14     04 \\u003e Imperious Reverent Transcendence                          \\u003c 08:59     05 \\u003e Enslaved Spirit of Forgotten Kingdoms                     \\u003c 11:48     06 \\u003e Sapientia Occulta                                         \\u003c 08:24     07 \\u003e Outro                                                     \\u003c 02:49 ?                                                                            ? ??                                                                          ?? ??                                                                          ?? ??                                                                        ?? ????                                                                      ???? ? ??????????????????                                      ?????????????????? ? ???                ??????????????????????????????????????                ??? ???                                                                        ??? ?                             ? ?\\u003c GREETINGS \\u003e? ?                            ? ?%                                                                          %? ??                                                                          ?? ?%                                                                          %? ??    Shout out to all of those who keep the dream of the scene alive.      ?? ?%                                                                          %? ??          Special thanks to those who have paved the way and parted.      ?? ?%                                                                          %? ??                                                        We miss you!      ?? ?%                                                                          %? ??                                                                          ?? ?%                                                                          %? ??                                                                          ?? ?%                                                                          %? ??                                                                          ?? ?%                                                                          %? ??                                                                          ?? ?%                               ???????????                                %? ???                        ?????? ????????? ??????                         ??? ???              ? ??????????????    ?    ??????????????? ?              ??? ? ???????????????? ??????????????  ?????  ????????????   ????????????????? ? ???????       ???????       ????? ??? ? ??? ??????       ???????       ??????? ??   ????????   ?     ??????    ???? ? ????     ??????     ?    ???????   ?? ??           ???????????             ???              ???????????           ?? ??               ??        +          ?           +        ??               ?? ?                           ????      o      ?????                           ?                                 ????????????? ?                                                                            ?  ?                                                                          ?  ???                                                                      ???  ?  ?                                                                    ?  ? ?   ?                                                                    ?   ? ????                                                                      ????",
				filter: "dark?metal,*black?metal*,gray?metal",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, containsFuzzy(tt.args.tag, tt.args.filter), "containsFuzzy(%v, %v)", tt.args.tag, tt.args.filter)
		})
	}
}
