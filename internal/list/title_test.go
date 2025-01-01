// Copyright (c) 2021 - 2024, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_processTitle(t *testing.T) {
	type args struct {
		title        string
		matchRelease bool
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "test_1",
			args: args{
				title:        "Sally Goes to the Mall",
				matchRelease: true,
			},
			want: []string{"*Sally?Goes?to?the?Mall*"},
		},
		{
			name: "test_2",
			args: args{
				title:        "*****… (los asteriscos…)",
				matchRelease: false,
			},
			want: []string{"*los?asteriscos*", "*los?asteriscos"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := processTitle(tt.args.title, tt.args.matchRelease)

			// order seem to be random so lets check if the elements are what we expect
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}
