package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilter_CheckFilter(t *testing.T) {
	type args struct {
		filter     Filter
		rejections []string
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
				rejections: []string{"category not matching. got: Movies want: *tv*"},
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
				rejections: []string{"season not matching. got: 2 want: 1"},
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
				rejections: []string{"unwanted uploaders. got: Anonymous unwanted: Anonymous"},
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
				rejections: []string{"tags not matching. got: [foreign] want: tv"},
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
				rejections: []string{"tags unwanted. got: [foreign] want: foreign"},
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
				rejections: []string{"release groups not matching. got: GROUP want: ift"},
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
				rejections: []string{"except releases: unwanted release. got: Good show shift S02 NORDiC 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP want: Good show shift"},
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
				rejections: []string{"except releases: unwanted release. got: Good show shift S02 NORDiC 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP want: NORDiC"},
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
				rejections: []string{"except releases: unwanted release. got: Good show shift S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP want: NORDiC,*shift*"},
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
				rejections: []string{"hdr unwanted. got: [DV] want: [DV HDR DoVi]"},
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
				rejections: []string{"hdr not matching. got: [] want: [DV HDR DoVi]"},
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
					Cue:             true,
					//LogScore:        100,
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
					PerfectFlac:     true,
				},
				rejections: []string{"wanted: perfect flac. got: [320 MP3]"},
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
					PerfectFlac:     true,
				},
				rejections: []string{"wanted: perfect flac. got: [FLAC Lossless Log100 Log]"},
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
				rejections: []string{"quality not matching. got: [FLAC Lossless Log100 Log] want: [24bit Lossless]", "wanted: cue", "log score. got: 0 want: 100"},
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
					//LogScore:          100,
					Cue: true,
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
				rejections: []string{"release type not matching. got: Album want: [Single]", "log score. got: 0 want: 100"},
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
				rejections: []string{"artists not matching. got: Artist - Albumname want: Artiiiist", "log score. got: 0 want: 100"},
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
					//LogScore:          100,
					Cue: true,
				},
			},
			want: true,
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
				rejections: []string{"wanted: freeleech"},
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
			assert.Equal(t, tt.args.rejections, rejections)
		})
	}
}

func TestFilter_CheckFilter1(t *testing.T) {
	type fields Filter
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
			wantRejections: []string{"episodes not matching. got: 0 want: 2-8"},
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
			wantRejections: []string{"hdr not matching. got: [DV] want: [HDR]"},
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
			wantRejections: []string{"hdr unwanted. got: [DV] want: [DV HDR]"},
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
			wantRejections: []string{"shows not matching. got: WeCrashed want: WeWork", "hdr unwanted. got: [DV] want: [DV HDR]"},
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
			wantRejections: []string{"shows not matching. got: WeCrashed want: WeWork", "unwanted release group. got: NOSiViD unwanted: NOSiViD", "hdr unwanted. got: [DV] want: [DV HDR]"},
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
			wantRejections: []string{"shows not matching. got: WeCrashed want: WeWork", "unwanted release group. got: NOSiViD unwanted: NOSiViD", "source not matching. got: WEB want: [WEB-DL]", "hdr unwanted. got: [DV] want: [DV HDR]"},
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
			wantRejections: []string{"source not matching. got: WEB-DL want: [WEB]"},
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
			wantRejections: []string{"source not matching. got: BluRay want: [WEB]"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "Stranger Things S02 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-FraMeSToR"}},
			wantRejections: []string{"source not matching. got: UHD.BluRay want: [BluRay]", "except other unwanted. got: [HYBRiD REMUX] unwanted: [REMUX HYBRID]"},
			wantMatch:      false,
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
			wantRejections: []string{"source not matching. got:  want: [BluRay]"},
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
				Episodes:    "0",
				Resolutions: []string{"2160p"},
				Sources:     []string{"WEB-DL"},
				Codecs:      []string{"x265"},
			},
			args:           args{&Release{TorrentName: "Preacher.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}},
			wantRejections: []string{"shows not matching. got: Preacher want: Reacher"},
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
			wantRejections: []string{"shows not matching. got: NBA want: Atlanta"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_22",
			fields: fields{
				PerfectFlac: true,
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_23",
			fields: fields{
				Origins: []string{"Internal"},
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene", Origin: "Internal"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_24",
			fields: fields{
				Origins: []string{"P2P"},
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene", Origin: "Internal"}},
			wantRejections: []string{"origin not matching. got: Internal want: [P2P]"},
			wantMatch:      false,
		},
		{
			name: "test_25",
			fields: fields{
				Origins: []string{"O-SCENE"},
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene", Origin: "SCENE"}},
			wantRejections: []string{"origin not matching. got: SCENE want: [O-SCENE]"},
			wantMatch:      false,
		},
		{
			name: "test_26",
			fields: fields{
				Origins: []string{"SCENE"},
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene", Origin: "O-SCENE"}},
			wantRejections: []string{"origin not matching. got: O-SCENE want: [SCENE]"},
			wantMatch:      false,
		},
		{
			name: "test_26",
			fields: fields{
				Origins: []string{"SCENE"},
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD / Scene"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_27",
			fields: fields{
				UseRegex:      true,
				MatchReleases: ".*1080p.+(group1|group3)",
			},
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: []string{"match release regex not matching. got: Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2 want: .*1080p.+(group1|group3)"},
			wantMatch:      false,
		},
		{
			name: "test_28",
			fields: fields{
				UseRegex:      true,
				MatchReleases: ".*2160p.+(group1|group2)",
			},
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_29",
			fields: fields{
				UseRegex:      true,
				MatchReleases: "*2160p*",
			},
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: []string{"match release regex not matching. got: Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2 want: *2160p*"},
			wantMatch:      false,
		},
		{
			name: "test_30",
			fields: fields{
				UseRegex:      true,
				MatchReleases: "2160p",
			},
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_31",
			fields: fields{
				UseRegex:      false,
				MatchReleases: "*2160p*",
			},
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: nil,
			wantMatch:      true,
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
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_33",
			fields: fields{
				MaxDownloads:     10,
				MaxDownloadsUnit: FilterMaxDownloadsMonth,
				Downloads: &FilterDownloads{
					MonthCount: 10,
				},
			},
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: []string{"max downloads (10) this (MONTH) reached"},
			wantMatch:      false,
		},
		{
			name: "test_34",
			fields: fields{
				MaxDownloads:     10,
				MaxDownloadsUnit: FilterMaxDownloadsMonth,
				Downloads: &FilterDownloads{
					MonthCount: 50,
				},
			},
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: []string{"max downloads (10) this (MONTH) reached"},
			wantMatch:      false,
		},
		{
			name: "test_35",
			fields: fields{
				MaxDownloads:     15,
				MaxDownloadsUnit: FilterMaxDownloadsHour,
				Downloads: &FilterDownloads{
					HourCount:  20,
					MonthCount: 50,
				},
			},
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: []string{"max downloads (15) this (HOUR) reached"},
			wantMatch:      false,
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
			args:           args{&Release{TorrentName: "Show.Name.S01.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-GROUP2"}},
			wantRejections: nil,
			wantMatch:      true,
		},
		{
			name: "test_37",
			fields: fields{
				ExceptOrigins: []string{"Internal"},
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", Origin: "Internal"}},
			wantRejections: []string{"except origin not matching. got: Internal unwanted: [Internal]"},
			wantMatch:      false,
		},
		{
			name: "test_38",
			fields: fields{
				ExceptOrigins: []string{"Internal"},
			},
			args:           args{&Release{TorrentName: "Gillan - Future Shock", Origin: "Scene"}},
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
				MaxDownloads:        tt.fields.MaxDownloads,
				MaxDownloadsUnit:    tt.fields.MaxDownloadsUnit,
				MatchReleases:       tt.fields.MatchReleases,
				ExceptReleases:      tt.fields.ExceptReleases,
				UseRegex:            tt.fields.UseRegex,
				MatchReleaseGroups:  tt.fields.MatchReleaseGroups,
				ExceptReleaseGroups: tt.fields.ExceptReleaseGroups,
				Scene:               tt.fields.Scene,
				Origins:             tt.fields.Origins,
				ExceptOrigins:       tt.fields.ExceptOrigins,
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
				MatchOther:          tt.fields.MatchOther,
				ExceptOther:         tt.fields.ExceptOther,
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
				Downloads:           tt.fields.Downloads,
			}
			tt.args.r.ParseString(tt.args.r.TorrentName)
			rejections, match := f.CheckFilter(tt.args.r)
			assert.Equalf(t, tt.wantRejections, rejections, "CheckFilter(%v)", tt.args.r)
			assert.Equalf(t, tt.wantMatch, match, "CheckFilter(%v)", tt.args.r)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, matchRegex(tt.args.tag, tt.args.filter), "matchRegex(%v, %v)", tt.args.tag, tt.args.filter)
		})
	}
}
