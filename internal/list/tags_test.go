// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package list

import (
	"testing"

	"github.com/autobrr/autobrr/pkg/arr"

	"github.com/magiconair/properties/assert"
)

func Test_containsTag(t *testing.T) {
	type args struct {
		tags      []*arr.Tag
		titleTags []int
		checkTags []string
	}

	tags := []*arr.Tag{
		{
			ID:    1,
			Label: "Want",
		},
		{
			ID:    2,
			Label: "exclude-me",
		},
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "test_1",
			args: args{
				tags:      tags,
				titleTags: []int{},
				checkTags: []string{"Want"},
			},
			want: false,
		},
		{
			name: "test_2",
			args: args{
				tags:      tags,
				titleTags: []int{1},
				checkTags: []string{"Want"},
			},
			want: true,
		},
		{
			name: "test_3",
			args: args{
				tags:      tags,
				titleTags: []int{1},
				checkTags: nil,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, containsTag(tt.args.tags, tt.args.titleTags, tt.args.checkTags))
		})
	}
}
