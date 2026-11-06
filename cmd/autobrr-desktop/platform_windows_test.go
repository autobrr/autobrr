// Copyright (c) 2021 - 2025, Ludvig Lundgren and the autobrr contributors.
// SPDX-License-Identifier: GPL-2.0-or-later

//go:build windows

package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// A named mutex reports ERROR_ALREADY_EXISTS to its own process too, so a
// second lockInstance call stands in for a second instance.
func TestLockInstance(t *testing.T) {
	dir := t.TempDir()

	locked, err := lockInstance(dir)
	if err != nil || !locked {
		t.Fatalf("first lock: locked=%v err=%v", locked, err)
	}

	for _, same := range []string{dir, strings.ToUpper(dir), dir + `\`, filepath.Join(dir, "sub", "..")} {
		locked, err := lockInstance(same)
		if err != nil || locked {
			t.Errorf("lockInstance(%q): locked=%v err=%v, want already running", same, locked, err)
		}
	}

	locked, err = lockInstance(t.TempDir())
	if err != nil || !locked {
		t.Errorf("other config dir: locked=%v err=%v, want its own lock", locked, err)
	}
}
