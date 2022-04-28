package domain

import (
	"testing"
)

func TestParseReleaseTags(t *testing.T) {
	type args struct {
		tags []string
	}
	tests := []struct {
		name string
		args args
	}{
		{name: "test_1", args: args{tags: []string{"CD", "FLAC", "Lossless"}}},
		{name: "test_2", args: args{tags: []string{"MP4", "2160p", "BluRay"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ParseReleaseTags(tt.args.tags)
		})
	}
}

func TestParseReleaseTagString(t *testing.T) {
	type args struct {
		tags string
	}
	tests := []struct {
		name string
		args args
		//want ReleaseTags
	}{
		// TODO: Add test cases.
		{name: "test_1", args: args{tags: "FLAC / Lossless / Log / 80% / Cue / CD"}},
		{name: "test_2", args: args{tags: "FLAC Lossless Log 80% Cue CD"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ParseReleaseTagString(tt.args.tags)
			//assert.Equalf(t, tt.want, ParseReleaseTagString(tt.args.tags), "ParseReleaseTagString(%v)", tt.args.tags)
		})
	}
}
