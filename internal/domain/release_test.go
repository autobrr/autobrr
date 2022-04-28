package domain

import (
	"testing"
	"time"

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
		{
			name: "parse_3",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags: "MKV / 2160p / WEB-DL",
			},
			want: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				Clean:       "Servant S01 2160p ATVP WEB DL DDP 5 1 Atmos DV HEVC FLUX",
				ReleaseTags: "MKV / 2160p / WEB-DL",
				Container:   "MKV",
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
		{
			name: "parse_4",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags: "MKV | 2160p | WEB-DL",
			},
			want: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				Clean:       "Servant S01 2160p ATVP WEB DL DDP 5 1 Atmos DV HEVC FLUX",
				ReleaseTags: "MKV | 2160p | WEB-DL",
				Container:   "MKV",
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
		{
			name: "parse_5",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags: "MP4 | 2160p | WEB-DL",
			},
			want: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				Clean:       "Servant S01 2160p ATVP WEB DL DDP 5 1 Atmos DV HEVC FLUX",
				ReleaseTags: "MP4 | 2160p | WEB-DL",
				Container:   "MP4",
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
		{
			name: "parse_6",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags: "MP4 | 2160p | WEB-DL | Freeleech!",
			},
			want: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				Clean:       "Servant S01 2160p ATVP WEB DL DDP 5 1 Atmos DV HEVC FLUX",
				ReleaseTags: "MP4 | 2160p | WEB-DL | Freeleech!",
				Container:   "MP4",
				Season:      1,
				Episode:     0,
				Resolution:  "2160p",
				Source:      "WEB-DL",
				Codec:       "HEVC",
				HDR:         "DV",
				Audio:       "DDP.5.1", // need to fix audio parsing
				Group:       "FLUX",
				Website:     "ATVP",
				Freeleech:   true,
			},
			wantErr: false,
		},
		{
			name: "parse_music_1",
			fields: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
			},
			want: Release{
				TorrentName: "Artist - Albumname",
				Clean:       "Artist   Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
				Group:       "",
				Audio:       "FLAC",
				Format:      "FLAC",
				Quality:     "Lossless",
				Source:      "CD",
				HasCue:      true,
				HasLog:      true,
				LogScore:    100,
			},
			wantErr: false,
		},
		{
			name: "parse_music_2",
			fields: Release{
				TorrentName: "Various Artists - Music '21",
				Tags:        []string{"house, techno, tech.house, electro.house, future.house, bass.house, melodic.house"},
				ReleaseTags: "MP3 / 320 / Cassette",
			},
			want: Release{
				TorrentName: "Various Artists - Music '21",
				Clean:       "Various Artists   Music '21",
				Tags:        []string{"house, techno, tech.house, electro.house, future.house, bass.house, melodic.house"},
				ReleaseTags: "MP3 / 320 / Cassette",
				Group:       "",
				Format:      "MP3",
				Source:      "Cassette",
				Quality:     "320",
			},
			wantErr: false,
		},
		{
			name: "parse_music_3",
			fields: Release{
				TorrentName: "The artist (ザ・フリーダムユニティ) - Long album name",
				ReleaseTags: "MP3 / V0 (VBR) / CD",
			},
			want: Release{
				TorrentName: "The artist (ザ・フリーダムユニティ) - Long album name",
				Clean:       "The artist (ザ・フリーダムユニティ)   Long album name",
				ReleaseTags: "MP3 / V0 (VBR) / CD",
				Group:       "",
				Format:      "MP3",
				Quality:     "V0 (VBR)",
				Source:      "CD",
			},
			wantErr: false,
		},
		{
			name: "parse_music_4",
			fields: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
			},
			want: Release{
				TorrentName: "Artist - Albumname",
				Clean:       "Artist   Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
				Group:       "",
				Audio:       "FLAC",
				Format:      "FLAC",
				Quality:     "Lossless",
				Source:      "CD",
				HasCue:      true,
				HasLog:      true,
				LogScore:    100,
			},
			wantErr: false,
		},
		{
			name: "parse_music_5",
			fields: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / 24bit Lossless / Log / 100% / Cue / CD",
			},
			want: Release{
				TorrentName: "Artist - Albumname",
				Clean:       "Artist   Albumname",
				ReleaseTags: "FLAC / 24bit Lossless / Log / 100% / Cue / CD",
				Group:       "",
				Audio:       "FLAC",
				Format:      "FLAC",
				Quality:     "24bit Lossless",
				Source:      "CD",
				HasCue:      true,
				HasLog:      true,
				LogScore:    100,
			},
			wantErr: false,
		},
		{
			name: "parse_movies_case_1",
			fields: Release{
				TorrentName: "I Am Movie 2007 Theatrical UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-GROUP1",
			},
			want: Release{
				TorrentName: "I Am Movie 2007 Theatrical UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-GROUP1",
				Clean:       "I Am Movie 2007 Theatrical UHD BluRay 2160p DTS HD MA 5 1 DV HEVC HYBRID REMUX GROUP1",
				Resolution:  "2160p",
				Source:      "BluRay",
				Codec:       "HEVC",
				HDR:         "DV",
				Audio:       "DTS-HD MA 5.1", // need to fix audio parsing
				Edition:     "Theatrical",
				Hybrid:      true,
				Year:        2007,
				Group:       "GROUP1",
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

func TestRelease_MapVars(t *testing.T) {
	type args struct {
		varMap     map[string]string
		definition IndexerDefinition
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
		{
			name:   "8",
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
			args: args{
				varMap: map[string]string{
					"torrentName":      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
					"category":         "tv",
					"year":             "2020",
					"freeleechPercent": "100%",
					"uploader":         "Anon",
					"torrentSize":      "10000",
					"tags":             "hip.hop,rhythm.and.blues, 2000s",
				},
				definition: IndexerDefinition{Parse: &IndexerParse{ForceSizeUnit: "MB"}},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields
			_ = r.MapVars(tt.args.definition, tt.args.varMap)

			assert.Equal(t, tt.want, r)
		})
	}
}

func TestSplitAny(t *testing.T) {
	type args struct {
		s    string
		seps string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test_1",
			args: args{
				s:    "Tag1 / Tag2 / Tag3",
				seps: "/ ",
			},
			want: []string{"Tag1", "Tag2", "Tag3"},
		},
		{
			name: "test_2",
			args: args{
				s:    "Tag1 | Tag2 | Tag3",
				seps: "| ",
			},
			want: []string{"Tag1", "Tag2", "Tag3"},
		},
		{
			name: "test_3",
			args: args{
				s:    "Tag1 | Tag2 / Tag3",
				seps: "| /",
			},
			want: []string{"Tag1", "Tag2", "Tag3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, SplitAny(tt.args.s, tt.args.seps), "SplitAny(%v, %v)", tt.args.s, tt.args.seps)
		})
	}
}

func TestRelease_ParseString(t *testing.T) {
	type fields struct {
		ID                          int64
		FilterStatus                ReleaseFilterStatus
		Rejections                  []string
		Indexer                     string
		FilterName                  string
		Protocol                    ReleaseProtocol
		Implementation              ReleaseImplementation
		Timestamp                   time.Time
		GroupID                     string
		TorrentID                   string
		TorrentURL                  string
		TorrentTmpFile              string
		TorrentHash                 string
		TorrentName                 string
		Size                        uint64
		Raw                         string
		Clean                       string
		Title                       string
		Category                    string
		Season                      int
		Episode                     int
		Year                        int
		Resolution                  string
		Source                      string
		Codec                       string
		Container                   string
		HDR                         string
		Audio                       string
		Group                       string
		Region                      string
		Language                    string
		Edition                     string
		Unrated                     bool
		Hybrid                      bool
		Proper                      bool
		Repack                      bool
		Website                     string
		ThreeD                      bool
		Artists                     []string
		Type                        string
		Format                      string
		Quality                     string
		LogScore                    int
		HasLog                      bool
		HasCue                      bool
		IsScene                     bool
		Origin                      string
		Tags                        []string
		ReleaseTags                 string
		Freeleech                   bool
		FreeleechPercent            int
		Uploader                    string
		PreTime                     string
		RawCookie                   string
		AdditionalSizeCheckRequired bool
		FilterID                    int
		Filter                      *Filter
		ActionStatus                []ReleaseActionStatus
	}
	type args struct {
		title string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{name: "parse_1", fields: fields{}, args: args{title: "Phenomena 1985 International Cut UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-FraMeSToR"}, wantErr: false},
		{name: "parse_2", fields: fields{}, args: args{title: "Justice League: Dark 2017 UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-FraMeSToR"}, wantErr: false},
		{name: "parse_3", fields: fields{}, args: args{title: "Outer.Range.S01E02.The.Land.1080p.AMZN.WEB-DL.DDP5.1.H.264-TOMMY"}, wantErr: false},
		{name: "parse_4", fields: fields{}, args: args{title: "WeCrashed S01E07 The Power of We 2160p ATVP WEB-DL DDP 5.1 Atmos HDR H.265-NOSiViD"}, wantErr: false},
		{name: "parse_5", fields: fields{}, args: args{title: "WeCrashed.S01E07.The.Power.of.We.DV.2160p.ATVP.WEB-DL.DDPA5.1.H.265-NOSiViD"}, wantErr: false},
		{name: "parse_6", fields: fields{}, args: args{title: "WeCrashed.S01E07.The.Power.of.We.DV.2160p.ATVP.WEB-DL.DDPA5.1.H265-NOSiViD"}, wantErr: false},
		{name: "parse_7", fields: fields{}, args: args{title: "WeCrashed.S01E07.The.Power.of.We.DV.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}, wantErr: false},
		{name: "parse_8", fields: fields{}, args: args{title: "WeCrashed.S01E07.The.Power.of.We.HDR.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}, wantErr: false},
		{name: "parse_9", fields: fields{}, args: args{title: "WeCrashed.S01.HDR.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}, wantErr: false},
		{name: "parse_10", fields: fields{}, args: args{title: "WeCrashed.S01.DV.HDR+.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}, wantErr: false},
		{name: "parse_11", fields: fields{}, args: args{title: "WeCrashed.S01.DoVi.HDR10+.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}, wantErr: false},
		{name: "parse_12", fields: fields{}, args: args{title: "WeCrashed.S01.Dolby.Vision.2160p.ATVP.WEB-DL.DDPA5.1.x265-NOSiViD"}, wantErr: false},
		{name: "parse_13", fields: fields{}, args: args{title: "WeCrashed.S01.Dolby.Vision.1080p.ATVP.WEB-DL.DDPA5.1.x264-NOSiViD"}, wantErr: false},
		{name: "parse_14", fields: fields{}, args: args{title: "Without Remorse 2021 1080p Blu-ray AVC DTS-HD MA 5.1-MTeam"}, wantErr: false},
		{name: "parse_15", fields: fields{}, args: args{title: "Annette 2021 2160p GER UHD Blu-ray SDR HEVC DTS-HD MA 5.1-UNTOUCHED"}, wantErr: false},
		{name: "parse_16", fields: fields{}, args: args{title: "Sing 2 2021 MULTi COMPLETE UHD Blu-ray TrueHD Atmos 7.1-MMCLX"}, wantErr: false},
		{name: "parse_17", fields: fields{}, args: args{title: "NBC.Nightly.News.2022.04.12.1080p.NBC.WEB-DL.AAC2.0.H.264-TEPES"}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Release{
				ID:                          tt.fields.ID,
				FilterStatus:                tt.fields.FilterStatus,
				Rejections:                  tt.fields.Rejections,
				Indexer:                     tt.fields.Indexer,
				FilterName:                  tt.fields.FilterName,
				Protocol:                    tt.fields.Protocol,
				Implementation:              tt.fields.Implementation,
				Timestamp:                   tt.fields.Timestamp,
				GroupID:                     tt.fields.GroupID,
				TorrentID:                   tt.fields.TorrentID,
				TorrentURL:                  tt.fields.TorrentURL,
				TorrentTmpFile:              tt.fields.TorrentTmpFile,
				TorrentHash:                 tt.fields.TorrentHash,
				TorrentName:                 tt.fields.TorrentName,
				Size:                        tt.fields.Size,
				Raw:                         tt.fields.Raw,
				Clean:                       tt.fields.Clean,
				Title:                       tt.fields.Title,
				Category:                    tt.fields.Category,
				Season:                      tt.fields.Season,
				Episode:                     tt.fields.Episode,
				Year:                        tt.fields.Year,
				Resolution:                  tt.fields.Resolution,
				Source:                      tt.fields.Source,
				Codec:                       tt.fields.Codec,
				Container:                   tt.fields.Container,
				HDR:                         tt.fields.HDR,
				Audio:                       tt.fields.Audio,
				Group:                       tt.fields.Group,
				Region:                      tt.fields.Region,
				Language:                    tt.fields.Language,
				Edition:                     tt.fields.Edition,
				Unrated:                     tt.fields.Unrated,
				Hybrid:                      tt.fields.Hybrid,
				Proper:                      tt.fields.Proper,
				Repack:                      tt.fields.Repack,
				Website:                     tt.fields.Website,
				ThreeD:                      tt.fields.ThreeD,
				Artists:                     tt.fields.Artists,
				Type:                        tt.fields.Type,
				Format:                      tt.fields.Format,
				Quality:                     tt.fields.Quality,
				LogScore:                    tt.fields.LogScore,
				HasLog:                      tt.fields.HasLog,
				HasCue:                      tt.fields.HasCue,
				IsScene:                     tt.fields.IsScene,
				Origin:                      tt.fields.Origin,
				Tags:                        tt.fields.Tags,
				ReleaseTags:                 tt.fields.ReleaseTags,
				Freeleech:                   tt.fields.Freeleech,
				FreeleechPercent:            tt.fields.FreeleechPercent,
				Uploader:                    tt.fields.Uploader,
				PreTime:                     tt.fields.PreTime,
				RawCookie:                   tt.fields.RawCookie,
				AdditionalSizeCheckRequired: tt.fields.AdditionalSizeCheckRequired,
				FilterID:                    tt.fields.FilterID,
				Filter:                      tt.fields.Filter,
				ActionStatus:                tt.fields.ActionStatus,
			}
			_ = r.ParseString(tt.args.title)
			//fmt.Sprintf("ParseString(%v)", tt.args.title)
		})
	}
}
