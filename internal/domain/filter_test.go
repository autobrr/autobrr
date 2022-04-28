package domain

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilter_CheckFilter(t *testing.T) {
	type args struct {
		filter Filter
	}
	tests := []struct {
		name   string
		fields *Release
		args   args
		want   bool
	}{
		{
			name:   "size_between_max_min",
			fields: &Release{Size: uint64(10000000001)},
			args: args{
				filter: Filter{
					Enabled: true,
					MinSize: "10 GB",
					MaxSize: "20GB",
				},
			},
			want: true,
		},
		{
			name:   "size_larger_than_max",
			fields: &Release{Size: uint64(30000000001)},
			args: args{
				filter: Filter{
					Enabled: true,
					MinSize: "10 GB",
					MaxSize: "20GB",
				},
			},
			want: false,
		},
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
			},
			want: false,
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
			},
			want: true,
		},
		{
			name: "match_tags",
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
				},
			},
			want: true,
		},
		{
			name: "match_tags_bad",
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
				},
			},
			want: false,
		},
		{
			name: "match_except_tags",
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
				},
			},
			want: true,
		},
		{
			name: "match_except_tags_2",
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
					ExceptTags:      "foreign",
				},
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
			},
			want: false,
		},
		{
			name: "match_group_potential_partial_3",
			fields: &Release{
				TorrentName: "Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-de[42]",
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
					MatchReleaseGroups: "de[42]",
				},
			},
			want: true,
		},
		{
			name: "match_group_potential_partial_3",
			fields: &Release{
				TorrentName: "[AnimeGroup] Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC",
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
					MatchReleaseGroups: "[AnimeGroup]",
				},
			},
			want: true,
		},
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
			},
			want: false,
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
			},
			want: true,
		},
		{
			name: "match_music_1",
			fields: &Release{
				TorrentName: "Artist - Albumname",
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
					LogScore:        100,
					Cue:             true,
				},
			},
			want: true,
		},
		{
			name: "match_music_2",
			fields: &Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "MP3 / 320 / WEB",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "Album",
					Artists:         "Artist",
					//Sources:         []string{"CD"},
					//Formats:         []string{"FLAC"},
					//Quality:         []string{"24bit Lossless"},
					PerfectFlac: true,
					//Log:             true,
					//LogScore:        100,
					//Cue:             true,
				},
			},
			want: false,
		},
		{
			name: "match_music_3",
			fields: &Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / CD",
				Category:    "Album",
			},
			args: args{
				filter: Filter{
					Enabled:         true,
					MatchCategories: "Album",
					Artists:         "Artist",
					//Sources:         []string{"CD"},
					//Formats:         []string{"FLAC"},
					//Quality:         []string{"24bit Lossless"},
					PerfectFlac: true,
					//Log:             true,
					//LogScore:        100,
					//Cue:             true,
				},
			},
			want: false,
		},
		{
			name: "match_music_4",
			fields: &Release{
				TorrentName: "Artist - Albumname",
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
			},
			want: false,
		},
		{
			name: "match_music_5",
			fields: &Release{
				TorrentName: "Artist - Albumname",
				Year:        2022,
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
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
					Quality:           []string{"24bit Lossless", "Lossless"},
					PerfectFlac:       true,
					Log:               true,
					LogScore:          100,
					Cue:               true,
				},
			},
			want: true,
		},
		{
			name: "match_music_6",
			fields: &Release{
				TorrentName: "Artist - Albumname",
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
			},
			want: false,
		},
		{
			name: "match_music_7",
			fields: &Release{
				TorrentName: "Artist - Albumname",
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
			},
			want: false,
		},
		{
			name: "match_music_8",
			fields: &Release{
				TorrentName: "Artist - Albumname",
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
					LogScore:          100,
					Cue:               true,
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields // Release

			_ = r.Parse() // Parse TorrentName into struct
			rejections, got := tt.args.filter.CheckFilter(r)
			fmt.Println(rejections)

			assert.Equal(t, tt.want, got)

			//assert.Equalf(t, tt.wantRejections, rejections, "CheckFilter(%v)", tt.args.r)
			//assert.Equalf(t, tt.wantMatch, match, "CheckFilter(%v)", tt.args.r)
		})
	}
}

func TestFilter_CheckFilter1(t *testing.T) {
	type fields struct {
		ID                  int
		Name                string
		Enabled             bool
		CreatedAt           time.Time
		UpdatedAt           time.Time
		MinSize             string
		MaxSize             string
		Delay               int
		Priority            int32
		MatchReleases       string
		ExceptReleases      string
		UseRegex            bool
		MatchReleaseGroups  string
		ExceptReleaseGroups string
		Scene               bool
		Origins             string
		Freeleech           bool
		FreeleechPercent    string
		Shows               string
		Seasons             string
		Episodes            string
		Resolutions         []string
		Codecs              []string
		Sources             []string
		Containers          []string
		MatchHDR            []string
		ExceptHDR           []string
		Years               string
		Artists             string
		Albums              string
		MatchReleaseTypes   []string
		ExceptReleaseTypes  string
		Formats             []string
		Quality             []string
		Media               []string
		PerfectFlac         bool
		Cue                 bool
		Log                 bool
		LogScore            int
		MatchCategories     string
		ExceptCategories    string
		MatchUploaders      string
		ExceptUploaders     string
		Tags                string
		ExceptTags          string
		TagsAny             string
		ExceptTagsAny       string
		Actions             []*Action
		Indexers            []Indexer
	}
	type args struct {
		r *Release
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantRejections []string
		wantMatch      bool
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: nil,
			wantMatch:      true,
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"episodes not matching. wanted: 2-8 got: 0"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"hdr not matching. wanted: [HDR] got: [DV]"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"hdr unwanted. [DV HDR] got: [DV]"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"shows not matching", "hdr unwanted. [DV HDR] got: [DV]"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"shows not matching", "unwanted release group. unwanted: NOSiViD got: NOSiViD", "hdr unwanted. [DV HDR] got: [DV]"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"shows not matching", "unwanted release group. unwanted: NOSiViD got: NOSiViD", "source not matching. wanted: [WEB-DL] got: WEB", "hdr unwanted. [DV HDR] got: [DV]"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"source not matching. wanted: [WEB] got: WEB-DL"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "WeCrashed.S01.DV.2160p.Blu-ray.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"source not matching. wanted: [WEB] got: BluRay"},
			wantMatch:      false,
		},
		{
			name: "test_10",
			fields: fields{
				Resolutions: []string{"2160p"},
				Sources:     []string{"BluRay"},
				Codecs:      []string{"x265", "HEVC"},
				MatchHDR:    []string{"DV", "HDR"},
			},
			args:           args{&Release{TorrentName: "Stranger Things S02 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-FraMeSToR"}},
			wantRejections: []string{"source not matching. wanted: [BluRay] got: UHD.BluRay"},
			wantMatch:      false,
		},
		{
			name: "test_10",
			fields: fields{
				Resolutions: []string{"2160p"},
				Sources:     []string{"UHD.BluRay"},
				Codecs:      []string{"x265", "HEVC"},
				MatchHDR:    []string{"DV", "HDR"},
			},
			args:           args{&Release{TorrentName: "Stranger Things S02 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-FraMeSToR"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_11",
			fields: fields{
				Resolutions: []string{"1080p"},
				Sources:     []string{"BluRay"},
				Codecs:      []string{"HEVC"},
				//MatchHDR:    []string{"DV", "HDR"},
			},
			args:           args{&Release{TorrentName: "Food Wars!: Shokugeki no Soma S05 2020 1080p BluRay HEVC 10-Bit DD2.0 Dual Audio -ZR-"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_12",
			fields: fields{
				Resolutions: []string{"2160p"},
				Codecs:      []string{"h.265"},
			},
			args:           args{&Release{TorrentName: "The.First.Lady.S01E01.DV.2160p.WEB-DL.DD5.1.H265-GLHF"}},
			wantRejections: nil,
			wantMatch:      true,
		},

		{
			name: "test_13",
			fields: fields{
				Resolutions: []string{"2160p"},
				Codecs:      []string{"h.265"},
			},
			args:           args{&Release{TorrentName: "The First Lady S01E01 DV 2160p WEB-DL DD5.1 H 265-GLHF"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_14",
			fields: fields{
				Sources: []string{"WEBRip"},
			},
			args:           args{&Release{TorrentName: "Halt and Catch Fire S04 1080p WEBRip x265-HiQVE"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_15",
			fields: fields{
				Sources: []string{"WEB"},
			},
			args:           args{&Release{TorrentName: "Dominik Walter-Cocktail Girl-(NS1083)-WEB-2022-AFO"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_16",
			fields: fields{
				Sources: []string{"ViNYL"},
			},
			args:           args{&Release{TorrentName: "Love Unlimited - Under the Influence of Love Unlimited [1973] [Album] - MP3 / V0 (VBR) / Vinyl"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_17",
			fields: fields{
				Resolutions: []string{"1080p"},
				Sources:     []string{"BluRay"},
			},
			args:           args{&Release{TorrentName: "A Movie [2015] - GROUP", ReleaseTags: "Type: Movie / 1080p / Encode / Freeleech: 100 Size: 7.00GB"}},
			wantRejections: []string{"source not matching. wanted: [BluRay] got: "},
			wantMatch:      false,
		},
		{
			name: "test_18",
			fields: fields{
				Resolutions: []string{"2160p"},
			},
			args:           args{&Release{TorrentName: "The Green Mile [1999] - playBD", ReleaseTags: "Type: Movie / 2160p / Remux / Freeleech: 100 Size: 72.78GB"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_19",
			fields: fields{
				Shows:       "Reacher",
				Seasons:     "1",
				Resolutions: []string{"2160p"},
				Sources:     []string{"WEB-DL"},
				Codecs:      []string{"x265"},
			},
			args:           args{&Release{TorrentName: "Preacher.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"shows not matching"},
			wantMatch:      false,
		},
		{
			name: "test_20",
			fields: fields{
				Shows:       "Atlanta",
				Resolutions: []string{"1080p"},
				Sources:     []string{"WEB-DL", "WEB"},
			},
			args:           args{&Release{TorrentName: "NBA.2022.04.19.Atlanta.Hawks.vs.Miami.Heat.1080p.WEB.H264-SPLASH"}},
			wantRejections: []string{"shows not matching"},
			wantMatch:      false,
		},
		{
			name: "test_20",
			fields: fields{
				Formats:  []string{"FLAC"},
				Quality:  []string{"Lossless"},
				Media:    []string{"CD"},
				Log:      true,
				LogScore: 100,
				Cue:      true,
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_20",
			fields: fields{
				Formats:  []string{"FLAC"},
				Quality:  []string{"Lossless"},
				Media:    []string{"CD"},
				Log:      true,
				LogScore: 100,
				Cue:      true,
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock [1981] [Album] - FLAC / Lossless / Log / 100% / Cue / CD"}},
			wantRejections: nil,
			wantMatch:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := Filter{
				ID:                  tt.fields.ID,
				Name:                tt.fields.Name,
				Enabled:             tt.fields.Enabled,
				CreatedAt:           tt.fields.CreatedAt,
				UpdatedAt:           tt.fields.UpdatedAt,
				MinSize:             tt.fields.MinSize,
				MaxSize:             tt.fields.MaxSize,
				Delay:               tt.fields.Delay,
				Priority:            tt.fields.Priority,
				MatchReleases:       tt.fields.MatchReleases,
				ExceptReleases:      tt.fields.ExceptReleases,
				UseRegex:            tt.fields.UseRegex,
				MatchReleaseGroups:  tt.fields.MatchReleaseGroups,
				ExceptReleaseGroups: tt.fields.ExceptReleaseGroups,
				Scene:               tt.fields.Scene,
				Origins:             tt.fields.Origins,
				Freeleech:           tt.fields.Freeleech,
				FreeleechPercent:    tt.fields.FreeleechPercent,
				Shows:               tt.fields.Shows,
				Seasons:             tt.fields.Seasons,
				Episodes:            tt.fields.Episodes,
				Resolutions:         tt.fields.Resolutions,
				Codecs:              tt.fields.Codecs,
				Sources:             tt.fields.Sources,
				Containers:          tt.fields.Containers,
				MatchHDR:            tt.fields.MatchHDR,
				ExceptHDR:           tt.fields.ExceptHDR,
				Years:               tt.fields.Years,
				Artists:             tt.fields.Artists,
				Albums:              tt.fields.Albums,
				MatchReleaseTypes:   tt.fields.MatchReleaseTypes,
				ExceptReleaseTypes:  tt.fields.ExceptReleaseTypes,
				Formats:             tt.fields.Formats,
				Quality:             tt.fields.Quality,
				Media:               tt.fields.Media,
				PerfectFlac:         tt.fields.PerfectFlac,
				Cue:                 tt.fields.Cue,
				Log:                 tt.fields.Log,
				LogScore:            tt.fields.LogScore,
				MatchCategories:     tt.fields.MatchCategories,
				ExceptCategories:    tt.fields.ExceptCategories,
				MatchUploaders:      tt.fields.MatchUploaders,
				ExceptUploaders:     tt.fields.ExceptUploaders,
				Tags:                tt.fields.Tags,
				ExceptTags:          tt.fields.ExceptTags,
				TagsAny:             tt.fields.TagsAny,
				ExceptTagsAny:       tt.fields.ExceptTagsAny,
				Actions:             tt.fields.Actions,
				Indexers:            tt.fields.Indexers,
			}
			tt.args.r.ParseString(tt.args.r.TorrentName)
			rejections, match := f.CheckFilter(tt.args.r)
			assert.Equalf(t, tt.wantRejections, rejections, "CheckFilter(%v)", tt.args.r)
			assert.Equalf(t, tt.wantMatch, match, "CheckFilter(%v)", tt.args.r)
		})
	}
}
