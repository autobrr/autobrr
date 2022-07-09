package action

import (
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/stretchr/testify/assert"
)

func Test_service_parseExecArgs(t *testing.T) {
	type args struct {
		release  domain.Release
		execArgs string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "test_1",
			args: args{
				release:  domain.Release{TorrentName: "Sally Goes to the Mall S04E29"},
				execArgs: `echo "{{ .TorrentName }}"`,
			},
			want: []string{
				"echo",
				"Sally Goes to the Mall S04E29",
			},
			wantErr: false,
		},
		{
			name: "test_2",
			args: args{
				release:  domain.Release{TorrentName: "Sally Goes to the Mall S04E29"},
				execArgs: `"{{ .TorrentName }}"`,
			},
			want: []string{
				"Sally Goes to the Mall S04E29",
			},
			wantErr: false,
		},
		{
			name: "test_3",
			args: args{
				release:  domain.Release{TorrentName: "Sally Goes to the Mall S04E29"},
				execArgs: `--header "Content-Type: application/json" --request POST --data '{"release":"{{ .TorrentName }}"}' http://localhost:3000/api/release`,
			},
			want: []string{
				"--header",
				"Content-Type: application/json",
				"--request",
				"POST",
				"--data",
				`{"release":"Sally Goes to the Mall S04E29"}`,
				"http://localhost:3000/api/release",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				log:       logger.Mock().With().Logger(),
				repo:      nil,
				clientSvc: nil,
				bus:       nil,
			}
			got, _ := s.parseExecArgs(tt.args.release, tt.args.execArgs)
			assert.Equalf(t, tt.want, got, "parseExecArgs(%v, %v)", tt.args.release, tt.args.execArgs)
		})
	}
}

func Test_service_execCmd(t *testing.T) {
	type args struct {
		release domain.Release
		action  domain.Action
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test_1",
			args: args{
				release: domain.Release{
					TorrentName:    "This is a test",
					TorrentTmpFile: "tmp-10000",
					Indexer:        "mock",
				},
				action: domain.Action{
					Name:     "echo",
					ExecCmd:  "echo",
					ExecArgs: "hello",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &service{
				log:       logger.Mock().With().Logger(),
				repo:      nil,
				clientSvc: nil,
				bus:       nil,
			}
			s.execCmd(tt.args.action, tt.args.release)
		})
	}
}
