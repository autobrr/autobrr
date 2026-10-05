// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package meta

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetUserAgent(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{name: "release", version: "v1.87.0", want: "v1.87.0"},
		{name: "no_v_prefix", version: "1.87.0", want: "v1.87.0"},
		{name: "pr_build", version: "pr-1234", want: "pr-1234"},
		{name: "develop_build", version: "develop", want: "develop"},
		{name: "spaces_and_parens", version: "1.87.0 (custom)", want: "v1.87.0custom"},
		{name: "slash", version: "feature/foo", want: "featurefoo"},
		{name: "all_invalid", version: "()", want: devVersion},
		{name: "dev", version: devVersion, want: devVersion},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := version
			t.Cleanup(func() { version = old })
			version = tt.version

			assert.Equal(t, "autobrr/"+tt.want+" ("+runtime.GOOS+"/"+runtime.GOARCH+")", GetUserAgent())
		})
	}
}
