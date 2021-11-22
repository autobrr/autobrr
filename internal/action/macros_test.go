package action

import (
	"testing"
)

func TestMacros_Parse(t *testing.T) {
	type fields struct {
		TorrentName     string
		TorrentPathName string
		TorrentUrl      string
	}
	type args struct {
		text string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "test_ok",
			fields:  fields{TorrentPathName: "/tmp/a-temporary-file.torrent"},
			args:    args{text: "Print mee {{.TorrentPathName}}"},
			want:    "Print mee /tmp/a-temporary-file.torrent",
			wantErr: false,
		},
		{
			name:    "test_bad",
			fields:  fields{TorrentPathName: "/tmp/a-temporary-file.torrent"},
			args:    args{text: "Print mee {{TorrentPathName}}"},
			want:    "",
			wantErr: true,
		},
		{
			name:    "test_program_arg",
			fields:  fields{TorrentPathName: "/tmp/a-temporary-file.torrent"},
			args:    args{text: "add {{.TorrentPathName}} --category test"},
			want:    "add /tmp/a-temporary-file.torrent --category test",
			wantErr: false,
		},
		{
			name:    "test_program_arg_bad",
			fields:  fields{TorrentPathName: "/tmp/a-temporary-file.torrent"},
			args:    args{text: "add {{.TorrenttPathName}} --category test"},
			want:    "",
			wantErr: true,
		},
		{
			name: "test_program_arg",
			fields: fields{
				TorrentName:     "This movie 2021",
				TorrentPathName: "/tmp/a-temporary-file.torrent",
			},
			args:    args{text: "add {{.TorrentPathName}} --category test --other {{.TorrentName}}"},
			want:    "add /tmp/a-temporary-file.torrent --category test --other This movie 2021",
			wantErr: false,
		},
		{
			name: "test_args_long",
			fields: fields{
				TorrentName: "This movie 2021",
				TorrentUrl:  "https://some.site/download/fakeid",
			},
			args:    args{text: "{{.TorrentName}} {{.TorrentUrl}} SOME_LONG_TOKEN"},
			want:    "This movie 2021 https://some.site/download/fakeid SOME_LONG_TOKEN",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Macro{
				TorrentPathName: tt.fields.TorrentPathName,
				TorrentUrl:      tt.fields.TorrentUrl,
				TorrentName:     tt.fields.TorrentName,
			}
			got, err := m.Parse(tt.args.text)

			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Parse() got = %v, want %v", got, tt.want)
			}
		})
	}
}
