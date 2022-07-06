package action

import (
	"fmt"
	"testing"
	"time"

	"github.com/autobrr/autobrr/internal/domain"
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
		release domain.Release
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test_ok",
			release: domain.Release{
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
			release: domain.Release{
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
			release: domain.Release{
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
			release: domain.Release{
				TorrentTmpFile: "/tmp/a-temporary-file.torrent",
				Indexer:        "mock1",
			},
			args:    args{text: "add {{.TorrenttPathName}} --category test"},
			want:    "",
			wantErr: true,
		},
		{
			name: "test_program_arg",
			release: domain.Release{
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
			release: domain.Release{
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
			release: domain.Release{
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
			release: domain.Release{
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
			release: domain.Release{
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
			release: domain.Release{
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
			release: domain.Release{
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
			release: domain.Release{
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
