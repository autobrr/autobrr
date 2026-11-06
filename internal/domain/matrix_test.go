// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeMatrixHomeserver(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "matrix.watchtower.pm", want: "https://matrix.watchtower.pm"},
		{in: "https://Matrix.Watchtower.pm/", want: "https://matrix.watchtower.pm"},
		{in: "http://localhost:8008", want: "http://localhost:8008"},
		{in: "https://matrix.example.org/base/", want: "https://matrix.example.org/base"},
		{in: "", wantErr: true},
		{in: "ftp://matrix.example.org", wantErr: true},
		{in: "https://user:pass@matrix.example.org", wantErr: true},
		{in: "https://matrix.example.org?token=1", wantErr: true},
		{in: "https://matrix.example.org#frag", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := NormalizeMatrixHomeserver(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateMatrixRequiresParseLines(t *testing.T) {
	def := IndexerDefinition{Matrix: &IndexerMatrix{
		Homeserver: "https://matrix.example.org",
		Rooms: []IndexerIRCV2Channel{{
			Name:       "#announce:matrix.example.org",
			Announcers: []string{"@bot:matrix.example.org"},
			Parse:      &IndexerIRCV2Parse{Type: "single"},
		}},
	}}

	assert.ErrorContains(t, def.ValidateMatrix(), "no parse lines")

	def.Matrix.Rooms[0].Parse.Lines = []IndexerIRCParseLine{{Pattern: ".+"}}
	assert.NoError(t, def.ValidateMatrix())
}
