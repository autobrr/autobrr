// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitHubReleaseChecker_checkNewVersion(t *testing.T) {
	t.Parallel()
	type fields struct {
		Repo string
	}
	type args struct {
		version string
		release *Release
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantNew     bool
		wantVersion string
		wantErr     bool
	}{
		{
			name:   "outdated new available",
			fields: fields{},
			args: args{
				version: "v0.2.0",
				release: &Release{
					TagName:         "v0.3.0",
					TargetCommitish: "",
				},
			},
			wantNew:     true,
			wantVersion: "0.3.0",
			wantErr:     false,
		},
		{
			name:   "same version",
			fields: fields{},
			args: args{
				version: "v0.2.0",
				release: &Release{
					TagName:         "v0.2.0",
					TargetCommitish: "",
				},
			},
			wantNew:     false,
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:   "no new version",
			fields: fields{},
			args: args{
				version: "v0.3.0",
				release: &Release{
					TagName:         "v0.2.0",
					TargetCommitish: "",
				},
			},
			wantNew:     false,
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:   "new rc available",
			fields: fields{},
			args: args{
				version: "v0.3.0",
				release: &Release{
					TagName:         "v0.3.0-rc1",
					TargetCommitish: "",
				},
			},
			wantNew:     false,
			wantVersion: "",
			wantErr:     false,
		},
		{
			name:   "new rc available",
			fields: fields{},
			args: args{
				version: "v0.3.0-RC1",
				release: &Release{
					TagName:         "v0.3.0-RC2",
					TargetCommitish: "",
				},
			},
			wantNew:     true,
			wantVersion: "0.3.0-RC2",
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &Checker{
				Repo: tt.fields.Repo,
			}
			got, gotVersion, err := g.checkNewVersion(tt.args.version, tt.args.release)
			if tt.wantErr && assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err)
			}
			assert.Equal(t, tt.wantNew, got)
			assert.Equal(t, tt.wantVersion, gotVersion)
		})
	}
}

func Test_isDevelop(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		version string
		want    bool
	}{
		{name: "test_1", version: "dev", want: true},
		{name: "test_2", version: "develop", want: true},
		{name: "test_3", version: "master", want: true},
		{name: "test_4", version: "latest", want: true},
		{name: "test_5", version: "v1.0.1", want: false},
		{name: "test_6", version: "1.0.1", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, isDevelop(tt.version), "isDevelop(%v)", tt.version)
		})
	}
}
