// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRelease_Parse(t *testing.T) {
	tests := []struct {
		name   string
		fields Release
		want   Release
	}{
		{
			name: "parse_1",
			fields: Release{
				TorrentName: "Servant S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-FLUX",
			},
			want: Release{
				TorrentName:   "Servant S01 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-FLUX",
				Title:         "Servant",
				Season:        1,
				Episode:       0,
				Resolution:    "2160p",
				Source:        "WEB-DL",
				Codec:         []string{"HEVC"},
				Audio:         []string{"DDP", "Atmos"},
				AudioChannels: "5.1",
				HDR:           []string{"DV"},
				Group:         "FLUX",
				//Website: "ATVP",
				Type: "series",
			},
		},
		{
			name: "parse_2",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
			},
			want: Release{
				TorrentName:   "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				Title:         "Servant",
				Season:        1,
				Episode:       0,
				Resolution:    "2160p",
				Source:        "WEB-DL",
				Codec:         []string{"HEVC"},
				Audio:         []string{"DDP", "Atmos"},
				AudioChannels: "5.1",
				HDR:           []string{"DV"},
				Group:         "FLUX",
				Type:          "series",
			},
		},
		{
			name: "parse_3",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags: "MKV / 2160p / WEB-DL",
			},
			want: Release{
				TorrentName:   "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags:   "MKV / 2160p / WEB-DL",
				Title:         "Servant",
				Season:        1,
				Episode:       0,
				Resolution:    "2160p",
				Source:        "WEB-DL",
				Container:     "mkv",
				Codec:         []string{"HEVC"},
				Audio:         []string{"DDP", "Atmos"},
				AudioChannels: "5.1",
				HDR:           []string{"DV"},
				Group:         "FLUX",
				Type:          "series",
			},
		},
		{
			name: "parse_4",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags: "MKV | 2160p | WEB-DL",
			},
			want: Release{
				TorrentName:   "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags:   "MKV | 2160p | WEB-DL",
				Title:         "Servant",
				Season:        1,
				Episode:       0,
				Resolution:    "2160p",
				Source:        "WEB-DL",
				Container:     "mkv",
				Codec:         []string{"HEVC"},
				Audio:         []string{"DDP", "Atmos"},
				AudioChannels: "5.1",
				HDR:           []string{"DV"},
				Group:         "FLUX",
				Type:          "series",
			},
		},
		{
			name: "parse_5",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags: "MP4 | 2160p | WEB-DL",
			},
			want: Release{
				TorrentName:   "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags:   "MP4 | 2160p | WEB-DL",
				Title:         "Servant",
				Season:        1,
				Episode:       0,
				Resolution:    "2160p",
				Source:        "WEB-DL",
				Container:     "mp4",
				Codec:         []string{"HEVC"},
				Audio:         []string{"DDP", "Atmos"},
				AudioChannels: "5.1",
				HDR:           []string{"DV"},
				Group:         "FLUX",
				Type:          "series",
			},
		},
		{
			name: "parse_6",
			fields: Release{
				TorrentName: "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags: "MP4 | 2160p | WEB-DL | Freeleech!",
			},
			want: Release{
				TorrentName:   "Servant.S01.2160p.ATVP.WEB-DL.DDP.5.1.Atmos.DV.HEVC-FLUX",
				ReleaseTags:   "MP4 | 2160p | WEB-DL | Freeleech!",
				Title:         "Servant",
				Season:        1,
				Episode:       0,
				Resolution:    "2160p",
				Source:        "WEB-DL",
				Container:     "mp4",
				Codec:         []string{"HEVC"},
				Audio:         []string{"DDP", "Atmos"},
				AudioChannels: "5.1",
				HDR:           []string{"DV"},
				Group:         "FLUX",
				Freeleech:     true,
				Bonus:         []string{"Freeleech"},
				Type:          "series",
			},
		},
		{
			name: "parse_8",
			fields: Release{
				TorrentName: "Rippers.Revenge.2023.German.DL.1080p.BluRay.MPEG2-GROUP",
			},
			want: Release{
				TorrentName: "Rippers.Revenge.2023.German.DL.1080p.BluRay.MPEG2-GROUP",
				Title:       "Rippers Revenge",
				Year:        2023,
				Language:    []string{"GERMAN", "DL"},
				Resolution:  "1080p",
				Source:      "BluRay",
				Codec:       []string{"MPEG-2"},
				Group:       "GROUP",
				Type:        "movie",
			},
		},
		{
			name: "parse_7",
			fields: Release{
				TorrentName: "Analogue.1080i.AHDTV.H264-ABCDEF",
			},
			want: Release{
				TorrentName: "Analogue.1080i.AHDTV.H264-ABCDEF",
				Title:       "Analogue",
				Resolution:  "1080p", // rls does not differentiate between 1080i and 1080p which results in all 1080 releases being parsed as 1080p
				Source:      "AHDTV",
				Codec:       []string{"H.264"},
				Group:       "ABCDEF",
				Type:        "movie",
			},
		},
		{
			name: "parse_music_1",
			fields: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
			},
			want: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
				Title:       "Artist",
				Group:       "Albumname",
				Audio:       []string{"Cue", "FLAC", "Lossless", "Log100", "Log"},
				AudioFormat: "FLAC",
				Source:      "CD",
				Bitrate:     "Lossless",
				HasLog:      true,
				LogScore:    100,
				HasCue:      true,
			},
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
				Tags:        []string{"house, techno, tech.house, electro.house, future.house, bass.house, melodic.house"},
				ReleaseTags: "MP3 / 320 / Cassette",
				Title:       "Various Artists - Music '21",
				Source:      "Cassette",
				Audio:       []string{"320", "MP3"},
				AudioFormat: "MP3",
				Bitrate:     "320",
			},
		},
		{
			name: "parse_music_3",
			fields: Release{
				TorrentName: "The artist (ザ・フリーダムユニティ) - Long album name",
				ReleaseTags: "MP3 / V0 (VBR) / CD",
			},
			want: Release{
				TorrentName: "The artist (ザ・フリーダムユニティ) - Long album name",
				ReleaseTags: "MP3 / V0 (VBR) / CD",
				Title:       "The artist",
				Group:       "name",
				Source:      "CD",
				Audio:       []string{"MP3", "VBR", "V0 (VBR)"},
				AudioFormat: "MP3",
				Bitrate:     "V0 (VBR)",
			},
		},
		{
			name: "parse_music_4",
			fields: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
			},
			want: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / Lossless / Log / 100% / Cue / CD",
				Title:       "Artist",
				Group:       "Albumname",
				Audio:       []string{"Cue", "FLAC", "Lossless", "Log100", "Log"},
				AudioFormat: "FLAC",
				Source:      "CD",
				Bitrate:     "Lossless",
				HasLog:      true,
				LogScore:    100,
				HasCue:      true,
			},
		},
		{
			name: "parse_music_5",
			fields: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / 24bit Lossless / Log / 100% / Cue / CD",
			},
			want: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / 24bit Lossless / Log / 100% / Cue / CD",
				Title:       "Artist",
				Group:       "Albumname",
				Audio:       []string{"24BIT Lossless", "Cue", "FLAC", "Log100", "Log"},
				AudioFormat: "FLAC",
				Source:      "CD",
				Bitrate:     "24BIT Lossless",
				HasLog:      true,
				LogScore:    100,
				HasCue:      true,
			},
		},
		{
			name: "parse_music_6",
			fields: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / 24bit Lossless / Log / 78% / Cue / CD",
			},
			want: Release{
				TorrentName: "Artist - Albumname",
				ReleaseTags: "FLAC / 24bit Lossless / Log / 78% / Cue / CD",
				Title:       "Artist",
				Group:       "Albumname",
				Audio:       []string{"24BIT Lossless", "Cue", "FLAC", "Log78", "Log"},
				AudioFormat: "FLAC",
				Source:      "CD",
				Bitrate:     "24BIT Lossless",
				HasLog:      true,
				LogScore:    78,
				HasCue:      true,
			},
		},
		{
			name: "parse_movies_case_1",
			fields: Release{
				TorrentName: "I Am Movie 2007 Theatrical UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-GROUP1",
			},
			want: Release{
				TorrentName:   "I Am Movie 2007 Theatrical UHD BluRay 2160p DTS-HD MA 5.1 DV HEVC HYBRID REMUX-GROUP1",
				Title:         "I Am Movie",
				Resolution:    "2160p",
				Source:        "UHD.BluRay",
				Codec:         []string{"HEVC"},
				HDR:           []string{"DV"},
				Audio:         []string{"DTS-HD.MA"},
				AudioChannels: "5.1",
				Year:          2007,
				Group:         "GROUP1",
				Other:         []string{"HYBRiD", "REMUX"},
				Type:          "movie",
			},
		},
		{
			name: "parse_title_1",
			fields: Release{
				TorrentName: "The Peripheral (2022) S01 (2160p AMZN WEB-DL H265 HDR10+ DDP 5.1 English - GROUP1)",
			},
			want: Release{
				TorrentName:   "The Peripheral (2022) S01 (2160p AMZN WEB-DL H265 HDR10+ DDP 5.1 English - GROUP1)",
				Title:         "The Peripheral",
				Resolution:    "2160p",
				Source:        "WEB-DL",
				Codec:         []string{"H.265"},
				HDR:           []string{"HDR10+"},
				Audio:         []string{"DDP"},
				AudioChannels: "5.1",
				Year:          2022,
				Group:         "GROUP1",
				Season:        1,
				Language:      []string{"ENGLiSH"},
				Type:          "series",
			},
		},
		{
			name: "parse_missing_source",
			fields: Release{
				TorrentName: "Old Movie 1954 2160p Remux DoVi HDR10 HEVC DTS-HD MA 5.1-CiNEPHiLES",
			},
			want: Release{
				TorrentName:   "Old Movie 1954 2160p Remux DoVi HDR10 HEVC DTS-HD MA 5.1-CiNEPHiLES",
				Title:         "Old Movie",
				Year:          1954,
				Source:        "UHD.BluRay",
				Resolution:    "2160p",
				Other:         []string{"REMUX"},
				HDR:           []string{"DV", "HDR10"},
				Codec:         []string{"HEVC"},
				Audio:         []string{"DTS-HD.MA"},
				AudioChannels: "5.1",
				Group:         "CiNEPHiLES",
				Type:          "movie",
			},
		},
		{
			name: "parse_missing_source",
			fields: Release{
				TorrentName: "Death Hunt 1981 1080p Remux AVC DTS-HD MA 2.0-playBD",
			},
			want: Release{
				TorrentName:   "Death Hunt 1981 1080p Remux AVC DTS-HD MA 2.0-playBD",
				Title:         "Death Hunt",
				Year:          1981,
				Source:        "BluRay",
				Resolution:    "1080p",
				Other:         []string{"REMUX"},
				Codec:         []string{"AVC"},
				Audio:         []string{"DTS-HD.MA"},
				AudioChannels: "2.0",
				Group:         "playBD",
				Type:          "movie",
			},
		},
		{
			name: "parse_confusing_group",
			fields: Release{
				TorrentName: "Old Movie 1954 2160p Remux DoVi HDR10 HEVC DTS-HD MA 5.1-VHS",
			},
			want: Release{
				TorrentName:   "Old Movie 1954 2160p Remux DoVi HDR10 HEVC DTS-HD MA 5.1-VHS",
				Title:         "Old Movie",
				Year:          1954,
				Source:        "UHD.BluRay",
				Resolution:    "2160p",
				Other:         []string{"REMUX"},
				HDR:           []string{"DV", "HDR10"},
				Codec:         []string{"HEVC"},
				Audio:         []string{"DTS-HD.MA"},
				AudioChannels: "5.1",
				Group:         "VHS",
				Type:          "movie",
			},
		},
		{
			name: "parse_confusing_group",
			fields: Release{
				TorrentName: "Old Movie 1954 2160p Remux DoVi HDR10 HEVC DTS-HD MA 5.1 VHS",
			},
			want: Release{
				TorrentName:   "Old Movie 1954 2160p Remux DoVi HDR10 HEVC DTS-HD MA 5.1 VHS",
				Title:         "Old Movie",
				Year:          1954,
				Source:        "UHD.BluRay",
				Resolution:    "2160p",
				Other:         []string{"REMUX"},
				HDR:           []string{"DV", "HDR10"},
				Codec:         []string{"HEVC"},
				Audio:         []string{"DTS-HD.MA"},
				AudioChannels: "5.1",
				Group:         "VHS",
				Type:          "movie",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields
			r.ParseString(tt.fields.TorrentName)

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
				TorrentName:      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:         "tv",
				Freeleech:        true,
				FreeleechPercent: 100,
				Bonus:            []string{"Freeleech"},
				Uploader:         "Anon",
				Size:             uint64(10000000000),
			},
			args: args{
				varMap: map[string]string{
					"torrentName": "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
					"category":    "tv",
					"freeleech":   "freeleech",
					"uploader":    "Anon",
					"torrentSize": "10GB",
				},
			},
		},
		{
			name:   "3",
			fields: &Release{},
			want: &Release{
				TorrentName:      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:         "tv",
				Freeleech:        true,
				FreeleechPercent: 100,
				Bonus:            []string{"Freeleech", "Freeleech100"},
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
				Freeleech:        true,
				FreeleechPercent: 50,
				Bonus:            []string{"Freeleech", "Freeleech50"},
				Uploader:         "Anon",
				Size:             uint64(10000000000),
				Tags:             []string{"foreign", "tv"},
			},
			args: args{varMap: map[string]string{
				"torrentName":      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				"category":         "tv",
				"freeleechPercent": "50%",
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
				Freeleech:        true,
				FreeleechPercent: 100,
				Bonus:            []string{"Freeleech", "Freeleech100"},
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
				Freeleech:        true,
				FreeleechPercent: 100,
				Bonus:            []string{"Freeleech", "Freeleech100"},
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
				Freeleech:        true,
				FreeleechPercent: 25,
				Bonus:            []string{"Freeleech", "Freeleech25"},
				Uploader:         "Anon",
				Size:             uint64(10000000000),
				Tags:             []string{"hip.hop", "rhythm.and.blues", "2000s"},
			},
			args: args{varMap: map[string]string{
				"torrentName":      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				"category":         "tv",
				"year":             "2020",
				"freeleechPercent": "25%",
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
				Freeleech:        true,
				FreeleechPercent: 100,
				Bonus:            []string{"Freeleech", "Freeleech100"},
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
				definition: IndexerDefinition{IRC: &IndexerIRC{Parse: &IndexerIRCParse{ForceSizeUnit: "MB"}}},
			},
		},
		{
			name:   "9",
			fields: &Release{},
			want: &Release{
				TorrentName: "Greatest Anime Ever",
				Year:        2022,
				Group:       "GROUP1",
				Tags:        []string{"comedy", "fantasy", "school.life", "shounen", "slice.of.life"},
				Uploader:    "Tester",
			},
			args: args{varMap: map[string]string{
				"torrentName":  "Greatest Anime Ever",
				"year":         "2022",
				"releaseGroup": "GROUP1",
				"tags":         "comedy, fantasy, school.life, shounen, slice.of.life",
				"uploader":     "Tester",
			}},
		},
		{
			name:   "10",
			fields: &Release{},
			want: &Release{
				TorrentName:      "Greatest Anime Ever",
				Year:             2022,
				Group:            "GROUP1",
				Tags:             []string{"comedy", "fantasy", "school.life", "shounen", "slice.of.life"},
				Uploader:         "Tester",
				Freeleech:        true,
				FreeleechPercent: 100,
				Bonus:            []string{"Freeleech"},
			},
			args: args{varMap: map[string]string{
				"torrentName":  "Greatest Anime Ever",
				"year":         "2022",
				"releaseGroup": "GROUP1",
				"tags":         "comedy, fantasy, school.life, shounen, slice.of.life",
				"uploader":     "Tester",
				"freeleech":    "VIP",
			}},
		},
		{
			name:   "11",
			fields: &Release{},
			want: &Release{
				TorrentName:      "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
				Category:         "tv",
				Freeleech:        true,
				FreeleechPercent: 100,
				Bonus:            []string{"Freeleech"},
				Uploader:         "Anon",
				Size:             uint64(10000000000),
				Tags:             []string{"comedy", "science fiction", "fantasy", "school.life", "shounen", "slice.of.life"},
			},
			args: args{
				varMap: map[string]string{
					"torrentName": "Good show S02 2160p ATVP WEB-DL DDP 5.1 Atmos DV HEVC-GROUP2",
					"category":    "tv",
					"tags":        "comedy, science fiction, fantasy, school.life, shounen, slice.of.life",
					"freeleech":   "freeleech",
					"uploader":    "Anon",
					"torrentSize": "10GB",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields
			_ = r.MapVars(&tt.args.definition, tt.args.varMap)

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
		Release
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
		{name: "parse_18", fields: fields{}, args: args{title: "[SubsPlease] Heroine Tarumono! Kiraware Heroine to Naisho no Oshigoto - 04 (1080p) [17083ED9]"}, wantErr: false},
		{name: "parse_19", fields: fields{}, args: args{title: "The World is Not Enough 1999 2160p WEB-DL HEVC DTS-HD MA 5.1 H.265-DEFLATE"}, wantErr: false},
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
				DownloadURL:                 tt.fields.DownloadURL,
				TorrentTmpFile:              tt.fields.TorrentTmpFile,
				TorrentHash:                 tt.fields.TorrentHash,
				TorrentName:                 tt.fields.TorrentName,
				Size:                        tt.fields.Size,
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
				Proper:                      tt.fields.Proper,
				Repack:                      tt.fields.Repack,
				Website:                     tt.fields.Website,
				Artists:                     tt.fields.Artists,
				Type:                        tt.fields.Type,
				LogScore:                    tt.fields.LogScore,
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
			r.ParseString(tt.args.title)
		})
	}
}

func Test_getUniqueTags(t *testing.T) {
	type args struct {
		target []string
		source []string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "1",
			args: args{
				target: []string{},
				source: []string{"mp4"},
			},
			want: []string{"mp4"},
		},
		{
			name: "2",
			args: args{
				target: []string{"mp4"},
				source: []string{"mp4"},
			},
			want: []string{"mp4"},
		},
		{
			name: "3",
			args: args{
				target: []string{"mp4"},
				source: []string{"mp4", "dv"},
			},
			want: []string{"mp4", "dv"},
		},
		{
			name: "4",
			args: args{
				target: []string{"dv"},
				source: []string{"mp4", "dv"},
			},
			want: []string{"dv", "mp4"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, getUniqueTags(tt.args.target, tt.args.source), "getUniqueTags(%v, %v)", tt.args.target, tt.args.source)
		})
	}
}
