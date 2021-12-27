package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRelease_Parse(t *testing.T) {
	tests := []struct {
		name    string
		fields  Release
		want    Release
		wantErr bool
	}{
		{
			name: "parse_1",
			fields: Release{
				TorrentName: "Servant S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-FLUX",
			},
			want: Release{
				TorrentName: "Servant S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-FLUX",
				Clean:       "Servant S01 2160p ATVP WEB DL DDP 5 1 Atmos DV HEVC FLUX",
				Season:      1,
				Episode:     0,
				Resolution:  "2160p",
				Source:      "WEB-DL",
				Codec:       "HEVC",
				HDR:         "DV",
				Audio:       "DDP 5.1 Atmos",
				Group:       "FLUX",
				Website:     "ATVP",
			},
			wantErr: false,
		},
		{
			name: "parse_2",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
			},
			want: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				Clean:       "Servant S01 2160p ATVP WEB DL DDP 5 1 Atmos DV HEVC FLUX",
				Season:      1,
				Episode:     0,
				Resolution:  "2160p",
				Source:      "WEB-DL",
				Codec:       "HEVC",
				HDR:         "DV",
				Audio:       "DDP.5.1", // need to fix audio parsing
				Group:       "FLUX",
				Website:     "ATVP",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields
			if err := r.Parse(); (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.want, r)
		})
	}
}

func TestRelease_CheckFilter(t *testing.T) {
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields // Release

			_ = r.Parse() // Parse TorrentName into struct
			got := r.CheckFilter(tt.args.filter)

			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRelease_MapVars(t *testing.T) {
	type args struct {
		varMap map[string]string
	}
	tests := []struct {
		name   string
		fields *Release
		want   *Release
		args   args
	}{
		{
			name:   "1",
			fields: &Release{},
			want:   &Release{TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2"},
			args: args{varMap: map[string]string{
				"torrentName": "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
			}},
		},
		{
			name:   "2",
			fields: &Release{},
			want: &Release{
				TorrentName: "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:    "tv",
				Freeleech:   true,
				Uploader:    "Anon",
				Size:        uint64(10000000000),
			},
			args: args{varMap: map[string]string{
				"torrentName": "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				"category":    "tv",
				"freeleech":   "freeleech",
				"uploader":    "Anon",
				"torrentSize": "10GB",
			}},
		},
		{
			name:   "3",
			fields: &Release{},
			want: &Release{
				TorrentName:      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:         "tv",
				FreeleechPercent: 100,
				Uploader:         "Anon",
				Size:             uint64(10000000000),
			},
			args: args{varMap: map[string]string{
				"torrentName":      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				"category":         "tv",
				"freeleechPercent": "100%",
				"uploader":         "Anon",
				"torrentSize":      "10GB",
			}},
		},
		{
			name:   "4",
			fields: &Release{},
			want: &Release{
				TorrentName:      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:         "tv",
				FreeleechPercent: 100,
				Uploader:         "Anon",
				Size:             uint64(10000000000),
				Tags:             []string{"foreign", "tv"},
			},
			args: args{varMap: map[string]string{
				"torrentName":      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				"category":         "tv",
				"freeleechPercent": "100%",
				"uploader":         "Anon",
				"torrentSize":      "10GB",
				"tags":             "foreign,tv",
			}},
		},
		{
			name:   "5",
			fields: &Release{},
			want: &Release{
				TorrentName:      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:         "tv",
				FreeleechPercent: 100,
				Uploader:         "Anon",
				Size:             uint64(10000000000),
				Tags:             []string{"foreign", "tv"},
			},
			args: args{varMap: map[string]string{
				"torrentName":      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				"category":         "tv",
				"freeleechPercent": "100%",
				"uploader":         "Anon",
				"torrentSize":      "10GB",
				"tags":             "foreign,tv",
			}},
		},
		{
			name:   "6",
			fields: &Release{},
			want: &Release{
				TorrentName:      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:         "tv",
				Year:             2020,
				FreeleechPercent: 100,
				Uploader:         "Anon",
				Size:             uint64(10000000000),
				Tags:             []string{"foreign", "tv"},
			},
			args: args{varMap: map[string]string{
				"torrentName":      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				"category":         "tv",
				"year":             "2020",
				"freeleechPercent": "100%",
				"uploader":         "Anon",
				"torrentSize":      "10GB",
				"tags":             "foreign, tv",
			}},
		},
		{
			name:   "7",
			fields: &Release{},
			want: &Release{
				TorrentName:      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:         "tv",
				Year:             2020,
				FreeleechPercent: 100,
				Uploader:         "Anon",
				Size:             uint64(10000000000),
				Tags:             []string{"hip.hop", "rhythm.and.blues", "2000s"},
			},
			args: args{varMap: map[string]string{
				"torrentName":      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				"category":         "tv",
				"year":             "2020",
				"freeleechPercent": "100%",
				"uploader":         "Anon",
				"torrentSize":      "10GB",
				"tags":             "hip.hop,rhythm.and.blues, 2000s",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields
			_ = r.MapVars(tt.args.varMap)

			assert.Equal(t, tt.want, r)
		})
	}
}
