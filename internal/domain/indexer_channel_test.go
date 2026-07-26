// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

package domain

import "testing"

// TestIndexerIRCV2_GetChannel_CaseInsensitive is a regression test for the
// "announce: no channel found for name" log spam: many indexer definitions use
// mixed-case channel names (e.g. "#Announce"), while the IRC/announce layers
// canonicalize to lowercase. Prepare() must key ChannelsMap by lowercase and
// GetChannel must resolve regardless of casing.
func TestIndexerIRCV2_GetChannel_CaseInsensitive(t *testing.T) {
	def := &IndexerDefinition{
		Implementation: IndexerImplementationIRC,
		IRC: &IndexerIRCV2{
			Channels: []IndexerIRCV2Channel{
				{Name: "#Announce", Parse: &IndexerIRCV2Parse{}},
			},
		},
	}

	def.Prepare()

	tests := []struct {
		name   string
		lookup string
		want   bool
	}{
		{name: "lowercased (as the IRC/announce layers pass it)", lookup: "#announce", want: true},
		{name: "original case", lookup: "#Announce", want: true},
		{name: "mixed case", lookup: "#AnNoUnCe", want: true},
		{name: "unknown channel", lookup: "#other", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := def.IRC.GetChannel(tt.lookup); ok != tt.want {
				t.Fatalf("GetChannel(%q) ok = %v, want %v", tt.lookup, ok, tt.want)
			}
		})
	}
}
