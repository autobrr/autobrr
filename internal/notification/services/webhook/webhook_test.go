// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseHeaders(t *testing.T) {
	t.Parallel()

	type args struct {
		headers string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "empty",
			args: args{headers: ""},
			want: nil,
		},
		{
			name: "single",
			args: args{headers: "Authorization=Bearer mock-key"},
			want: map[string]string{"Authorization": "Bearer mock-key"},
		},
		{
			name: "multiple",
			args: args{headers: "Authorization=Bearer mock-key, X-Custom-Header=custom-value"},
			want: map[string]string{"Authorization": "Bearer mock-key", "X-Custom-Header": "custom-value"},
		},
		{
			name: "value with equals",
			args: args{headers: "X-Signature=key=value"},
			want: map[string]string{"X-Signature": "key=value"},
		},
		{
			name: "missing value",
			args: args{headers: "X-Broken, X-Custom-Header=custom-value"},
			want: map[string]string{"X-Custom-Header": "custom-value"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, parseHeaders(tt.args.headers), "parseHeaders(%v)", tt.args.headers)
		})
	}
}
