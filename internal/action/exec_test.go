// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package action

import (
	"context"
	"testing"

	"github.com/autobrr/autobrr/internal/domain"
	"github.com/autobrr/autobrr/internal/logger"

	"github.com/stretchr/testify/assert"
)

func Test_service_parseMacros(t *testing.T) {
	t.Parallel()
	type args struct {
		release domain.Release
		action  *domain.Action
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "test_1",
			args: args{
				release: domain.Release{TorrentName: "Sally Goes to the Mall S04E29"},
				action: &domain.Action{
					ExecArgs: `echo "{{ .TorrentName }}"`,
				},
			},
			want:    `echo "Sally Goes to the Mall S04E29"`,
			wantErr: false,
		},
		{
			name: "test_2",
			args: args{
				release: domain.Release{TorrentName: "Sally Goes to the Mall S04E29"},
				action: &domain.Action{
					ExecArgs: `"{{ .TorrentName }}"`,
				},
			},
			want:    `"Sally Goes to the Mall S04E29"`,
			wantErr: false,
		},
		{
			name: "test_3",
			args: args{
				release: domain.Release{TorrentName: "Sally Goes to the Mall S04E29"},
				action: &domain.Action{
					ExecArgs: `--header "Content-Type: application/json" --request POST --data '{"release":"{{ .TorrentName }}"}' http://localhost:3000/api/release`,
				},
			},
			want:    `--header "Content-Type: application/json" --request POST --data '{"release":"Sally Goes to the Mall S04E29"}' http://localhost:3000/api/release`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.args.action.ParseMacros(&tt.args.release)
			assert.Equalf(t, tt.want, tt.args.action.ExecArgs, "parseMacros(%v, %v)", tt.args.action, tt.args.release)
		})
	}
}

func Test_service_execCmd(t *testing.T) {
	t.Parallel()
	type args struct {
		release domain.Release
		action  *domain.Action
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
					Indexer: domain.IndexerMinimal{
						ID:                 0,
						Name:               "Mock Indexer",
						Identifier:         "mock",
						IdentifierExternal: "Mock Indexer",
					},
				},
				action: &domain.Action{
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
			s.execCmd(context.TODO(), tt.args.action, tt.args.release)
		})
	}
}
