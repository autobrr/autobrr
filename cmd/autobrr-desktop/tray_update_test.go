// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows || darwin || linux

package main

import (
	"slices"
	"testing"
	"time"

	pkgversion "github.com/autobrr/autobrr/pkg/version"
)

func TestAutoUpdateDue(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		published time.Time
		want      bool
	}{
		{name: "just released", published: now.Add(-time.Hour), want: false},
		{name: "one minute short", published: now.Add(-24*time.Hour + time.Minute), want: false},
		{name: "exactly 24 hours", published: now.Add(-24 * time.Hour), want: true},
		{name: "older", published: now.Add(-72 * time.Hour), want: true},
		{name: "no publish date", published: time.Time{}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rel := &pkgversion.Release{TagName: "v1.91.0", PublishedAt: tt.published}
			if got := autoUpdateDue(rel, now); got != tt.want {
				t.Fatalf("autoUpdateDue = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRestartArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{name: "no args", args: nil, want: []string{"--updated-from=1.90.0"}},
		{name: "keeps config", args: []string{"--config", `C:\autobrr`}, want: []string{"--config", `C:\autobrr`, "--updated-from=1.90.0"}},
		{name: "replaces earlier update", args: []string{"--config=x", "--updated-from=1.89.0"}, want: []string{"--config=x", "--updated-from=1.90.0"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := restartArgs(tt.args, "1.90.0"); !slices.Equal(got, tt.want) {
				t.Fatalf("restartArgs = %q, want %q", got, tt.want)
			}
		})
	}
}
