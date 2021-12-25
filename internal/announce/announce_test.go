package announce

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_announceProcessor_processTorrentUrl(t *testing.T) {
	type args struct {
		match     string
		vars      map[string]string
		extraVars map[string]string
		encode    []string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "passing with vars_1",
			args: args{
				match: "{{ .baseUrl }}api/v1/torrents/{{ .torrentId }}/torrent?key={{ .apikey }}",
				vars: map[string]string{
					"baseUrl":   "https://example.test/",
					"torrentId": "000000",
				},
				extraVars: map[string]string{
					"apikey": "abababab+01010101",
				},
				encode: []string{"apikey"},
			},
			want:    "https://example.test/api/v1/torrents/000000/torrent?key=abababab%2B01010101",
			wantErr: false,
		},
		{
			name: "passing with vars_2",
			args: args{
				match: "{{ .baseUrl }}/download.php/{{ .torrentId }}/{{ .torrentName }}.torrent?torrent_pass={{ .passkey }}",
				vars: map[string]string{
					"baseUrl":     "https://example.test",
					"torrentId":   "000000",
					"torrentName": "That Movie 2020 Blu-ray 1080p REMUX AVC DTS-HD MA 7 1 GROUP",
				},
				extraVars: map[string]string{
					"passkey": "abababab01010101",
				},
				encode: []string{"torrentName"},
			},
			want:    "https://example.test/download.php/000000/That+Movie+2020+Blu-ray+1080p+REMUX+AVC+DTS-HD+MA+7+1+GROUP.torrent?torrent_pass=abababab01010101",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &announceProcessor{}
			got, err := a.processTorrentUrl(tt.args.match, tt.args.vars, tt.args.extraVars, tt.args.encode)
			if (err != nil) != tt.wantErr {
				t.Errorf("processTorrentUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
