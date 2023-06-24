// Copyright (c) 2021 - 2023, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMacros_Parse(t *testing.T) {
	currentTime := time.Now()

	type fields struct {
		TorrentName     string
		TorrentPathName string
		TorrentUrl      string
		Indexer         string
	}
	type args struct {
		text string
	}
	tests := []struct {
		name    string
		fields  fields
		release Release
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test_ok",
			release: Release{
				TorrentName:    "This movie 2021",
				TorrentTmpFile: "/tmp/a-temporary-file.torrent",
				Indexer:        "mock1",
			},
			args:    args{text: "Print mee {{.TorrentPathName}}"},
			want:    "Print mee /tmp/a-temporary-file.torrent",
			wantErr: false,
		},
		{
			name: "test_bad",
			release: Release{
				TorrentName:    "This movie 2021",
				TorrentTmpFile: "/tmp/a-temporary-file.torrent",
				Indexer:        "mock1",
			},
			args:    args{text: "Print mee {{TorrentPathName}}"},
			want:    "",
			wantErr: true,
		},
		{
			name: "test_program_arg",
			release: Release{
				TorrentName:    "This movie 2021",
				TorrentTmpFile: "/tmp/a-temporary-file.torrent",
				Indexer:        "mock1",
			},
			args:    args{text: "add {{.TorrentPathName}} --category test"},
			want:    "add /tmp/a-temporary-file.torrent --category test",
			wantErr: false,
		},
		{
			name: "test_program_arg_bad",
			release: Release{
				TorrentTmpFile: "/tmp/a-temporary-file.torrent",
				Indexer:        "mock1",
			},
			args:    args{text: "add {{.TorrenttPathName}} --category test"},
			want:    "",
			wantErr: true,
		},
		{
			name: "test_program_arg",
			release: Release{
				TorrentName:    "This movie 2021",
				TorrentTmpFile: "/tmp/a-temporary-file.torrent",
				Indexer:        "mock1",
			},
			args:    args{text: "add {{.TorrentPathName}} --category test --other {{.TorrentName}}"},
			want:    "add /tmp/a-temporary-file.torrent --category test --other This movie 2021",
			wantErr: false,
		},
		{
			name: "test_args_long",
			release: Release{
				TorrentName: "This movie 2021",
				TorrentURL:  "https://some.site/download/fakeid",
				Indexer:     "mock1",
			},
			args:    args{text: "{{.TorrentName}} {{.TorrentUrl}} SOME_LONG_TOKEN"},
			want:    "This movie 2021 https://some.site/download/fakeid SOME_LONG_TOKEN",
			wantErr: false,
		},
		{
			name: "test_args_long_1",
			release: Release{
				TorrentName: "This movie 2021",
				TorrentURL:  "https://some.site/download/fakeid",
				Indexer:     "mock1",
			},
			args:    args{text: "{{.Indexer}} {{.TorrentName}} {{.TorrentUrl}} SOME_LONG_TOKEN"},
			want:    "mock1 This movie 2021 https://some.site/download/fakeid SOME_LONG_TOKEN",
			wantErr: false,
		},
		{
			name: "test_args_category",
			release: Release{
				TorrentName: "This movie 2021",
				TorrentURL:  "https://some.site/download/fakeid",
				Indexer:     "mock1",
			},
			args:    args{text: "{{.Indexer}}-race"},
			want:    "mock1-race",
			wantErr: false,
		},
		{
			name: "test_args_category_year",
			release: Release{
				TorrentName: "This movie 2021",
				TorrentURL:  "https://some.site/download/fakeid",
				Indexer:     "mock1",
			},
			args:    args{text: "{{.Indexer}}-{{.CurrentYear}}-race"},
			want:    fmt.Sprintf("mock1-%v-race", currentTime.Year()),
			wantErr: false,
		},
		{
			name: "test_args_category_year",
			release: Release{
				TorrentName: "This movie 2021",
				TorrentURL:  "https://some.site/download/fakeid",
				Indexer:     "mock1",
				Resolution:  "2160p",
				HDR:         []string{"DV"},
			},
			args:    args{text: "movies-{{.Resolution}}{{ if .HDR }}-{{.HDR}}{{ end }}"},
			want:    "movies-2160p-DV",
			wantErr: false,
		},
		{
			name: "test_args_category_and_if",
			release: Release{
				TorrentName: "This movie 2021",
				TorrentURL:  "https://some.site/download/fakeid",
				Indexer:     "mock1",
				Resolution:  "2160p",
				HDR:         []string{"HDR"},
			},
			args:    args{text: "movies-{{.Resolution}}{{ if .HDR }}-{{.HDR}}{{ end }}"},
			want:    "movies-2160p-HDR",
			wantErr: false,
		},
		{
			name: "test_release_year_1",
			release: Release{
				TorrentName: "This movie 2021",
				TorrentURL:  "https://some.site/download/fakeid",
				Indexer:     "mock1",
				Resolution:  "2160p",
				HDR:         []string{"HDR"},
				Year:        2021,
			},
			args:    args{text: "movies-{{.Year}}"},
			want:    "movies-2021",
			wantErr: false,
		},
		{
			name: "test_size_formating",
			release: Release{
				Size: 3834225472,
			},
			args:    args{text: "{{printf \"%.2f GB\" (divf .Size 1073741824)}}"},
			want:    "3.57 GB",
			wantErr: false,
		},
		{
			name: "test_text_manipulation",
			release: Release{
				TorrentName: "Title Name 2 - Keyword [Blu-ray][MKV][h264 10-bit][1080p][FLAC 2.0][Dual Audio][Softsubs (Sub Group)][Freeleech]",
			},
			args:    args{text: "{{join \"\" (regexSplit \"^.+- Keyword \" .TorrentName -1)}}"},
			want:    "[Blu-ray][MKV][h264 10-bit][1080p][FLAC 2.0][Dual Audio][Softsubs (Sub Group)][Freeleech]",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMacro(tt.release)
			got, err := m.Parse(tt.args.text)

			assert.Equal(t, currentTime.Year(), m.CurrentYear)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
