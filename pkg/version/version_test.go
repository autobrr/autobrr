package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitHubReleaseChecker_checkNewVersion(t *testing.T) {
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
					TargetCommitish: nil,
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
					TargetCommitish: nil,
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
					TargetCommitish: nil,
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
					TargetCommitish: nil,
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
					TargetCommitish: nil,
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
